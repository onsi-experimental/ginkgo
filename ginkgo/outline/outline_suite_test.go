package outline_test

import (
	"testing"

	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOutline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Outline Suite")
}
