package nolabels_test

import (
	"testing"

	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNolabels(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nolabels Suite")
}
