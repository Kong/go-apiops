package openapi2mcp

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOpenapi2mcp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Openapi2mcp Suite")
}
