package failing_ginkgo_tests_test

import (
	. "github.com/onsi-experimental/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFailing_ginkgo_tests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Failing_ginkgo_tests Suite")
}
