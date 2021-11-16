package fail_fixture_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi-experimental/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = It("handles top level failures", func() {
	Ω("a top level failure on line 10").Should(Equal("nope"))
	println("NEVER SEE THIS")
})

var _ = It("handles async top level failures", func(done Done) {
	Fail("an async top level failure on line 15")
	println("NEVER SEE THIS")
}, 0.1)

var _ = It("FAIL in a goroutine", func(done Done) {
	go func() {
		defer GinkgoRecover()
		Fail("a top level goroutine failure on line 22")
		println("NEVER SEE THIS")
	}()
}, 0.1)

var _ = Describe("Excercising different failure modes", func() {
	It("synchronous failures", func() {
		Ω("a sync failure").Should(Equal("nope"))
		println("NEVER SEE THIS")
	})

	It("synchronous panics", func() {
		panic("a sync panic")
		println("NEVER SEE THIS")
	})

	It("synchronous failures with FAIL", func() {
		Fail("a sync FAIL failure")
		println("NEVER SEE THIS")
	})

	It("async timeout", func(done Done) {
		Ω(true).Should(BeTrue())
	}, 0.1)

	It("async failure", func(done Done) {
		Ω("an async failure").Should(Equal("nope"))
		println("NEVER SEE THIS")
	}, 0.1)

	It("async panic", func(done Done) {
		panic("an async panic")
		println("NEVER SEE THIS")
	}, 0.1)

	It("async failure with FAIL", func(done Done) {
		Fail("an async FAIL failure")
		println("NEVER SEE THIS")
	}, 0.1)

	It("FAIL in a goroutine", func(done Done) {
		go func() {
			defer GinkgoRecover()
			Fail("a goroutine FAIL failure")
			println("NEVER SEE THIS")
		}()
	}, 0.1)

	It("Gomega in a goroutine", func(done Done) {
		go func() {
			defer GinkgoRecover()
			Ω("a goroutine failure").Should(Equal("nope"))
			println("NEVER SEE THIS")
		}()
	}, 0.1)

	It("Panic in a goroutine", func(done Done) {
		go func() {
			defer GinkgoRecover()
			panic("a goroutine panic")
			println("NEVER SEE THIS")
		}()
	}, 0.1)

	Measure("a FAIL measure", func(Benchmarker) {
		Fail("a measure FAIL failure")
		println("NEVER SEE THIS")
	}, 1)

	Measure("a gomega failed measure", func(Benchmarker) {
		Ω("a measure failure").Should(Equal("nope"))
		println("NEVER SEE THIS")
	}, 1)

	Measure("a panicking measure", func(Benchmarker) {
		panic("a measure panic")
		println("NEVER SEE THIS")
	}, 1)
})

var _ = Specify("a top level specify", func() {
	Fail("fail the test")
})

var _ = DescribeTable("a top level DescribeTable",
	func(x, y int) {
		Expect(x).To(Equal(y))
	},
	Entry("a TableEntry constructed by Entry", 2, 3),
	TableEntry{
		Description: "a directly constructed TableEntry",
		Parameters:  []interface{}{2, 3},
		Pending:     false,
		Focused:     false,
	},
)
