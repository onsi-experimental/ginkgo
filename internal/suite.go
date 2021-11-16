package internal

import (
	"fmt"
	"time"

	"github.com/onsi-experimental/ginkgo/v2/formatter"
	"github.com/onsi-experimental/ginkgo/v2/internal/interrupt_handler"
	"github.com/onsi-experimental/ginkgo/v2/internal/parallel_support"
	"github.com/onsi-experimental/ginkgo/v2/reporters"
	"github.com/onsi-experimental/ginkgo/v2/types"
)

type Phase uint

const (
	PhaseBuildTopLevel Phase = iota
	PhaseBuildTree
	PhaseRun
)

type Suite struct {
	tree               *TreeNode
	topLevelContainers Nodes

	phase Phase

	suiteNodes   Nodes
	cleanupNodes Nodes

	failer            *Failer
	reporter          reporters.Reporter
	writer            WriterInterface
	outputInterceptor OutputInterceptor
	interruptHandler  interrupt_handler.InterruptHandlerInterface
	config            types.SuiteConfig

	skipAll           bool
	report            types.Report
	currentSpecReport types.SpecReport
	currentNode       Node

	client parallel_support.Client
}

func NewSuite() *Suite {
	return &Suite{
		tree:  &TreeNode{},
		phase: PhaseBuildTopLevel,
	}
}

func (suite *Suite) BuildTree() error {
	// During PhaseBuildTopLevel, the top level containers are stored in suite.topLevelCotainers and entered
	// We now enter PhaseBuildTree where these top level containers are entered and added to the spec tree
	suite.phase = PhaseBuildTree
	for _, topLevelContainer := range suite.topLevelContainers {
		err := suite.PushNode(topLevelContainer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (suite *Suite) Run(description string, suitePath string, failer *Failer, reporter reporters.Reporter, writer WriterInterface, outputInterceptor OutputInterceptor, interruptHandler interrupt_handler.InterruptHandlerInterface, client parallel_support.Client, suiteConfig types.SuiteConfig) (bool, bool) {
	if suite.phase != PhaseBuildTree {
		panic("cannot run before building the tree = call suite.BuildTree() first")
	}
	ApplyNestedFocusPolicyToTree(suite.tree)
	specs := GenerateSpecsFromTreeRoot(suite.tree)
	specs, hasProgrammaticFocus := ApplyFocusToSpecs(specs, description, suiteConfig)

	suite.phase = PhaseRun
	suite.client = client
	suite.failer = failer
	suite.reporter = reporter
	suite.writer = writer
	suite.outputInterceptor = outputInterceptor
	suite.interruptHandler = interruptHandler
	suite.config = suiteConfig

	success := suite.runSpecs(description, suitePath, hasProgrammaticFocus, specs)

	return success, hasProgrammaticFocus
}

/*
  Tree Construction methods

  PushNode is used during PhaseBuildTopLevel and PhaseBuildTree
*/

func (suite *Suite) PushNode(node Node) error {
	if node.NodeType.Is(types.NodeTypeCleanupInvalid | types.NodeTypeCleanupAfterEach | types.NodeTypeCleanupAfterAll | types.NodeTypeCleanupAfterSuite) {
		return suite.pushCleanupNode(node)
	}

	if node.NodeType.Is(types.NodeTypeBeforeSuite | types.NodeTypeAfterSuite | types.NodeTypeSynchronizedBeforeSuite | types.NodeTypeSynchronizedAfterSuite | types.NodeTypeReportAfterSuite) {
		return suite.pushSuiteNode(node)
	}

	if suite.phase == PhaseRun {
		return types.GinkgoErrors.PushingNodeInRunPhase(node.NodeType, node.CodeLocation)
	}

	if node.MarkedSerial {
		firstOrderedNode := suite.tree.AncestorNodeChain().FirstNodeMarkedOrdered()
		if !firstOrderedNode.IsZero() && !firstOrderedNode.MarkedSerial {
			return types.GinkgoErrors.InvalidSerialNodeInNonSerialOrderedContainer(node.CodeLocation, node.NodeType)
		}
	}

	if node.NodeType.Is(types.NodeTypeBeforeAll | types.NodeTypeAfterAll) {
		firstOrderedNode := suite.tree.AncestorNodeChain().FirstNodeMarkedOrdered()
		if firstOrderedNode.IsZero() {
			return types.GinkgoErrors.SetupNodeNotInOrderedContainer(node.CodeLocation, node.NodeType)
		}
	}

	if node.NodeType == types.NodeTypeContainer {
		// During PhaseBuildTopLevel we only track the top level containers without entering them
		// We only enter the top level container nodes during PhaseBuildTree
		//
		// This ensures the tree is only constructed after `go spec` has called `flag.Parse()` and gives
		// the user an opportunity to load suiteConfiguration information in the `TestX` go spec hook just before `RunSpecs`
		// is invoked.  This makes the lifecycle easier to reason about and solves issues like #693.
		if suite.phase == PhaseBuildTopLevel {
			suite.topLevelContainers = append(suite.topLevelContainers, node)
			return nil
		}
		if suite.phase == PhaseBuildTree {
			parentTree := suite.tree
			suite.tree = &TreeNode{Node: node}
			parentTree.AppendChild(suite.tree)
			err := func() (err error) {
				defer func() {
					if e := recover(); e != nil {
						err = types.GinkgoErrors.CaughtPanicDuringABuildPhase(e, node.CodeLocation)
					}
				}()
				node.Body()
				return err
			}()
			suite.tree = parentTree
			return err
		}
	} else {
		suite.tree.AppendChild(&TreeNode{Node: node})
		return nil
	}

	return nil
}

func (suite *Suite) pushSuiteNode(node Node) error {
	if suite.phase == PhaseBuildTree {
		return types.GinkgoErrors.SuiteNodeInNestedContext(node.NodeType, node.CodeLocation)
	}

	if suite.phase == PhaseRun {
		return types.GinkgoErrors.SuiteNodeDuringRunPhase(node.NodeType, node.CodeLocation)
	}

	switch node.NodeType {
	case types.NodeTypeBeforeSuite, types.NodeTypeSynchronizedBeforeSuite:
		existingBefores := suite.suiteNodes.WithType(types.NodeTypeBeforeSuite | types.NodeTypeSynchronizedBeforeSuite)
		if len(existingBefores) > 0 {
			return types.GinkgoErrors.MultipleBeforeSuiteNodes(node.NodeType, node.CodeLocation, existingBefores[0].NodeType, existingBefores[0].CodeLocation)
		}
	case types.NodeTypeAfterSuite, types.NodeTypeSynchronizedAfterSuite:
		existingAfters := suite.suiteNodes.WithType(types.NodeTypeAfterSuite | types.NodeTypeSynchronizedAfterSuite)
		if len(existingAfters) > 0 {
			return types.GinkgoErrors.MultipleAfterSuiteNodes(node.NodeType, node.CodeLocation, existingAfters[0].NodeType, existingAfters[0].CodeLocation)
		}
	}

	suite.suiteNodes = append(suite.suiteNodes, node)
	return nil
}

func (suite *Suite) pushCleanupNode(node Node) error {
	if suite.phase != PhaseRun || suite.currentNode.IsZero() {
		return types.GinkgoErrors.PushingCleanupNodeDuringTreeConstruction(node.CodeLocation)
	}

	switch suite.currentNode.NodeType {
	case types.NodeTypeBeforeSuite, types.NodeTypeSynchronizedBeforeSuite, types.NodeTypeAfterSuite, types.NodeTypeSynchronizedAfterSuite:
		node.NodeType = types.NodeTypeCleanupAfterSuite
	case types.NodeTypeBeforeAll, types.NodeTypeAfterAll:
		node.NodeType = types.NodeTypeCleanupAfterAll
	case types.NodeTypeReportBeforeEach, types.NodeTypeReportAfterEach, types.NodeTypeReportAfterSuite:
		return types.GinkgoErrors.PushingCleanupInReportingNode(node.CodeLocation, suite.currentNode.NodeType)
	case types.NodeTypeCleanupInvalid, types.NodeTypeCleanupAfterEach, types.NodeTypeCleanupAfterAll, types.NodeTypeCleanupAfterSuite:
		return types.GinkgoErrors.PushingCleanupInCleanupNode(node.CodeLocation)
	default:
		node.NodeType = types.NodeTypeCleanupAfterEach
	}

	node.NodeIDWhereCleanupWasGenerated = suite.currentNode.ID
	node.NestingLevel = suite.currentNode.NestingLevel
	suite.cleanupNodes = append(suite.cleanupNodes, node)

	return nil
}

/*
  Spec Running methods - used during PhaseRun
*/
func (suite *Suite) CurrentSpecReport() types.SpecReport {
	report := suite.currentSpecReport
	if suite.writer != nil {
		report.CapturedGinkgoWriterOutput = string(suite.writer.Bytes())
	}
	return report
}

func (suite *Suite) AddReportEntry(entry ReportEntry) error {
	if suite.phase != PhaseRun {
		return types.GinkgoErrors.AddReportEntryNotDuringRunPhase(entry.Location)
	}
	suite.currentSpecReport.ReportEntries = append(suite.currentSpecReport.ReportEntries, entry)
	return nil
}

func (suite *Suite) isRunningInParallel() bool {
	return suite.config.ParallelTotal > 1
}

func (suite *Suite) processCurrentSpecReport() {
	suite.reporter.DidRun(suite.currentSpecReport)
	if suite.isRunningInParallel() {
		suite.client.PostDidRun(suite.currentSpecReport)
	}
	suite.report.SpecReports = append(suite.report.SpecReports, suite.currentSpecReport)

	if suite.currentSpecReport.State.Is(types.SpecStateFailureStates) {
		suite.report.SuiteSucceeded = false
		if suite.config.FailFast || suite.currentSpecReport.State.Is(types.SpecStateAborted) {
			suite.skipAll = true
			if suite.isRunningInParallel() {
				suite.client.PostAbort()
			}
		}
	}
}

func (suite *Suite) runSpecs(description string, suitePath string, hasProgrammaticFocus bool, specs Specs) bool {
	numSpecsThatWillBeRun := specs.CountWithoutSkip()

	suite.report = types.Report{
		SuitePath:                 suitePath,
		SuiteDescription:          description,
		SuiteConfig:               suite.config,
		SuiteHasProgrammaticFocus: hasProgrammaticFocus,
		PreRunStats: types.PreRunStats{
			TotalSpecs:       len(specs),
			SpecsThatWillRun: numSpecsThatWillBeRun,
		},
		StartTime: time.Now(),
	}

	suite.reporter.SuiteWillBegin(suite.report)
	if suite.isRunningInParallel() {
		suite.client.PostSuiteWillBegin(suite.report)
	}

	suite.report.SuiteSucceeded = true
	suite.runBeforeSuite(numSpecsThatWillBeRun)

	if suite.report.SuiteSucceeded {
		groupedSpecIndices, serialGroupedSpecIndices := OrderSpecs(specs, suite.config)
		nextIndex := MakeIncrementingIndexCounter()
		if suite.isRunningInParallel() {
			nextIndex = suite.client.FetchNextCounter
		}

		for {
			groupedSpecIdx, err := nextIndex()
			if err != nil {
				suite.report.SpecialSuiteFailureReasons = append(suite.report.SpecialSuiteFailureReasons, fmt.Sprintf("Failed to iterate over specs:\n%s", err.Error()))
				suite.report.SuiteSucceeded = false
				break
			}

			if groupedSpecIdx >= len(groupedSpecIndices) {
				if suite.config.ParallelProcess == 1 && len(serialGroupedSpecIndices) > 0 {
					groupedSpecIndices, serialGroupedSpecIndices, nextIndex = serialGroupedSpecIndices, GroupedSpecIndices{}, MakeIncrementingIndexCounter()
					suite.client.BlockUntilNonprimaryProcsHaveFinished()
					continue
				}
				break
			}

			suite.runGroup(specs.AtIndices(groupedSpecIndices[groupedSpecIdx]))
		}

		if specs.HasAnySpecsMarkedPending() && suite.config.FailOnPending {
			suite.report.SpecialSuiteFailureReasons = append(suite.report.SpecialSuiteFailureReasons, "Detected pending specs and --fail-on-pending is set")
			suite.report.SuiteSucceeded = false
		}
	}

	suite.runAfterSuiteCleanup(numSpecsThatWillBeRun)

	interruptStatus := suite.interruptHandler.Status()
	if interruptStatus.Interrupted {
		suite.report.SpecialSuiteFailureReasons = append(suite.report.SpecialSuiteFailureReasons, interruptStatus.Cause.String())
		suite.report.SuiteSucceeded = false
	}
	suite.report.EndTime = time.Now()
	suite.report.RunTime = suite.report.EndTime.Sub(suite.report.StartTime)

	if suite.config.ParallelProcess == 1 {
		suite.runReportAfterSuite()
	}
	suite.reporter.SuiteDidEnd(suite.report)
	if suite.isRunningInParallel() {
		suite.client.PostSuiteDidEnd(suite.report)
	}

	return suite.report.SuiteSucceeded
}

func (suite *Suite) runBeforeSuite(numSpecsThatWillBeRun int) {
	interruptStatus := suite.interruptHandler.Status()
	beforeSuiteNode := suite.suiteNodes.FirstNodeWithType(types.NodeTypeBeforeSuite | types.NodeTypeSynchronizedBeforeSuite)
	if !beforeSuiteNode.IsZero() && !interruptStatus.Interrupted && numSpecsThatWillBeRun > 0 {
		suite.currentSpecReport = types.SpecReport{
			LeafNodeType:     beforeSuiteNode.NodeType,
			LeafNodeLocation: beforeSuiteNode.CodeLocation,
			ParallelProcess:  suite.config.ParallelProcess,
		}
		suite.reporter.WillRun(suite.currentSpecReport)
		suite.runSuiteNode(beforeSuiteNode, interruptStatus.Channel)
		if suite.currentSpecReport.State.Is(types.SpecStateSkipped) {
			suite.report.SpecialSuiteFailureReasons = append(suite.report.SpecialSuiteFailureReasons, "Suite skipped in BeforeSuite")
			suite.skipAll = true
		}
		suite.processCurrentSpecReport()
	}
}

func (suite *Suite) runAfterSuiteCleanup(numSpecsThatWillBeRun int) {
	afterSuiteNode := suite.suiteNodes.FirstNodeWithType(types.NodeTypeAfterSuite | types.NodeTypeSynchronizedAfterSuite)
	if !afterSuiteNode.IsZero() && numSpecsThatWillBeRun > 0 {
		suite.currentSpecReport = types.SpecReport{
			LeafNodeType:     afterSuiteNode.NodeType,
			LeafNodeLocation: afterSuiteNode.CodeLocation,
			ParallelProcess:  suite.config.ParallelProcess,
		}
		suite.reporter.WillRun(suite.currentSpecReport)
		suite.runSuiteNode(afterSuiteNode, suite.interruptHandler.Status().Channel)
		suite.processCurrentSpecReport()
	}

	afterSuiteCleanup := suite.cleanupNodes.WithType(types.NodeTypeCleanupAfterSuite).Reverse()
	if len(afterSuiteCleanup) > 0 {
		for _, cleanupNode := range afterSuiteCleanup {
			suite.currentSpecReport = types.SpecReport{
				LeafNodeType:     cleanupNode.NodeType,
				LeafNodeLocation: cleanupNode.CodeLocation,
				ParallelProcess:  suite.config.ParallelProcess,
			}
			suite.reporter.WillRun(suite.currentSpecReport)
			suite.runSuiteNode(cleanupNode, suite.interruptHandler.Status().Channel)
			suite.processCurrentSpecReport()
		}
	}
}

func (suite *Suite) runReportAfterSuite() {
	for _, node := range suite.suiteNodes.WithType(types.NodeTypeReportAfterSuite) {
		suite.currentSpecReport = types.SpecReport{
			LeafNodeType:     node.NodeType,
			LeafNodeLocation: node.CodeLocation,
			LeafNodeText:     node.Text,
			ParallelProcess:  suite.config.ParallelProcess,
		}
		suite.reporter.WillRun(suite.currentSpecReport)
		suite.runReportAfterSuiteNode(node, suite.report)
		suite.processCurrentSpecReport()
	}
}

func (suite *Suite) runGroup(specs Specs) {
	nodeState := map[uint]types.SpecState{}
	groupSucceeded := true

	indexOfLastSpecContainingNodeID := func(id uint) int {
		lastIdx := -1
		for idx := range specs {
			if specs[idx].Nodes.ContainsNodeID(id) && !specs[idx].Skip {
				lastIdx = idx
			}
		}
		return lastIdx
	}

	for i, spec := range specs {
		suite.currentSpecReport = types.SpecReport{
			ContainerHierarchyTexts:     spec.Nodes.WithType(types.NodeTypeContainer).Texts(),
			ContainerHierarchyLocations: spec.Nodes.WithType(types.NodeTypeContainer).CodeLocations(),
			ContainerHierarchyLabels:    spec.Nodes.WithType(types.NodeTypeContainer).Labels(),
			LeafNodeLocation:            spec.FirstNodeWithType(types.NodeTypeIt).CodeLocation,
			LeafNodeType:                types.NodeTypeIt,
			LeafNodeText:                spec.FirstNodeWithType(types.NodeTypeIt).Text,
			LeafNodeLabels:              []string(spec.FirstNodeWithType(types.NodeTypeIt).Labels),
			ParallelProcess:             suite.config.ParallelProcess,
			IsSerial:                    spec.Nodes.HasNodeMarkedSerial(),
			IsInOrderedContainer:        !spec.Nodes.FirstNodeMarkedOrdered().IsZero(),
		}

		skip := spec.Skip
		if spec.Nodes.HasNodeMarkedPending() {
			skip = true
			suite.currentSpecReport.State = types.SpecStatePending
		} else {
			if suite.interruptHandler.Status().Interrupted || suite.skipAll {
				skip = true
			}
			if !groupSucceeded {
				skip = true
				suite.currentSpecReport.Failure = suite.failureForLeafNodeWithMessage(spec.FirstNodeWithType(types.NodeTypeIt),
					"Spec skipped because an earlier spec in an ordered container failed")
			}
			for _, node := range spec.Nodes.WithType(types.NodeTypeBeforeAll) {
				if nodeState[node.ID] == types.SpecStateSkipped {
					skip = true
					suite.currentSpecReport.Failure = suite.failureForLeafNodeWithMessage(spec.FirstNodeWithType(types.NodeTypeIt),
						"Spec skipped because Skip() was called in BeforeAll")
					break
				}
			}
			if skip {
				suite.currentSpecReport.State = types.SpecStateSkipped
			}
		}

		if suite.config.DryRun && !skip {
			skip = true
			suite.currentSpecReport.State = types.SpecStatePassed
		}

		suite.reporter.WillRun(suite.currentSpecReport)
		//send the spec report to any attached ReportBeforeEach blocks - this will update suite.currentSpecReport if failures occur in these blocks
		suite.reportEach(spec, types.NodeTypeReportBeforeEach)
		if suite.currentSpecReport.State.Is(types.SpecStateFailureStates) {
			//the reportEach failed, skip this spec
			skip = true
		}

		suite.currentSpecReport.StartTime = time.Now()
		maxAttempts := max(1, spec.FlakeAttempts())
		if suite.config.FlakeAttempts > 0 {
			maxAttempts = suite.config.FlakeAttempts
		}

		for attempt := 0; !skip && (attempt < maxAttempts); attempt++ {
			suite.currentSpecReport.NumAttempts = attempt + 1
			suite.writer.Truncate()
			suite.outputInterceptor.StartInterceptingOutput()
			if attempt > 0 {
				fmt.Fprintf(suite.writer, "\nGinkgo: Attempt #%d Failed.  Retrying...\n", attempt)
			}
			isFinalAttempt := (attempt == maxAttempts-1)

			interruptStatus := suite.interruptHandler.Status()
			deepestNestingLevelAttained := -1
			var nodes = spec.Nodes.WithType(types.NodeTypeBeforeAll).Filter(func(n Node) bool {
				return nodeState[n.ID] != types.SpecStatePassed
			})
			nodes = nodes.CopyAppend(spec.Nodes.WithType(types.NodeTypeBeforeEach)...).SortedByAscendingNestingLevel()
			nodes = nodes.CopyAppend(spec.Nodes.WithType(types.NodeTypeJustBeforeEach).SortedByAscendingNestingLevel()...)
			nodes = nodes.CopyAppend(spec.Nodes.WithType(types.NodeTypeIt)...)

			var terminatingNode Node
			for j := range nodes {
				deepestNestingLevelAttained = max(deepestNestingLevelAttained, nodes[j].NestingLevel)
				suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(nodes[j], interruptStatus.Channel, spec.Nodes.BestTextFor(nodes[j]))
				suite.currentSpecReport.RunTime = time.Since(suite.currentSpecReport.StartTime)
				nodeState[nodes[j].ID] = suite.currentSpecReport.State
				if suite.currentSpecReport.State != types.SpecStatePassed {
					terminatingNode = nodes[j]
					break
				}
			}

			afterAllNodesThatRan := map[uint]bool{}
			// pull out some shared code so we aren't repeating ourselves down below. this just runs after and cleanup nodes
			runAfterAndCleanupNodes := func(nodes Nodes) {
				for j := range nodes {
					state, failure := suite.runNode(nodes[j], suite.interruptHandler.Status().Channel, spec.Nodes.BestTextFor(nodes[j]))
					suite.currentSpecReport.RunTime = time.Since(suite.currentSpecReport.StartTime)
					nodeState[nodes[j].ID] = state
					if suite.currentSpecReport.State == types.SpecStatePassed || state == types.SpecStateAborted {
						suite.currentSpecReport.State = state
						suite.currentSpecReport.Failure = failure
						if state != types.SpecStatePassed {
							terminatingNode = nodes[j]
						}
					}
					if nodes[j].NodeType.Is(types.NodeTypeAfterAll) {
						afterAllNodesThatRan[nodes[j].ID] = true
					}
				}
			}

			// pull out a helper that captures the logic of whether or not we should run a given After node.
			// there is complexity here stemming from the fact that we allow nested ordered contexts and flakey retries
			shouldRunAfterNode := func(n Node) bool {
				if n.NodeType.Is(types.NodeTypeAfterEach | types.NodeTypeJustAfterEach) {
					return true
				}
				var id uint
				if n.NodeType.Is(types.NodeTypeAfterAll) {
					id = n.ID
					if afterAllNodesThatRan[id] { //we've already run on this attempt. don't run again.
						return false
					}
				}
				if n.NodeType.Is(types.NodeTypeCleanupAfterAll) {
					id = n.NodeIDWhereCleanupWasGenerated
				}
				isLastSpecWithNode := indexOfLastSpecContainingNodeID(id) == i

				switch suite.currentSpecReport.State {
				case types.SpecStatePassed: //we've passed so far...
					return isLastSpecWithNode //... and we're the last spec with this AfterNode, so we should run it
				case types.SpecStateSkipped: //the spec was skipped by the user...
					if isLastSpecWithNode {
						return true //...we're the last spec, so we should run the AfterNode
					}
					if terminatingNode.NodeType.Is(types.NodeTypeBeforeAll) && terminatingNode.NestingLevel == n.NestingLevel {
						return true //...or, a BeforeAll was skipped and it's at our nesting level, so our subgroup is going to skip
					}
				case types.SpecStateFailed, types.SpecStatePanicked: // the spec has failed...
					if isFinalAttempt {
						return true //...if this was the last attempt then we're the last spec to run and so the AfterNode should run
					}
					if terminatingNode.NodeType.Is(types.NodeTypeBeforeAll) {
						//...we'll be rerunning a BeforeAll so we should cleanup after it if...
						if n.NodeType.Is(types.NodeTypeAfterAll) && terminatingNode.NestingLevel == n.NestingLevel {
							return true //we're at the same nesting level
						}
						if n.NodeType.Is(types.NodeTypeCleanupAfterAll) && terminatingNode.ID == n.NodeIDWhereCleanupWasGenerated {
							return true //we're a DeferCleanup generated by it
						}
					}
					if terminatingNode.NodeType.Is(types.NodeTypeAfterAll) {
						//...we'll be rerunning an AfterAll so we should cleanup after it if...
						if n.NodeType.Is(types.NodeTypeCleanupAfterAll) && terminatingNode.ID == n.NodeIDWhereCleanupWasGenerated {
							return true //we're a DeferCleanup generated by it
						}
					}
				case types.SpecStateInterrupted, types.SpecStateAborted: // ...we've been interrupted and/or aborted
					return true //...that means the test run is over and we should clean up the stack.  Run the AfterNode
				}
				return false
			}

			// first pass - run all the JustAfterEach, Aftereach, and AfterAlls.  Our shoudlRunAfterNode filter function will clean up the AfterAlls for us.
			afterNodes := spec.Nodes.WithType(types.NodeTypeJustAfterEach).SortedByDescendingNestingLevel()
			afterNodes = afterNodes.CopyAppend(spec.Nodes.WithType(types.NodeTypeAfterEach).CopyAppend(spec.Nodes.WithType(types.NodeTypeAfterAll)...).SortedByDescendingNestingLevel()...)
			afterNodes = afterNodes.WithinNestingLevel(deepestNestingLevelAttained)
			afterNodes = afterNodes.Filter(shouldRunAfterNode)
			runAfterAndCleanupNodes(afterNodes)

			// second-pass perhaps we didn't run the AfterAlls but a state change due to an AfterEach now requires us to run the AfterAlls:
			afterNodes = spec.Nodes.WithType(types.NodeTypeAfterAll).WithinNestingLevel(deepestNestingLevelAttained).Filter(shouldRunAfterNode)
			runAfterAndCleanupNodes(afterNodes)

			// now we run any DeferCleanups
			afterNodes = suite.cleanupNodes.WithType(types.NodeTypeCleanupAfterEach).Reverse()
			afterNodes = append(afterNodes, suite.cleanupNodes.WithType(types.NodeTypeCleanupAfterAll).Filter(shouldRunAfterNode).Reverse()...)
			runAfterAndCleanupNodes(afterNodes)

			// third-pass, perhaps a DeferCleanup failed and now we need to run the AfterAlls.
			afterNodes = spec.Nodes.WithType(types.NodeTypeAfterAll).WithinNestingLevel(deepestNestingLevelAttained).Filter(shouldRunAfterNode)
			runAfterAndCleanupNodes(afterNodes)

			// and finally - running AfterAlls may have generated some new DeferCleanup nodes, let's run them to finish up
			afterNodes = suite.cleanupNodes.WithType(types.NodeTypeCleanupAfterAll).Reverse().Filter(shouldRunAfterNode)
			runAfterAndCleanupNodes(afterNodes)

			suite.currentSpecReport.EndTime = time.Now()
			suite.currentSpecReport.RunTime = suite.currentSpecReport.EndTime.Sub(suite.currentSpecReport.StartTime)
			suite.currentSpecReport.CapturedGinkgoWriterOutput += string(suite.writer.Bytes())
			suite.currentSpecReport.CapturedStdOutErr += suite.outputInterceptor.StopInterceptingAndReturnOutput()

			if suite.currentSpecReport.State.Is(types.SpecStatePassed | types.SpecStateSkipped | types.SpecStateAborted | types.SpecStateInterrupted) {
				break
			}
		}

		//send the spec report to any attached ReportAfterEach blocks - this will update suite.currentSpecReport if failures occur in these blocks
		suite.reportEach(spec, types.NodeTypeReportAfterEach)
		suite.processCurrentSpecReport()
		if suite.currentSpecReport.State.Is(types.SpecStateFailureStates) {
			groupSucceeded = false
		}
		suite.currentSpecReport = types.SpecReport{}
	}
}

func (suite *Suite) reportEach(spec Spec, nodeType types.NodeType) {
	if suite.config.DryRun {
		return
	}

	nodes := spec.Nodes.WithType(nodeType)
	if nodeType == types.NodeTypeReportAfterEach {
		nodes = nodes.SortedByDescendingNestingLevel()
	}
	if nodeType == types.NodeTypeReportBeforeEach {
		nodes = nodes.SortedByAscendingNestingLevel()
	}
	if len(nodes) == 0 {
		return
	}

	for i := range nodes {
		suite.writer.Truncate()
		suite.outputInterceptor.StartInterceptingOutput()
		report := suite.currentSpecReport
		nodes[i].Body = func() {
			nodes[i].ReportEachBody(report)
		}
		suite.interruptHandler.SetInterruptPlaceholderMessage(formatter.Fiw(0, formatter.COLS,
			"{{yellow}}Ginkgo received an interrupt signal but is currently running a %s node.  To avoid an invalid report the %s node will not be interrupted however subsequent tests will be skipped.{{/}}\n\n{{bold}}The running %s node is at:\n%s.{{/}}",
			nodeType, nodeType, nodeType,
			nodes[i].CodeLocation,
		))
		state, failure := suite.runNode(nodes[i], nil, spec.Nodes.BestTextFor(nodes[i]))
		suite.interruptHandler.ClearInterruptPlaceholderMessage()
		// If the spec is not in a failure state (i.e. it's Passed/Skipped/Pending) and the reporter has failed, override the state.
		// Also, if the reporter is every aborted - always override the state to propagate the abort
		if (!suite.currentSpecReport.State.Is(types.SpecStateFailureStates) && state.Is(types.SpecStateFailureStates)) || state.Is(types.SpecStateAborted) {
			suite.currentSpecReport.State = state
			suite.currentSpecReport.Failure = failure
		}
		suite.currentSpecReport.CapturedGinkgoWriterOutput += string(suite.writer.Bytes())
		suite.currentSpecReport.CapturedStdOutErr += suite.outputInterceptor.StopInterceptingAndReturnOutput()
	}
}

func (suite *Suite) runSuiteNode(node Node, interruptChannel chan interface{}) {
	if suite.config.DryRun {
		suite.currentSpecReport.State = types.SpecStatePassed
		return
	}

	suite.writer.Truncate()
	suite.outputInterceptor.StartInterceptingOutput()
	suite.currentSpecReport.StartTime = time.Now()

	var err error
	switch node.NodeType {
	case types.NodeTypeBeforeSuite, types.NodeTypeAfterSuite:
		suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(node, interruptChannel, "")
	case types.NodeTypeCleanupAfterSuite:
		if suite.config.ParallelTotal > 1 && suite.config.ParallelProcess == 1 {
			err = suite.client.BlockUntilNonprimaryProcsHaveFinished()
		}
		if err == nil {
			suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(node, interruptChannel, "")
		}
	case types.NodeTypeSynchronizedBeforeSuite:
		var data []byte
		var runAllProcs bool
		if suite.config.ParallelProcess == 1 {
			if suite.config.ParallelTotal > 1 {
				suite.outputInterceptor.StopInterceptingAndReturnOutput()
				suite.outputInterceptor.StartInterceptingOutputAndForwardTo(suite.client)
			}
			node.Body = func() { data = node.SynchronizedBeforeSuiteProc1Body() }
			suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(node, interruptChannel, "")
			if suite.config.ParallelTotal > 1 {
				suite.currentSpecReport.CapturedStdOutErr += suite.outputInterceptor.StopInterceptingAndReturnOutput()
				suite.outputInterceptor.StartInterceptingOutput()
				if suite.currentSpecReport.State.Is(types.SpecStatePassed) {
					err = suite.client.PostSynchronizedBeforeSuiteCompleted(types.SpecStatePassed, data)
				} else {
					err = suite.client.PostSynchronizedBeforeSuiteCompleted(suite.currentSpecReport.State, nil)
				}
			}
			runAllProcs = suite.currentSpecReport.State.Is(types.SpecStatePassed) && err == nil
		} else {
			var proc1State types.SpecState
			proc1State, data, err = suite.client.BlockUntilSynchronizedBeforeSuiteData()
			switch proc1State {
			case types.SpecStatePassed:
				runAllProcs = true
			case types.SpecStateFailed, types.SpecStatePanicked:
				err = types.GinkgoErrors.SynchronizedBeforeSuiteFailedOnProc1()
			case types.SpecStateInterrupted, types.SpecStateAborted, types.SpecStateSkipped:
				suite.currentSpecReport.State = proc1State
			}
		}
		if runAllProcs {
			node.Body = func() { node.SynchronizedBeforeSuiteAllProcsBody(data) }
			suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(node, interruptChannel, "")
		}
	case types.NodeTypeSynchronizedAfterSuite:
		node.Body = node.SynchronizedAfterSuiteAllProcsBody
		suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(node, interruptChannel, "")
		if suite.config.ParallelProcess == 1 {
			if suite.config.ParallelTotal > 1 {
				err = suite.client.BlockUntilNonprimaryProcsHaveFinished()
			}
			if err == nil {
				if suite.config.ParallelTotal > 1 {
					suite.currentSpecReport.CapturedStdOutErr += suite.outputInterceptor.StopInterceptingAndReturnOutput()
					suite.outputInterceptor.StartInterceptingOutputAndForwardTo(suite.client)
				}

				node.Body = node.SynchronizedAfterSuiteProc1Body
				state, failure := suite.runNode(node, interruptChannel, "")
				if suite.currentSpecReport.State.Is(types.SpecStatePassed) {
					suite.currentSpecReport.State, suite.currentSpecReport.Failure = state, failure
				}
			}
		}
	}

	if err != nil && !suite.currentSpecReport.State.Is(types.SpecStateFailureStates) {
		suite.currentSpecReport.State, suite.currentSpecReport.Failure = types.SpecStateFailed, suite.failureForLeafNodeWithMessage(node, err.Error())
	}

	suite.currentSpecReport.EndTime = time.Now()
	suite.currentSpecReport.RunTime = suite.currentSpecReport.EndTime.Sub(suite.currentSpecReport.StartTime)
	suite.currentSpecReport.CapturedGinkgoWriterOutput = string(suite.writer.Bytes())
	suite.currentSpecReport.CapturedStdOutErr += suite.outputInterceptor.StopInterceptingAndReturnOutput()

	return
}

func (suite *Suite) runReportAfterSuiteNode(node Node, report types.Report) {
	if suite.config.DryRun {
		suite.currentSpecReport.State = types.SpecStatePassed
		return
	}

	suite.writer.Truncate()
	suite.outputInterceptor.StartInterceptingOutput()
	suite.currentSpecReport.StartTime = time.Now()

	if suite.config.ParallelTotal > 1 {
		aggregatedReport, err := suite.client.BlockUntilAggregatedNonprimaryProcsReport()
		if err != nil {
			suite.currentSpecReport.State, suite.currentSpecReport.Failure = types.SpecStateFailed, suite.failureForLeafNodeWithMessage(node, err.Error())
			return
		}
		report = report.Add(aggregatedReport)
	}

	node.Body = func() { node.ReportAfterSuiteBody(report) }
	suite.interruptHandler.SetInterruptPlaceholderMessage(formatter.Fiw(0, formatter.COLS,
		"{{yellow}}Ginkgo received an interrupt signal but is currently running a ReportAfterSuite node.  To avoid an invalid report the ReportAfterSuite node will not be interrupted.{{/}}\n\n{{bold}}The running ReportAfterSuite node is at:\n%s.{{/}}",
		node.CodeLocation,
	))
	suite.currentSpecReport.State, suite.currentSpecReport.Failure = suite.runNode(node, nil, "")
	suite.interruptHandler.ClearInterruptPlaceholderMessage()

	suite.currentSpecReport.EndTime = time.Now()
	suite.currentSpecReport.RunTime = suite.currentSpecReport.EndTime.Sub(suite.currentSpecReport.StartTime)
	suite.currentSpecReport.CapturedGinkgoWriterOutput = string(suite.writer.Bytes())
	suite.currentSpecReport.CapturedStdOutErr = suite.outputInterceptor.StopInterceptingAndReturnOutput()

	return
}

func (suite *Suite) runNode(node Node, interruptChannel chan interface{}, text string) (types.SpecState, types.Failure) {
	if node.NodeType.Is(types.NodeTypeCleanupAfterEach | types.NodeTypeCleanupAfterAll | types.NodeTypeCleanupAfterSuite) {
		suite.cleanupNodes = suite.cleanupNodes.WithoutNode(node)
	}

	suite.currentNode = node
	defer func() {
		suite.currentNode = Node{}
	}()

	if suite.config.EmitSpecProgress {
		if text == "" {
			text = "TOP-LEVEL"
		}
		s := fmt.Sprintf("[%s] %s\n  %s\n", node.NodeType.String(), text, node.CodeLocation.String())
		suite.writer.Write([]byte(s))
	}

	var failure types.Failure
	failure.FailureNodeType, failure.FailureNodeLocation = node.NodeType, node.CodeLocation
	if node.NodeType.Is(types.NodeTypeIt) || node.NodeType.Is(types.NodeTypesForSuiteLevelNodes) {
		failure.FailureNodeContext = types.FailureNodeIsLeafNode
	} else if node.NestingLevel <= 0 {
		failure.FailureNodeContext = types.FailureNodeAtTopLevel
	} else {
		failure.FailureNodeContext, failure.FailureNodeContainerIndex = types.FailureNodeInContainer, node.NestingLevel-1
	}

	outcomeC := make(chan types.SpecState)
	failureC := make(chan types.Failure)

	go func() {
		finished := false
		defer func() {
			if e := recover(); e != nil || !finished {
				suite.failer.Panic(types.NewCodeLocationWithStackTrace(2), e)
			}

			outcome, failureFromRun := suite.failer.Drain()
			outcomeC <- outcome
			failureC <- failureFromRun
		}()

		node.Body()
		finished = true
	}()

	select {
	case outcome := <-outcomeC:
		failureFromRun := <-failureC
		if outcome == types.SpecStatePassed {
			return outcome, types.Failure{}
		}
		failure.Message, failure.Location, failure.ForwardedPanic = failureFromRun.Message, failureFromRun.Location, failureFromRun.ForwardedPanic
		return outcome, failure
	case <-interruptChannel:
		failure.Message, failure.Location = suite.interruptHandler.InterruptMessageWithStackTraces(), node.CodeLocation
		return types.SpecStateInterrupted, failure
	}
}

func (suite *Suite) failureForLeafNodeWithMessage(node Node, message string) types.Failure {
	return types.Failure{
		Message:             message,
		Location:            node.CodeLocation,
		FailureNodeContext:  types.FailureNodeIsLeafNode,
		FailureNodeType:     node.NodeType,
		FailureNodeLocation: node.CodeLocation,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
