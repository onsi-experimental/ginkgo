package spec_iterator_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSpecIterator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpecIterator Suite")
}
