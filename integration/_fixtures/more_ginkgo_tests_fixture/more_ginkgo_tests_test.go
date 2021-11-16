package more_ginkgo_tests_test

import (
	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi-experimental/ginkgo/v2/integration/_fixtures/more_ginkgo_tests_fixture"
	. "github.com/onsi/gomega"
)

var _ = Describe("MoreGinkgoTests", func() {
	It("should pass", func() {
		Ω(AlwaysTrue()).Should(BeTrue())
	})

	It("should always pass", func() {
		Ω(AlwaysTrue()).Should(BeTrue())
	})
})
