package example_test

import (
	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestExample(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Example Suite")
}
