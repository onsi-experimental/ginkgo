package suite_command_test

import (
	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing", func() {
	It("it should succeed", func() {
		Ω(true).Should(Equal(true))
	})

	PIt("a failing test", func() {
		It("should fail", func() {
			Ω(true).Should(Equal(false))
		})
	})
})
