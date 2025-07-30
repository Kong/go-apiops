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
				"Error at line 1, column 0: expected '$'\nbad one\n^..\n"))
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
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name": "plugin-added",
			}, false)

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
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

		It("supports complex selectors", func() {
			dataInput := []byte(`{
  "services": [
    {
      "host": "api.example.com",
      "name": "example",
      "path": "/",
      "plugins": [],
      "port": 443,
      "protocol": "https",
      "routes": [
        {
          "name": "customer_post",
          "methods": [ "POST" ],
          "paths": [
            "~/accounts/customers$"
          ],
          "plugins": []
        },
        {
          "name": "demo_one_get",
          "methods": [ "GET" ],
          "paths": [
            "~/path/to/certificates$"
          ],
          "plugins": []
        },
        {
          "name": "demo_two_get",
          "methods": [ "OPTIONS", "GET" ],
          "paths": [
            "~/path/to/schedules$"
          ],
          "plugins": []
        },
        {
          "name": "demo_three_delete",
          "methods": [ "DELETE" ],
          "paths": [
            "~/path/to/schedules$"
          ],
          "plugins": []
        }
      ]
    }
  ],
}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$.services[*].routes[?(@.methods[?(@=='GET' ) || (@=='POST')])]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name": "plugin-added",
			}, false)

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`{
  "services": [
    {
      "host": "api.example.com",
      "name": "example",
      "path": "/",
      "plugins": [],
      "port": 443,
      "protocol": "https",
      "routes": [
        {
          "name": "customer_post",
          "methods": [ "POST" ],
          "paths": [
            "~/accounts/customers$"
          ],
          "plugins": [{
            "name": "plugin-added"
          }]
        },
        {
          "name": "demo_one_get",
          "methods": [ "GET" ],
          "paths": [
            "~/path/to/certificates$"
          ],
          "plugins": [{
            "name": "plugin-added"
          }]
        },
        {
          "name": "demo_two_get",
          "methods": [ "OPTIONS", "GET" ],
          "paths": [
            "~/path/to/schedules$"
          ],
          "plugins": [{
            "name": "plugin-added"
          }]
        },
        {
          "name": "demo_three_delete",
          "methods": [ "DELETE" ],
          "paths": [
            "~/path/to/schedules$"
          ],
          "plugins": []
        }
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
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name": "plugin-added",
			}, false)

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
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
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			plugger.AddPlugin(map[string]interface{}{
				"name":   "plugin1",
				"plugin": "was added",
			}, true)

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
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

		It("adds plugin with foreign keys to the main array", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":           "plugin1",
				"service":        "service1",
				"route":          "route1",
				"consumer":       "consumer1",
				"consumer_group": "consumer_group1",
			}, true)
			Expect(err).ToNot(HaveOccurred())

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{
							"name": "service1"
						}
					],
					"plugins": [
						{
							"name": "plugin1",
							"service": "service1",
							"route": "route1",
							"consumer": "consumer1",
							"consumer_group": "consumer_group1"
						}
					]
			}`))
		})

		It("adds plugin with non-matching foreign keys in the main array", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				],
				"plugins": [
					{
						"name": "plugin1",
						"service": "service1",
						"route": "route1",
						"consumer": "consumer1",
						"consumer_group": "consumer_group1"
					}
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":           "plugin1",
				"service":        "service2", // this one is different
				"route":          "route1",
				"consumer":       "consumer1",
				"consumer_group": "consumer_group1",
			}, true)
			Expect(err).ToNot(HaveOccurred())

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{
							"name": "service1"
						}
					],
					"plugins": [
						{
							"name": "plugin1",
							"service": "service1",
							"route": "route1",
							"consumer": "consumer1",
							"consumer_group": "consumer_group1"
						},
						{
							"name": "plugin1",
							"service": "service2",
							"route": "route1",
							"consumer": "consumer1",
							"consumer_group": "consumer_group1"
						}
					]
			}`))
		})

		It("overwrites plugin with matching foreign keys in the main array", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				],
				"plugins": [
					{
						"name": "plugin1",
						"service": "service1",
						"route": "route1",
						"consumer": "consumer1",
						"consumer_group": "consumer_group1"
					}
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":           "plugin1",
				"service":        "service1",
				"route":          "route1",
				"consumer":       "consumer1",
				"consumer_group": "consumer_group1",
				"plugin":         "was overwritten",
			}, true)
			Expect(err).ToNot(HaveOccurred())

			result := filebasics.MustSerialize(plugger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{
							"name": "service1"
						}
					],
					"plugins": [
						{
							"name": "plugin1",
							"service": "service1",
							"route": "route1",
							"consumer": "consumer1",
							"consumer_group": "consumer_group1",
							"plugin": "was overwritten"
						}
					]
			}`))
		})

		It("fails to add plugin to nested array with service foreign key", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":    "plugin1",
				"service": "some_name",
			}, true)
			Expect(err).To(MatchError("plugin 0 has foreign keys, but they are only supported in the main plugin array"))
		})

		It("fails to add plugin to nested array with route foreign key", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":  "plugin1",
				"route": "some_name",
			}, true)
			Expect(err).To(MatchError("plugin 0 has foreign keys, but they are only supported in the main plugin array"))
		})

		It("fails to add plugin to nested array with consumer foreign key", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":     "plugin1",
				"consumer": "some_name",
			}, true)
			Expect(err).To(MatchError("plugin 0 has foreign keys, but they are only supported in the main plugin array"))
		})

		It("fails to add plugin to nested array with consumer_group foreign key", func() {
			dataInput := []byte(`
				{ "services": [
					{ "name": "service1" }
				]
			}`)

			plugger := plugins.Plugger{}
			plugger.SetData(filebasics.MustDeserialize(dataInput))
			plugger.SetSelectors([]string{
				"$..services[*]",
			})
			err := plugger.AddPlugin(map[string]interface{}{
				"name":           "plugin1",
				"consumer_group": "some_name",
			}, true)
			Expect(err).To(MatchError("plugin 0 has foreign keys, but they are only supported in the main plugin array"))
		})
	})
})
