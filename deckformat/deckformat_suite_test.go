package deckformat_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeckformat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deckformat Suite")
}
