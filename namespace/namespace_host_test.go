package namespace_test

import (
	"github.com/kong/go-apiops/namespace"
	"github.com/kong/go-apiops/yamlbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Host-Namespace", func() {
	Describe("ApplyNamespaceHost", func() {
		// clear == fasle/true
		// hosts exists/not exists
		// hosts has a name, has no name
		// hosts has the namespace (eg adding duplicate)
		Describe("clear hosts", func() {
			It("clears the hosts", func() {
				data := `{
					"routes": [
						{
							"hosts": ["one", "two"]
						}
					]
				}`
				deckfile := toYaml(data)
				hosts := []string{"three"}
				err := namespace.ApplyNamespaceHost(deckfile, yamlbasics.SelectorSet{}, hosts, true, false)
				Expect(err).To(BeNil())
				Expect(toString(deckfile)).To(MatchJSON(`{
					"routes": [
						{
							"hosts": ["three"]
						}
					]
				}`))
			})
			It("clears the hosts, no hosts", func() {
				data := `{
					"routes": [
						{
							"paths": []
						}
					]
				}`
				deckfile := toYaml(data)
				hosts := []string{"three"}
				err := namespace.ApplyNamespaceHost(deckfile, yamlbasics.SelectorSet{}, hosts, true, false)
				Expect(err).To(BeNil())
				Expect(toString(deckfile)).To(MatchJSON(`{
					"routes": [
						{
							"paths": [],
							"hosts": ["three"]
						}
					]
				}`))
			})
		})
	})
	Describe("appends hosts", func() {
		It("Route without hosts array", func() {
			data := `{
				"routes": [
					{
						"paths": []
					}
				]
			}`
			deckfile := toYaml(data)
			hosts := []string{"three"}
			err := namespace.ApplyNamespaceHost(deckfile, yamlbasics.SelectorSet{}, hosts, false, false)
			Expect(err).To(BeNil())
			Expect(toString(deckfile)).To(MatchJSON(`{
				"routes": [
					{
						"paths": [],
						"hosts": ["three"]
					}
				]
			}`))
		})
		It("Route with empty hosts array", func() {
			data := `{
				"routes": [
					{
						"hosts": []
					}
				]
			}`
			deckfile := toYaml(data)
			hosts := []string{"three"}
			err := namespace.ApplyNamespaceHost(deckfile, yamlbasics.SelectorSet{}, hosts, false, false)
			Expect(err).To(BeNil())
			Expect(toString(deckfile)).To(MatchJSON(`{
				"routes": [
					{
						"hosts": ["three"]
					}
				]
			}`))
		})
		It("adds hosts", func() {
			data := `{
				"routes": [
					{
						"hosts": ["one", "two"]
					}
				]
			}`
			deckfile := toYaml(data)
			hosts := []string{"three"}
			err := namespace.ApplyNamespaceHost(deckfile, yamlbasics.SelectorSet{}, hosts, false, false)
			Expect(err).To(BeNil())
			Expect(toString(deckfile)).To(MatchJSON(`{
				"routes": [
					{
						"hosts": ["one", "two", "three"]
					}
				]
			}`))
		})
		It("doesn't add duplicate hosts", func() {
			data := `{
				"routes": [
					{
						"hosts": ["one", "two"]
					}
				]
			}`
			deckfile := toYaml(data)
			hosts := []string{"one", "two", "three"}
			err := namespace.ApplyNamespaceHost(deckfile, yamlbasics.SelectorSet{}, hosts, false, false)
			Expect(err).To(BeNil())
			Expect(toString(deckfile)).To(MatchJSON(`{
				"routes": [
					{
						"hosts": ["one", "two", "three"]
					}
				]
			}`))
		})
	})
})
