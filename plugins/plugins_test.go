package plugins_test

import (
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/plugins"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugins", func() {
	Describe("Plugger.SetData", func() {
		It("panics if data is nil", func() {
			Expect(func() {
				plugger := plugins.Plugger{}
				plugger.SetData(nil)
			}).Should(PanicWith("data cannot be nil"))
		})

		It("does not panic if data is not nil", func() {
			Expect(func() {
				plugger := plugins.Plugger{}
				plugger.SetData(map[string]interface{}{})
			}).ShouldNot(Panic())
		})
	})

	Describe("Plugger.SetSelectors", func() {
		It("allows nil, or 0 length", func() {
			plugger := plugins.Plugger{}

			err := plugger.SetSelectors(nil)
			Expect(err).To(BeNil())

			err = plugger.SetSelectors([]string{})
			Expect(err).To(BeNil())
		})

		It("accepts a valid JSONpointer", func() {
			plugger := plugins.Plugger{}
			err := plugger.SetSelectors([]string{"$..routes[*]", "$..services[*]"})
			Expect(err).To(BeNil())
		})

		It("fails on a bad JSONpointer", func() {
			plugger := plugins.Plugger{}
			err := plugger.SetSelectors([]string{"bad one"})
			Expect(err).To(MatchError("selector 'bad one' is not a valid JSONpath expression; " +
				"invalid character ' ' at position 3, following \"bad\""))
		})
	})

	Describe("Plugger.AddPlugin", func() {
		It("adds plugin to existing plugin arrays and creates new plugin arrays", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" },
					{ "name": "service2",
					  "plugins": [
						  { "name": "plugin1" }
						]}
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(&dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name": "plugin-added",
			}, false)

			result := *(filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON))
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{
							"name": "service1",
							"plugins": [
								{ "name": "plugin-added" }
							]
						}, {
							"name": "service2",
							"plugins": [
								{ "name": "plugin1" },
								{ "name": "plugin-added" }
							]
						}
					]
			  }`))
		})

		It("only adds to plugin-arrays based on the selector", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" },
					{ "name": "service2",
					  "plugins": [
						  { "name": "plugin1" }
						]}
				],
					"routes": [
					{ "name": "route1" },
					{ "name": "route2",
					  "plugins": [
							{ "name": "plugin2" }
					  ]}
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(&dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name": "plugin-added",
			}, false)

			result := *(filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON))
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{
							"name": "service1",
							"plugins": [
								{ "name": "plugin-added" }
							]
						}, {
							"name": "service2",
							"plugins": [
								{ "name": "plugin1" },
								{ "name": "plugin-added" }
							]
						}
					],
					"routes": [
						{
							"name": "route1"
						}, {
							"name": "route2",
							"plugins": [
								{ "name": "plugin2" }
							]
					}
				]
			}`))
		})

		It("overwrites plugins if set to do so", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" },
					{ "name": "service2",
					  "plugins": [
						  { "name": "plugin1" }
						]}
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(&dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name":   "plugin1",
				"plugin": "was added",
			}, true)

			result := *(filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON))
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{
							"name": "service1",
							"plugins": [
								{ "name": "plugin1", "plugin": "was added" }
							]
						}, {
							"name": "service2",
							"plugins": [
								{ "name": "plugin1", "plugin": "was added" }
							]
						}
					]
			}`))
		})
	})
})
