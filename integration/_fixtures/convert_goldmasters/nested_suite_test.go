package nested_test

import (
	"testing"

	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNested(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nested Suite")
}
