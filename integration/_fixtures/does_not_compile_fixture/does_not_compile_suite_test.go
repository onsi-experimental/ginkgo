package does_not_compile_test

import (
	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDoes_not_compile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Does_not_compile Suite")
}
