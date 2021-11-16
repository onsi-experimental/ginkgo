package D_test

import (
	. "github.com/onsi-experimental/ginkgo/v2/integration/_fixtures/watch_fixture/D"

	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("D", func() {
	It("should do it", func() {
		Ω(DoIt()).Should(Equal("done!"))
	})
})
