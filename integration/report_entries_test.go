package integration_test

import (
	"time"

	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi-experimental/ginkgo/v2/internal/test_helpers"
	"github.com/onsi-experimental/ginkgo/v2/reporters"
	"github.com/onsi-experimental/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ReportEntries", func() {
	var output string
	var report types.Report
	var junit reporters.JUnitTestSuite

	Describe("when running a test that adds report entries", func() {
		BeforeEach(func() {
			fm.MountFixture("report_entries")
			session := startGinkgo(fm.PathTo("report_entries"), "--no-color", "--procs=2", "--json-report=out.json", "--junit-report=out.xml")
			Eventually(session).Should(gexec.Exit(1))
			output = string(session.Out.Contents())
			report = fm.LoadJSONReports("report_entries", "out.json")[0]
			junit = fm.LoadJUnitReport("report_entries", "out.xml").TestSuites[0]
		})

		It("should honor the report visibilities and stringer formatting when emitting output", func() {
			Ω(output).Should(ContainSubstring("passes-first-report"))
			Ω(output).Should(ContainSubstring("pass-bob 1"))
			Ω(output).Should(ContainSubstring("passes-second-report"))
			Ω(output).Should(ContainSubstring("passes-third-report"))
			Ω(output).Should(ContainSubstring("passes-pointer-report"))
			Ω(output).Should(ContainSubstring("passed 4"))
			Ω(output).ShouldNot(ContainSubstring("passes-failure-report"))
			Ω(output).ShouldNot(ContainSubstring("passes-never-see-report"))

			Ω(output).Should(ContainSubstring("fails-first-report"))
			Ω(output).Should(ContainSubstring("fail-bob 1"))
			Ω(output).Should(ContainSubstring("fails-second-report"))
			Ω(output).Should(ContainSubstring("fails-third-report"))
			Ω(output).Should(ContainSubstring("fails-pointer-report"))
			Ω(output).Should(ContainSubstring("failed 4"))
			Ω(output).Should(ContainSubstring("fails-failure-report"))
			Ω(output).ShouldNot(ContainSubstring("fails-never-see-report"))

			Ω(output).ShouldNot(ContainSubstring("registers a hidden AddReportEntry"))
		})

		It("captures all report entries in the JSON report", func() {
			reports := Reports(report.SpecReports)
			passes := reports.Find("passes")
			Ω(passes.ReportEntries).Should(HaveLen(6))
			Ω(passes.ReportEntries[0].Name).Should(Equal("passes-first-report"))
			Ω(passes.ReportEntries[0].GetRawValue()).Should(Equal(map[string]interface{}{"Label": "pass-bob", "Count": float64(1)}))
			Ω(passes.ReportEntries[0].StringRepresentation()).Should(Equal("{{red}}pass-bob {{green}}1{{/}}"))
			Ω(passes.ReportEntries[0].Time).Should(BeTemporally("~", time.Now(), time.Minute))

			Ω(passes.ReportEntries[1].Name).Should(Equal("passes-second-report"))
			Ω(passes.ReportEntries[1].GetRawValue()).Should(BeNil())
			Ω(passes.ReportEntries[1].StringRepresentation()).Should(BeZero())

			Ω(passes.ReportEntries[2].Name).Should(Equal("passes-third-report"))
			Ω(passes.ReportEntries[2].GetRawValue()).Should(Equal(float64(3)))
			Ω(passes.ReportEntries[2].StringRepresentation()).Should(Equal("3"))

			Ω(passes.ReportEntries[3].Name).Should(Equal("passes-pointer-report"))
			Ω(passes.ReportEntries[3].GetRawValue()).Should(Equal(map[string]interface{}{"Label": "passed", "Count": float64(4)}))
			Ω(passes.ReportEntries[3].StringRepresentation()).Should(Equal("{{red}}passed {{green}}4{{/}}"))

			Ω(passes.ReportEntries[4].Name).Should(Equal("passes-failure-report"))
			Ω(passes.ReportEntries[4].GetRawValue()).Should(Equal(float64(5)))
			Ω(passes.ReportEntries[4].StringRepresentation()).Should(Equal("5"))

			Ω(passes.ReportEntries[5].Name).Should(Equal("passes-never-see-report"))
			Ω(passes.ReportEntries[5].GetRawValue()).Should(Equal(float64(6)))
			Ω(passes.ReportEntries[5].StringRepresentation()).Should(Equal("6"))

			fails := reports.Find("fails")
			Ω(fails.ReportEntries[0].Name).Should(Equal("fails-first-report"))
			Ω(fails.ReportEntries[0].GetRawValue()).Should(Equal(map[string]interface{}{"Label": "fail-bob", "Count": float64(1)}))
			Ω(fails.ReportEntries[0].StringRepresentation()).Should(Equal("{{red}}fail-bob {{green}}1{{/}}"))
			Ω(fails.ReportEntries[0].Time).Should(BeTemporally("~", time.Now(), time.Minute))

			Ω(fails.ReportEntries[1].Name).Should(Equal("fails-second-report"))
			Ω(fails.ReportEntries[1].GetRawValue()).Should(BeNil())
			Ω(fails.ReportEntries[1].StringRepresentation()).Should(BeZero())

			Ω(fails.ReportEntries[2].Name).Should(Equal("fails-third-report"))
			Ω(fails.ReportEntries[2].GetRawValue()).Should(Equal(float64(3)))
			Ω(fails.ReportEntries[2].StringRepresentation()).Should(Equal("3"))

			Ω(fails.ReportEntries[3].Name).Should(Equal("fails-pointer-report"))
			Ω(fails.ReportEntries[3].GetRawValue()).Should(Equal(map[string]interface{}{"Label": "failed", "Count": float64(4)}))
			Ω(fails.ReportEntries[3].StringRepresentation()).Should(Equal("{{red}}failed {{green}}4{{/}}"))

			Ω(fails.ReportEntries[4].Name).Should(Equal("fails-failure-report"))
			Ω(fails.ReportEntries[4].GetRawValue()).Should(Equal(float64(5)))
			Ω(fails.ReportEntries[4].StringRepresentation()).Should(Equal("5"))

			Ω(fails.ReportEntries[5].Name).Should(Equal("fails-never-see-report"))
			Ω(fails.ReportEntries[5].GetRawValue()).Should(Equal(float64(6)))
			Ω(fails.ReportEntries[5].StringRepresentation()).Should(Equal("6"))

			by := reports.Find("has By entries")
			Ω(by.ReportEntries[0].Name).Should(Equal("By Step"))
			Ω(by.ReportEntries[0].Visibility).Should(Equal(ReportEntryVisibilityNever))
			value := by.ReportEntries[0].GetRawValue().(map[string]interface{})
			Ω(value["Text"]).Should(Equal("registers a hidden AddReportEntry"))
			Ω(value["Duration"]).Should(BeZero())
			Ω(by.ReportEntries[1].Name).Should(Equal("By Step"))
			Ω(by.ReportEntries[1].Visibility).Should(Equal(ReportEntryVisibilityNever))
			value = by.ReportEntries[1].GetRawValue().(map[string]interface{})
			Ω(value["Text"]).Should(Equal("includes durations"))
			Ω(time.Duration(value["Duration"].(float64))).Should(BeNumerically("~", time.Millisecond*100, time.Millisecond*100))
		})

		It("captures all report entries in the JUnit report", func() {
			var content string
			for _, testCase := range junit.TestCases {
				if testCase.Name == "[It] top-level container passes" {
					content = testCase.SystemOut
				}
			}

			buf := gbytes.BufferWithBytes([]byte(content))

			Ω(buf).Should(gbytes.Say("Report Entries:"))
			Ω(buf).Should(gbytes.Say("passes-first-report"))
			Ω(buf).Should(gbytes.Say(`report_entries/report_entries_fixture_suite_test\.go:\d+`))
			Ω(buf).Should(gbytes.Say("{{red}}pass-bob {{green}}1{{/}}"))
			Ω(buf).Should(gbytes.Say("--"))
			Ω(buf).Should(gbytes.Say("passes-second-report"))
			Ω(buf).Should(gbytes.Say("--"))
			Ω(buf).Should(gbytes.Say("passes-third-report"))
			Ω(buf).Should(gbytes.Say("3"))
			Ω(buf).Should(gbytes.Say("--"))
			Ω(buf).Should(gbytes.Say("passes-pointer-report"))
			Ω(buf).Should(gbytes.Say("{{red}}passed {{green}}4{{/}}"))
			Ω(buf).Should(gbytes.Say("--"))
			Ω(buf).Should(gbytes.Say("passes-failure-report"))
			Ω(buf).Should(gbytes.Say("5"))
			Ω(buf).Should(gbytes.Say("--"))
			Ω(buf).Should(gbytes.Say("passes-never-see-report"))
			Ω(buf).Should(gbytes.Say("6"))
			Ω(buf).ShouldNot(gbytes.Say("--"))
		})
	})

	Describe("when running in verbose mode", func() {
		BeforeEach(func() {
			fm.MountFixture("report_entries")
			session := startGinkgo(fm.PathTo("report_entries"), "--no-color", "--procs=2", "-v")
			Eventually(session).Should(gexec.Exit(1))
			output = string(session.Out.Contents())
		})

		It("should honor the report visibilities and stringer formatting when emitting output", func() {
			Ω(output).Should(ContainSubstring("passes-first-report"))
			Ω(output).Should(ContainSubstring("pass-bob 1"))
			Ω(output).Should(ContainSubstring("passes-second-report"))
			Ω(output).Should(ContainSubstring("passes-third-report"))
			Ω(output).Should(ContainSubstring("passes-pointer-report"))
			Ω(output).Should(ContainSubstring("passed 4"))
			Ω(output).Should(ContainSubstring("passes-failure-report"))
			Ω(output).ShouldNot(ContainSubstring("passes-never-see-report"))

			Ω(output).Should(ContainSubstring("fails-first-report"))
			Ω(output).Should(ContainSubstring("fail-bob 1"))
			Ω(output).Should(ContainSubstring("fails-second-report"))
			Ω(output).Should(ContainSubstring("fails-third-report"))
			Ω(output).Should(ContainSubstring("fails-pointer-report"))
			Ω(output).Should(ContainSubstring("failed 4"))
			Ω(output).Should(ContainSubstring("fails-failure-report"))
			Ω(output).ShouldNot(ContainSubstring("fails-never-see-report"))
		})
	})
})
