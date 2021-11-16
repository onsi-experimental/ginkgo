package reporters_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReporters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reporters Suite")
}
