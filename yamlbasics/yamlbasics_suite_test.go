package yamlbasics_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestYamlbasics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "YamlBasics Suite")
}
