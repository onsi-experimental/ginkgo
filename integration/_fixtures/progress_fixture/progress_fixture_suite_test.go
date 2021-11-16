package progress_fixture_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestProgressFixture(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ProgressFixture Suite")
}
