package B_test

import (
	. "github.com/onsi-experimental/ginkgo/v2/integration/_fixtures/watch_fixture/B"

	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("B", func() {
	It("should do it", func() {
		Ω(DoIt()).Should(Equal("done!"))
	})
})
