package containernode_test

import (
	. "github.com/onsi-experimental/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestContainernode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Containernode Suite")
}
