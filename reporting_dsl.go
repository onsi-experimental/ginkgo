package ginkgo

import (
	"fmt"
	"strings"

	"github.com/onsi-experimental/ginkgo/v2/internal"
	"github.com/onsi-experimental/ginkgo/v2/internal/global"
	"github.com/onsi-experimental/ginkgo/v2/reporters"
	"github.com/onsi-experimental/ginkgo/v2/types"
)

/*
Report represents the report for a Suite.
It is documented here: https://pkg.go.dev/github.com/onsi-experimental/ginkgo/v2/types#Report
*/
type Report = types.Report

/*
Report represents the report for a Spec.
It is documented here: https://pkg.go.dev/github.com/onsi-experimental/ginkgo/v2/types#SpecReport
*/
type SpecReport = types.SpecReport

/*
CurrentSpecReport returns information about the current running spec.
The returned object is a types.SpecReport which includes helper methods
to make extracting information about the spec easier.

You can learn more about SpecReport here: https://pkg.go.dev/github.com/onsi-experimental/ginkgo/v2/types#SpecReport
You can learn more about CurrentSpecReport() here: https://onsi.github.io/ginkgo/#getting-a-report-for-the-current-spec
*/
func CurrentSpecReport() SpecReport {
	return global.Suite.CurrentSpecReport()
}

/*
 ReportEntryVisibility governs the visibility of ReportEntries in Ginkgo's console reporter

- ReportEntryVisibilityAlways: the default behavior - the ReportEntry is always emitted.
- ReportEntryVisibilityFailureOrVerbose: the ReportEntry is only emitted if the spec fails or if the tests are run with -v (similar to GinkgoWriters behavior).
- ReportEntryVisibilityNever: the ReportEntry is never emitted though it appears in any generated machine-readable reports (e.g. by setting `--json-report`).

You can learn more about Report Entries here: https://onsi.github.io/ginkgo/#attaching-data-to-reports
*/
type ReportEntryVisibility = types.ReportEntryVisibility

const ReportEntryVisibilityAlways, ReportEntryVisibilityFailureOrVerbose, ReportEntryVisibilityNever = types.ReportEntryVisibilityAlways, types.ReportEntryVisibilityFailureOrVerbose, types.ReportEntryVisibilityNever

/*
AddReportEntry generates and adds a new ReportEntry to the current spec's SpecReport.
It can take any of the following arguments:
   - A single arbitrary object to attach as the Value of the ReportEntry.  This object will be included in any generated reports and will be emitted to the console when the report is emitted.
   - A ReportEntryVisibility enum to control the visibility of the ReportEntry
   - An Offset or CodeLocation decoration to control the reported location of the ReportEntry

If the Value object implements `fmt.Stringer`, it's `String()` representation is used when emitting to the console.

AddReportEntry() must be called within a Subject or Setup node - not in a Container node.

You can learn more about Report Entries here: https://onsi.github.io/ginkgo/#attaching-data-to-reports
*/
func AddReportEntry(name string, args ...interface{}) {
	cl := types.NewCodeLocation(1)
	reportEntry, err := internal.NewReportEntry(name, cl, args...)
	if err != nil {
		Fail(fmt.Sprintf("Failed to generate Report Entry:\n%s", err.Error()), 1)
	}
	err = global.Suite.AddReportEntry(reportEntry)
	if err != nil {
		Fail(fmt.Sprintf("Failed to add Report Entry:\n%s", err.Error()), 1)
	}
}

/*
ReportBeforeEach nodes are run for each spec, even if the spec is skipped or pending.  ReportBeforeEach nodes take a function that
receives a SpecReport.  They are called before the spec starts.

You cannot nest any other Ginkgo nodes within a ReportBeforeEach node's closure.
You can learn more about ReportBeforeEach here: https://onsi.github.io/ginkgo/#generating-reports-programmatically
*/
func ReportBeforeEach(body func(SpecReport)) bool {
	return pushNode(internal.NewReportBeforeEachNode(body, types.NewCodeLocation(1)))
}

/*
ReportAfterEach nodes are run for each spec, even if the spec is skipped or pending.  ReportAfterEach nodes take a function that
receives a SpecReport.  They are called after the spec has completed and receive the final report for the spec.

You cannot nest any other Ginkgo nodes within a ReportAfterEach node's closure.
You can learn more about ReportAfterEach here: https://onsi.github.io/ginkgo/#generating-reports-programmatically
*/
func ReportAfterEach(body func(SpecReport)) bool {
	return pushNode(internal.NewReportAfterEachNode(body, types.NewCodeLocation(1)))
}

/*
ReportAfterSuite nodes are run at the end of the suite.  ReportAfterSuite nodes take a function that receives a suite Report.

They are called at the end of the suite, after all specs have run and any AfterSuite or SynchronizedAfterSuite nodes, and are passed in the final report for the suite.
ReportAftersuite nodes must be created at the top-level (i.e. not nested in a Context/Describe/When node)

When running in parallel, Ginkgo ensures that only one of the parallel nodes runs the ReportAfterSuite and that it is passed a report that is aggregated across
all parallel nodes

In addition to using ReportAfterSuite to programatically generate suite reports, you can also generate JSON, JUnit, and Teamcity formatted reports using the --json-report, --junit-report, and --teamcity-report ginkgo CLI flags.

You cannot nest any other Ginkgo nodes within a ReportAfterSuite node's closure.
You can learn more about ReportAfterSuite here: https://onsi.github.io/ginkgo/#generating-reports-programmatically
You can learn more about Ginkgo's reporting infrastructure, including generating reports with the CLI here: https://onsi.github.io/ginkgo/#generating-machine-readable-reports
*/
func ReportAfterSuite(text string, body func(Report)) bool {
	return pushNode(internal.NewReportAfterSuiteNode(text, body, types.NewCodeLocation(1)))
}

func registerReportAfterSuiteNodeForAutogeneratedReports(reporterConfig types.ReporterConfig) {
	body := func(report Report) {
		if reporterConfig.JSONReport != "" {
			err := reporters.GenerateJSONReport(report, reporterConfig.JSONReport)
			if err != nil {
				Fail(fmt.Sprintf("Failed to generate JSON report:\n%s", err.Error()))
			}
		}
		if reporterConfig.JUnitReport != "" {
			err := reporters.GenerateJUnitReport(report, reporterConfig.JUnitReport)
			if err != nil {
				Fail(fmt.Sprintf("Failed to generate JSON report:\n%s", err.Error()))
			}
		}
		if reporterConfig.TeamcityReport != "" {
			err := reporters.GenerateTeamcityReport(report, reporterConfig.TeamcityReport)
			if err != nil {
				Fail(fmt.Sprintf("Failed to generate JSON report:\n%s", err.Error()))
			}
		}
	}

	flags := []string{}
	if reporterConfig.JSONReport != "" {
		flags = append(flags, "--json-report")
	}
	if reporterConfig.JUnitReport != "" {
		flags = append(flags, "--junit-report")
	}
	if reporterConfig.TeamcityReport != "" {
		flags = append(flags, "--teamcity-report")
	}
	pushNode(internal.NewReportAfterSuiteNode(
		fmt.Sprintf("Autogenerated ReportAfterSuite for %s", strings.Join(flags, " ")),
		body,
		types.NewCustomCodeLocation("autogenerated by Ginkgo"),
	))
}
