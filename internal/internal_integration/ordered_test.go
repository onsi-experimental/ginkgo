package internal_integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi-experimental/ginkgo/v2"
	"github.com/onsi-experimental/ginkgo/v2/internal/interrupt_handler"
	. "github.com/onsi-experimental/ginkgo/v2/internal/test_helpers"
	"github.com/onsi-experimental/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

const SKIP_DUE_TO_EARLIER_FAILURE = "Spec skipped because an earlier spec in an ordered container failed"
const SKIP_DUE_TO_BEFORE_ALL_SKIP = "Spec skipped because Skip() was called in BeforeAll"

var DC = func(label string, callback ...func()) func() {
	return func() {
		DeferCleanup(rt.T(label, callback...))
	}
}

var FlakeyFailer = func(n int) func() {
	i := 0
	return func() {
		i += 1
		if i <= n {
			F("fail")
		}
	}
}

var FlakeyFailerWithCleanup = func(n int, cleanupLabel string) func() {
	i := 0
	return func() {
		i += 1
		DeferCleanup(rt.T(cleanupLabel + "-pre"))
		if i <= n {
			F("fail")
		}
		DeferCleanup(rt.T(cleanupLabel + "-post"))
	}
}

var _ = DescribeTable("Ordered Containers",
	func(expectedSuccess bool, fixture func(), runs []string, args ...interface{}) {
		success, _ := RunFixture(CurrentSpecReport().LeafNodeText, fixture)
		Ω(success).Should(Equal(expectedSuccess))
		Ω(rt).Should(HaveTracked(runs...))
		specs := Reports{}
		for i := 0; i < len(args); i += 1 {
			switch v := args[i].(type) {
			case string:
				specs = append(specs, reporter.Did.Find(v))
			case OmegaMatcher:
				Ω(specs).ShouldNot(BeEmpty(), "Got a matcher but expected a spec to look up")
				for _, spec := range specs {
					Ω(spec).Should(v, "Spec that failed: %s", spec.LeafNodeText)
				}
				specs = Reports{}
			default:
				Fail(fmt.Sprintf("Unexpected type %T", args[i]))
			}
		}
		Ω(specs).Should(BeEmpty(), "Trailing spec - missing a matcher")
	},
	// basic ordering
	Entry("simply happy path", true, func() {
		Context("container", Ordered, func() {
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			It("C", rt.T("C"))
		})
	}, []string{"A", "B", "C"},
		"A", "B", "C", HavePassed(),
	),
	Entry("when a spec fails", false, func() {
		Context("outer container", func() {
			Context("container", Ordered, func() {
				It("A", rt.T("A"))
				It("B", rt.T("B"))
				It("C", rt.T("C", func() { F("fail") }))
				It("D", rt.T("D"))
				It("E", rt.T("E"))
			})
			Context("container", Ordered, func() {
				It("F", FlakeAttempts(3), rt.T("F", FlakeyFailer(2)))
				It("G", rt.T("G"))
			})
		})
	}, []string{"A", "B", "C", "F", "F", "F", "G"},
		"A", "B", HavePassed(),
		"C", HaveFailed(),
		"D", "E", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
		"F", HavePassed(NumAttempts(3)),
		"G", HavePassed(NumAttempts(1)),
	),
	// BeforeAll and AfterAll - happy paths
	Entry("BeforeAll and AfterAll Happy Path", true, func() {
		BeforeEach(rt.T("BE1"))
		JustBeforeEach(rt.T("JBE1"))
		AfterEach(rt.T("AE1"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE2"))
			JustBeforeEach(rt.T("JBE2"))
			BeforeAll(rt.T("BA1"))
			BeforeEach(rt.T("BE3"))
			JustBeforeEach(rt.T("JBE3"))
			BeforeAll(rt.T("BA2"))
			BeforeEach(rt.T("BE4"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			It("C", rt.T("C"))
			JustAfterEach(rt.T("JAE1"))
			AfterEach(rt.T("AE2"))
			AfterAll(rt.T("AA1"))
			AfterEach(rt.T("AE3"))
			JustAfterEach(rt.T("JAE2"))
			AfterAll(rt.T("AA2"))
			AfterEach(rt.T("AE4"))
			JustAfterEach(rt.T("JAE3"))
		})
		JustAfterEach(rt.T("JAE4"))
		AfterEach(rt.T("AE5"))
		BeforeEach(rt.T("BE5"))
		JustBeforeEach(rt.T("JBE4"))
	}, []string{
		"BE1", "BE5",
		"BA1", "BA2", "BE2", "BE3", "BE4",
		"JBE1", "JBE4", "JBE2", "JBE3",
		"A",
		"JAE1", "JAE2", "JAE3", "JAE4",
		"AE2", "AE3", "AE4",
		"AE1", "AE5",
		"BE1", "BE5",
		"BE2", "BE3", "BE4",
		"JBE1", "JBE4", "JBE2", "JBE3",
		"B",
		"JAE1", "JAE2", "JAE3", "JAE4",
		"AE2", "AE3", "AE4",
		"AE1", "AE5",
		"BE1", "BE5",
		"BE2", "BE3", "BE4",
		"JBE1", "JBE4", "JBE2", "JBE3",
		"C",
		"JAE1", "JAE2", "JAE3", "JAE4",
		"AE2", "AE3", "AE4", "AA1", "AA2",
		"AE1", "AE5",
	}),
	Entry("when there is only one spec", true, func() {
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
	}, []string{
		"BA", "BE", "A", "AE", "AA",
	}),
	Entry("when there are focused specs", true, func() {
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			FIt("B", rt.T("B"))
			FIt("C", rt.T("C"))
			It("D", rt.T("D"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
	}, []string{
		"BA", "BE", "B", "AE", "BE", "C", "AE", "AA",
	},
		"B", "C", HavePassed(),
		"A", "D", HaveBeenSkipped(),
	),
	Entry("when there is nothing to run", true, func() {
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			PIt("A", rt.T("A"))
			PIt("B", rt.T("B"))
			PIt("C", rt.T("C"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
	}, []string{}, "A", "B", "C", BePending()),
	// BeforeAll and AfterAll - when skips are called
	Entry("when a skip occurs in a BeforeAll, it skips the entire group", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA", func() { DeferCleanup(rt.T("DC")); Skip("skip") }))
			It("A", FlakeAttempts(3), rt.T("A"))
			It("B", rt.T("B"))
			It("C", rt.T("C"))
			AfterAll(rt.T("AA"))
		})
	}, []string{"BA", "AA", "DC"},
		"A", HaveBeenSkippedWithMessage("skip", NumAttempts(1)),
		"B", "C", HaveBeenSkippedWithMessage(SKIP_DUE_TO_BEFORE_ALL_SKIP),
	),
	Entry("when a skip occurs in a test", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B", func() { Skip("skip") }))
			It("C", rt.T("C"))
			AfterAll(rt.T("AA"))
		})
	}, []string{"BA", "A", "B", "C", "AA"},
		"A", "C", HavePassed(),
		"B", HaveBeenSkippedWithMessage("skip"),
	),
	Entry("when a skip occurs in the last test", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			It("C", rt.T("C", func() { Skip("skip") }))
			AfterAll(rt.T("AA"))
		})
	}, []string{"BA", "A", "B", "C", "AA"},
		"A", "B", HavePassed(),
		"C", HaveBeenSkippedWithMessage("skip"),
	),
	// BeforeAll and AfterAll - when failures, panics, interrupts, and aborts happen
	Entry("when a failure occurs prior to the BeforeAll, it doesn't run the Alls", false, func() {
		BeforeEach(rt.T("BE-outer", func() { F("fail") }))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{
		"BE-outer", "AE-outer",
	}, "A", HaveFailed(types.FailureNodeAtTopLevel, FailureNodeType(types.NodeTypeBeforeEach), "fail")),
	Entry("when a failure occurs in a spec, it runs the AfterAll and skips subsequent specs", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A", func() { F("fail") }))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "BE", "A", "AE", "AA", "AE-outer"},
		"A", HaveFailed(types.FailureNodeIsLeafNode, FailureNodeType(types.NodeTypeIt), "fail"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when a failure occurs in a BeforeAll", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA", func() { F("fail") }))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "AE", "AA", "AE-outer"},
		"A", HaveFailed(types.FailureNodeInContainer, FailureNodeType(types.NodeTypeBeforeAll), "fail"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when a failure occurs in an AfterAll", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA", func() { F("fail") }))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{
		"BE-outer", "BA", "BE", "A", "AE", "AE-outer",
		"BE-outer", "BE", "B", "AE", "AA", "AE-outer",
	}, "A", HavePassed(),
		"B", HaveFailed(types.FailureNodeInContainer, FailureNodeType(types.NodeTypeAfterAll), "fail"),
	),
	Entry("when a panic occurs in a BeforeAll", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA", func() { panic("boom") }))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "AE", "AA", "AE-outer"},
		"A", HavePanicked(types.FailureNodeInContainer, FailureNodeType(types.NodeTypeBeforeAll), "boom"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when a panic occurs in an AfterAll", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA", func() { panic("boom") }))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{
		"BE-outer", "BA", "BE", "A", "AE", "AE-outer",
		"BE-outer", "BE", "B", "AE", "AA", "AE-outer",
	}, "A", HavePassed(),
		"B", HavePanicked(types.FailureNodeInContainer, FailureNodeType(types.NodeTypeAfterAll), "boom"),
	),
	Entry("when a failure occurs in an AfterEach, it runs the AfterAll", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE", func() { F("fail") }))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "BE", "A", "AE", "AE-outer", "AA"},
		"A", HaveFailed(types.FailureNodeInContainer, FailureNodeType(types.NodeTypeAfterEach), "fail"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when a failure occurs in a DeferCleanup, it runs the AfterAll", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A", func() {
				DeferCleanup(func() {
					rt.Run("cleanup")
					Fail("fail")
				})
			}))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "BE", "A", "AE", "AE-outer", "cleanup", "AA"},
		"A", HaveFailed(types.FailureNodeInContainer, FailureNodeType(types.NodeTypeCleanupAfterEach), "fail"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when an interruption occurs, run the AfterAll and skip subsequent specs", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A", func() {
				interruptHandler.Interrupt(interrupt_handler.InterruptCauseSignal)
				time.Sleep(time.Minute)
			}))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "BE", "A", "AE", "AA", "AE-outer"},
		"A", HaveBeenInterrupted(interrupt_handler.InterruptCauseSignal),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when an interruption occurs in a BeforeAll, run the AfterAll and skip subsequent specs", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA", func() {
				DeferCleanup(rt.T("DC-BA"))
				interruptHandler.Interrupt(interrupt_handler.InterruptCauseSignal)
				time.Sleep(time.Minute)
			}))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "AE", "AA", "AE-outer", "DC-BA"},
		"A", HaveBeenInterrupted(interrupt_handler.InterruptCauseSignal),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when an interruption occurs in an AfterAll, run any remaining cleanup", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA", DC("DC-BA")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA", func() {
				DeferCleanup(rt.T("DC-AA"))
				interruptHandler.Interrupt(interrupt_handler.InterruptCauseSignal)
				time.Sleep(time.Minute)
			}))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{
		"BE-outer", "BA", "BE", "A", "AE", "AE-outer",
		"BE-outer", "BE", "B", "AE", "AA", "AE-outer",
		"DC-AA", "DC-BA",
	},
		"A", HavePassed(),
		"B", HaveBeenInterrupted(interrupt_handler.InterruptCauseSignal),
	),
	Entry("when an abort occurs, run the AfterAll and skip subsequent specs", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A", func() {
				Abort("abort!")
			}))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "BE", "A", "AE", "AA", "AE-outer"},
		"A", HaveAborted("abort!"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when an abort occurs in a BeforeAll, run the AfterAll and skip subsequent specs", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA", func() {
				DeferCleanup(rt.T("DC-BA"))
				Abort("abort!")
			}))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA"))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{"BE-outer", "BA", "AE", "AA", "AE-outer", "DC-BA"},
		"A", HaveAborted("abort!"),
		"B", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when an abort occurs in an AfterAll, run any remaining cleanup", false, func() {
		BeforeEach(rt.T("BE-outer"))
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeEach(rt.T("BE"))
			BeforeAll(rt.T("BA", DC("DC-BA")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			AfterAll(rt.T("AA", func() {
				DeferCleanup(rt.T("DC-AA"))
				Abort("abort!")
			}))
			AfterEach(rt.T("AE"))
		})
		AfterEach(rt.T("AE-outer"))
	}, []string{
		"BE-outer", "BA", "BE", "A", "AE", "AE-outer",
		"BE-outer", "BE", "B", "AE", "AA", "AE-outer",
		"DC-AA", "DC-BA",
	},
		"A", HavePassed(),
		"B", HaveAborted("abort!"),
	),
	//here be dragons: the interplay between BeforeAll/AfterAll and FlakeAttempts
	Entry("when the first spec is flaky, it runs the BeforeAll just once", true, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A", FlakeyFailer(2)))
			It("B", rt.T("B"))
			It("C", rt.T("C"))
			AfterAll(rt.T("AA"))
		})
	}, []string{"BA", "A", "A", "A", "B", "C", "AA"},
		"A", HavePassed(NumAttempts(3)),
		"B", "C", HavePassed(NumAttempts(1)),
	),
	Entry("when a spec is flaky and never succeeds, it runs the AfterAll (just once) when the spec ultimately fails", false, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B", FlakeyFailer(4)))
			It("C", rt.T("C"))
			AfterAll(rt.T("AA"))
		})
	}, []string{"BA", "A", "B", "B", "B", "B", "AA"},
		"A", HavePassed(),
		"B", HaveFailed("fail", NumAttempts(4)),
		"C", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when the last spec is flaky, it runs the AFterAll just once", true, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			It("C", rt.T("C", FlakeyFailer(2)))
			AfterAll(rt.T("AA"))
		})
	}, []string{"BA", "A", "B", "C", "C", "C", "AA"},
		"A", "B", HavePassed(NumAttempts(1)),
		"C", HavePassed(NumAttempts(3)),
	),
	Entry("When the BeforeAll is flaky", true, func() {
		Context("container", Ordered, FlakeAttempts(5), func() {
			BeforeAll(rt.T("BA", FlakeyFailerWithCleanup(2, "DC-BA")))
			It("A", rt.T("A", FlakeyFailer(2)))
			It("B", rt.T("B"))
			It("C", rt.T("C"))
			AfterAll(rt.T("AA"))
		})
	}, []string{
		"BA", "AA", "DC-BA-pre",
		"BA", "AA", "DC-BA-pre",
		"BA", "A", "A", "A",
		"B", "C",
		"AA", "DC-BA-post", "DC-BA-pre",
	},
		"A", HavePassed(NumAttempts(5)),
		"B", "C", HavePassed(),
	),
	Entry("When the AFterAll is flaky", true, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA", DC("DC-BA")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			It("C", rt.T("C"))
			AfterAll(rt.T("AA", FlakeyFailerWithCleanup(2, "DC-AA")))
		})
	}, []string{
		"BA", "A", "B", "C", "AA", "DC-AA-pre",
		"C", "AA", "DC-AA-pre",
		"C", "AA", "DC-AA-post", "DC-AA-pre", "DC-BA",
	},
		"A", "B", HavePassed(),
		"C", HavePassed(NumAttempts(3)),
	),

	//Let's enter the dragons nest!
	Entry("happy-path for nested containers", true, func() {
		BeforeEach(rt.T("BE-L0"))
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-1-L1", DC("DC-BA-L1")))
			BeforeAll(rt.T("BA-2-L1"))
			BeforeEach(rt.T("BE-L1"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-1-L2", DC("DC-BA-L2")))
				BeforeAll(rt.T("BA-2-L2"))
				BeforeEach(rt.T("BE-L2"))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterEach(rt.T("AE-L2"))
				AfterAll(rt.T("AA-1-L2", DC("DC-AA-L2")))
				AfterAll(rt.T("AA-2-L2"))
			})
			It("E", rt.T("E"))
			AfterEach(rt.T("AE-L1"))
			AfterAll(rt.T("AA-1-L1", DC("DC-AA-L1")))
			AfterAll(rt.T("AA-2-L1"))
		})
		AfterEach(rt.T("AE-L0"))
	}, []string{
		"BE-L0", "BA-1-L1", "BA-2-L1", "BE-L1", "A", "AE-L1", "AE-L0",
		"BE-L0", "BE-L1", "B", "AE-L1", "AE-L0",
		"BE-L0", "BE-L1", "BA-1-L2", "BA-2-L2", "BE-L2", "C", "AE-L2", "AE-L1", "AE-L0",
		"BE-L0", "BE-L1", "BE-L2", "D", "AE-L2", "AA-1-L2", "AA-2-L2", "AE-L1", "AE-L0", "DC-AA-L2", "DC-BA-L2",
		"BE-L0", "BE-L1", "E", "AE-L1", "AA-1-L1", "AA-2-L1", "AE-L0", "DC-AA-L1", "DC-BA-L1",
	}),
	Entry("happy-path where last spec is in nested container", true, func() {
		BeforeEach(rt.T("BE-L0"))
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-1-L1", DC("DC-BA-L1")))
			BeforeAll(rt.T("BA-2-L1"))
			BeforeEach(rt.T("BE-L1"))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-1-L2", DC("DC-BA-L2")))
				BeforeAll(rt.T("BA-2-L2"))
				BeforeEach(rt.T("BE-L2"))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterEach(rt.T("AE-L2"))
				AfterAll(rt.T("AA-1-L2", DC("DC-AA-L2")))
				AfterAll(rt.T("AA-2-L2"))
			})
			AfterEach(rt.T("AE-L1"))
			AfterAll(rt.T("AA-1-L1", DC("DC-AA-L1")))
			AfterAll(rt.T("AA-2-L1"))
		})
		AfterEach(rt.T("AE-L0"))
	}, []string{
		"BE-L0", "BA-1-L1", "BA-2-L1", "BE-L1", "A", "AE-L1", "AE-L0",
		"BE-L0", "BE-L1", "B", "AE-L1", "AE-L0",
		"BE-L0", "BE-L1", "BA-1-L2", "BA-2-L2", "BE-L2", "C", "AE-L2", "AE-L1", "AE-L0",
		"BE-L0", "BE-L1", "BE-L2", "D", "AE-L2", "AA-1-L2", "AA-2-L2", "AE-L1", "AA-1-L1", "AA-2-L1", "AE-L0", "DC-AA-L1", "DC-AA-L2", "DC-BA-L2", "DC-BA-L1",
	}),
	Entry("when an outer spec is skipped", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A", func() { Skip("skip") }))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "A", "B", "BA-I", "C", "D", "AA-I", "DC-I", "E", "AA-O", "DC-O"},
		"A", HaveBeenSkippedWithMessage("skip"),
		"B", "C", "D", "E", HavePassed(),
	),
	Entry("when an inner spec is skipped", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C", func() { Skip("skip") }))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "A", "B", "BA-I", "C", "D", "AA-I", "DC-I", "E", "AA-O", "DC-O"},
		"C", HaveBeenSkippedWithMessage("skip"),
		"A", "B", "D", "E", HavePassed(),
	),
	Entry("when an outer BeforeAll is skipped", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", func() { DeferCleanup(rt.T("DC-O")); Skip("skip") }))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I"))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "AA-O", "DC-O"},
		"A", HaveBeenSkippedWithMessage("skip"),
		"B", "C", "D", "E", HaveBeenSkippedWithMessage(SKIP_DUE_TO_BEFORE_ALL_SKIP),
	),
	Entry("when an inner BeforeAll is skipped", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", func() { DeferCleanup(rt.T("DC-I")); Skip("skip") }))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "A", "B", "BA-I", "AA-I", "DC-I", "E", "AA-O", "DC-O"},
		"A", "B", "E", HavePassed(),
		"C", HaveBeenSkippedWithMessage("skip"),
		"D", HaveBeenSkippedWithMessage(SKIP_DUE_TO_BEFORE_ALL_SKIP),
	),
	Entry("when the last spec is marked as pending", true, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			FIt("A", rt.T("A"))
			FIt("B", rt.T("B"))
			FContext("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C"))
				PIt("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			FIt("E", rt.T("E"))
			It("F", rt.T("F"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "A", "B", "BA-I", "C", "AA-I", "DC-I", "E", "AA-O", "DC-O"},
		"A", "B", "C", "E", HavePassed(),
		"D", BePending(), "F", HaveBeenSkipped(),
	),
	Entry("when an outer spec fails", false, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A"))
			It("B", rt.T("B", func() { F("fail") }))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "A", "B", "AA-O", "DC-O"},
		"A", HavePassed(), "B", HaveFailed("fail"),
		"C", "D", "E", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when an inner spec fails", false, func() {
		Context("container", Ordered, func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C", func() { F("fail") }))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{"BA-O", "A", "B", "BA-I", "C", "AA-I", "AA-O", "DC-I", "DC-O"},
		"A", HavePassed(), "B", HavePassed(), "C", HaveFailed("fail"),
		"D", "E", HaveBeenSkippedWithMessage(SKIP_DUE_TO_EARLIER_FAILURE),
	),
	Entry("when flakey, and an outer BeforeAll flakes", true, func() {
		i := 0
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA-O", func() {
				i += 1
				DeferCleanup(rt.T("DC-O"))
				if i < 3 {
					F("fail")
				}
			}))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{
		"BA-O", "AA-O", "DC-O",
		"BA-O", "AA-O", "DC-O",
		"BA-O", "A", "B", "BA-I", "C", "D", "AA-I", "DC-I", "E", "AA-O", "DC-O"},
		"A", HavePassed(NumAttempts(3)),
		"B", "C", "D", "E", HavePassed(),
	),
	Entry("when flakey, and an inner BeforeAll flakes", true, func() {
		i := 0
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", func() {
					i += 1
					DeferCleanup(rt.T("DC-I"))
					if i < 3 {
						F("fail")
					}
				}))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{
		"BA-O", "A", "B",
		"BA-I", "AA-I", "DC-I",
		"BA-I", "AA-I", "DC-I",
		"BA-I", "C", "D", "AA-I", "DC-I", "E", "AA-O", "DC-O"},
		"A", "B", "D", "E", HavePassed(),
		"C", HavePassed(NumAttempts(3)),
	),
	Entry("when specs are flakey", true, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A", FlakeyFailer(2)))
			It("B", rt.T("B", FlakeyFailer(2)))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C", FlakeyFailer(2)))
				It("D", rt.T("D", FlakeyFailer(2)))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E", FlakeyFailer(2)))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{
		"BA-O", "A", "A", "A",
		"B", "B", "B",
		"BA-I", "C", "C", "C",
		"D", "D", "D", "AA-I", "DC-I",
		"E", "E", "E", "AA-O", "DC-O",
	},
		"A", "B", "C", "D", "E", HavePassed(NumAttempts(3)),
	),
	Entry("when AfterAlls are flakey", true, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA-O", DC("DC-O")))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I")))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I", FlakeyFailer(2)))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O", FlakeyFailer(2)))
		})
	}, []string{
		"BA-O", "A", "B", "BA-I", "C",
		"D", "AA-I", "D", "AA-I", "D", "AA-I", "DC-I",
		"E", "AA-O", "E", "AA-O", "E", "AA-O", "DC-O",
	},
		"A", "B", "C", "D", "E", HavePassed(),
	),
	//this behavior is a bit weird, but it's such an edge case that we're going to leave it
	//unless an issue gets opened
	Entry("when DeferCleanups are flakey", true, func() {
		Context("container", Ordered, FlakeAttempts(4), func() {
			BeforeAll(rt.T("BA-O", DC("DC-O", FlakeyFailer(2))))
			It("A", rt.T("A"))
			It("B", rt.T("B"))
			Context("inner", func() {
				BeforeAll(rt.T("BA-I", DC("DC-I", FlakeyFailer(2))))
				It("C", rt.T("C"))
				It("D", rt.T("D"))
				AfterAll(rt.T("AA-I"))
			})
			It("E", rt.T("E"))
			AfterAll(rt.T("AA-O"))
		})
	}, []string{
		"BA-O", "A", "B", "BA-I", "C", "D", "AA-I", "DC-I", "D", "AA-I",
		"E", "AA-O", "DC-O", "E", "AA-O",
	},
		"A", "B", "C", "D", "E", HavePassed(),
	),
)
