package failing_ginkgo_tests_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi-experimental/ginkgo/integration/_fixtures/failing_ginkgo_tests"
	. "github.com/onsi/gomega"
)

var _ = Describe("FailingGinkgoTests", func() {
	It("should fail", func() {
		Ω(AlwaysFalse()).Should(BeTrue())
	})

	It("should pass", func() {
		Ω(AlwaysFalse()).Should(BeFalse())
	})
})
