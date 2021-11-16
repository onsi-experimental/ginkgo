package example_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi-experimental/ginkgo/extensions/table"
)

var _ = Describe("PendingFixture", func() {
	PDescribe("pending", func() {
		It("pending", func() {
			By("pending")
			By("pending")
		})
	})

	PContext("pending", func() {
		It("pending", func() {

		})
	})

	PWhen("pending", func() {
		It("pending", func() {

		})
	})

	PIt("pending", func() {

	})

	PSpecify("pending", func() {

	})

	PMeasure("pending", func(b Benchmarker) {

	}, 2)

	PDescribeTable("pending",
		func() {},
		Entry("pending"),
	)

	DescribeTable("pending",
		func() {},
		PEntry("pending"),
	)
})
