package interrupt_handler_test

import (
	"testing"

	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInterrupthandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interrupthandler Suite")
}
