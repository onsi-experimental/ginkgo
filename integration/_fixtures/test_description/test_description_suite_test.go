package test_description_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTestDescription(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TestDescription Suite")
}
