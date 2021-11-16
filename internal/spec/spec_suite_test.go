package spec_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSpec(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spec Suite")
}
