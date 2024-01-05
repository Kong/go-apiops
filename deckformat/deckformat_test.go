package deckformat_test

import (
	. "github.com/kong/go-apiops/deckformat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deckformat", func() {
	Describe("application version", func() {
		It("ToolVersionSet/Get/String", func() {
			ToolVersionSet("my-name", "1.2.3", "commit-xyz")

			n, v, c := ToolVersionGet()
			Expect(n).To(BeIdenticalTo("my-name"))
			Expect(v).To(BeIdenticalTo("1.2.3"))
			Expect(c).To(BeIdenticalTo("commit-xyz"))

			Expect(ToolVersionString()).Should(Equal("my-name 1.2.3 (commit-xyz)"))

			Expect(func() {
				ToolVersionSet("another name", "1.2.3", "commit-xyz")
			}).Should(Panic())
		})
	})
})
