package openapi2kong_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOpenapi2kong(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenAPI2Kong Suite")
}
