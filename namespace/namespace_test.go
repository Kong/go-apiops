package namespace_test

import (
	"github.com/kong/go-apiops/namespace"
	"github.com/kong/go-apiops/yamlbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

func toYaml(data string) *yaml.Node {
	var yNode yaml.Node
	err := yaml.Unmarshal([]byte(data), &yNode)
	if err != nil {
		panic(err)
	}
	return yNode.Content[0] // first entry is the node, yNode is the document
}

func toString(data *yaml.Node) string {
	out, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(out)
}

var _ = Describe("Namespace", func() {
	Describe("CheckNamespace", func() {
		It("validates a plain namespace", func() {
			prefix, err := namespace.CheckNamespace("/prefix/")
			Expect(err).ToNot(HaveOccurred())
			Expect(prefix).To(Equal("/prefix/"))
		})
		It("appends a post-fix /", func() {
			prefix, err := namespace.CheckNamespace("/prefix")
			Expect(err).ToNot(HaveOccurred())
			Expect(prefix).To(Equal("/prefix/"))
		})
		It("rejects a namespace without a leading /", func() {
			_, err := namespace.CheckNamespace("prefix/")
			Expect(err).To(HaveOccurred())
		})
		It("rejects a namespace with only a leading /", func() {
			_, err := namespace.CheckNamespace("/")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("UpdateSinglePathString", func() {
		ns, err := namespace.CheckNamespace("/namespace")
		if err != nil {
			panic(err)
		}
		It("updates plain paths", func() {
			Expect(namespace.UpdateSinglePathString("/one", ns)).To(Equal("/namespace/one"))
		})
		It("updates empty paths", func() {
			Expect(namespace.UpdateSinglePathString("/", ns)).To(Equal("/namespace/"))
		})
		It("updates regex paths", func() {
			Expect(namespace.UpdateSinglePathString("~/demo/(?<something>[^#?/]+)/else$",
				ns)).To(Equal("~/namespace/demo/(?<something>[^#?/]+)/else$"))
		})
	})

	Describe("UpdateRoute", func() {
		ns, err := namespace.CheckNamespace("/namespace")
		if err != nil {
			panic(err)
		}

		Describe("strip_path == true", func() {
			It("updates plain paths", func() {
				data := `{
					"strip_path": true,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": true,
					"paths": [
						"/namespace/one",
						"/namespace/two"
					]
				}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("updates route with no paths", func() {
				data := `{
					"strip_path": true
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": true,
					"paths": [
						"/namespace/"
					]
				}`))
				Expect(needsStripping).To(BeFalse())
			})
		})

		Describe("strip_path == false", func() {
			It("updates plain paths", func() {
				data := `{
					"strip_path": false,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": false,
					"paths": [
						"/namespace/one",
						"/namespace/two"
					]
				}`))
				Expect(needsStripping).To(BeTrue())
			})
			It("updates route with no paths", func() {
				data := `{
					"strip_path": false,
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": false,
					"paths": [
						"/namespace/"
					]
				}`))
				Expect(needsStripping).To(BeTrue())
			})
		})
	})

	Describe("Apply", func() {
		ns := "/my-namespace/"
		pluginConf := `{
			"config": {
				"access": [
					"local u,s,e=ngx.var.upstream_uri s,e=u:find('` + ns +
			`',1,true)ngx.var.upstream_uri=u:sub(1,s)..u:sub(e+1,-1)"
				]
			},
			"name": "pre-function"
		}`

		//
		// first we check proper conversions, and injection of the plugin
		//

		It("handles plain and regex", func() {
			data := `{
				"routes": [
					{
						"name": "route1",
						"strip_path": false,
						"paths": [
							"/one",
							"/two"
						]
					},{
						"name": "route2",
						"strip_path": false,
						"paths": [
							"~/xyz/one$",
							"~/xyz/two$"
						]
					}
				]
			}`

			config := toYaml(data)
			selectors, err := yamlbasics.NewSelectorSet(nil)
			if err != nil {
				panic(err)
			}
			namespace.Apply(config, selectors, ns)

			Expect(toString(config)).To(MatchJSON(`{
				"routes": [
					{
						"name": "route1",
						"strip_path": false,
						"paths": [
							"` + ns + `one",
							"` + ns + `two"
						],
						"plugins": [` + pluginConf + `]
					},{
						"name": "route2",
						"strip_path": false,
						"paths": [
							"~` + ns + `xyz/one$",
							"~` + ns + `xyz/two$"
						],
						"plugins": [` + pluginConf + `]
					}
				]
			}`))
		})

		It("handles routes without paths, strip=true", func() {
			data := `{
				"routes": [
					{
						"name": "routeA",
						"strip_path": true,
						"hosts": [
							"acme-corp.com"
						]
					}
				]
			}`

			config := toYaml(data)
			selectors, err := yamlbasics.NewSelectorSet(nil)
			if err != nil {
				panic(err)
			}
			namespace.Apply(config, selectors, ns)

			Expect(toString(config)).To(MatchJSON(`{
        "routes": [
          {
            "name": "routeA",
						"hosts": [
							"acme-corp.com"
						],
						"strip_path": true,
            "paths": [
              "` + ns + `"
            ]
          }
        ]
			}`))
		})

		It("handles routes without paths, strip=false", func() {
			data := `{
				"routes": [
					{
						"name": "routeA",
						"strip_path": false,
						"hosts": [
							"acme-corp.com"
						]
					}
				]
			}`

			config := toYaml(data)
			selectors, err := yamlbasics.NewSelectorSet(nil)
			if err != nil {
				panic(err)
			}
			namespace.Apply(config, selectors, ns)

			Expect(toString(config)).To(MatchJSON(`{
        "routes": [
          {
            "name": "routeA",
						"hosts": [
							"acme-corp.com"
						],
						"strip_path": false,
            "paths": [
              "` + ns + `"
            ],
            "plugins": [` + pluginConf + `]
          }
        ]
			}`))
		})

		//
		// check plugin position; on service or route
		//

		It("attaches a plugin to the service, if all routes match", func() {
			data := `{
				"services": [
					{
						"name": "service1",
						"routes": [
							{
								"name": "route1",
								"strip_path": false,
								"paths": [
									"/one"
								]
							},{
								"name": "route2",
								"strip_path": false,
								"paths": [
									"~/xyz/two$"
								]
							}
						]
					}
				]
			}`

			config := toYaml(data)
			selectors, err := yamlbasics.NewSelectorSet(nil)
			if err != nil {
				panic(err)
			}
			namespace.Apply(config, selectors, ns)

			Expect(toString(config)).To(MatchJSON(`{
        "services": [
          {
            "name": "service1",
            "routes": [
              {
                "name": "route1",
								"strip_path": false,
                "paths": [
									"` + ns + `one"
								]
              },
              {
                "name": "route2",
								"strip_path": false,
                "paths": [
									"~` + ns + `xyz/two$"
								]
              }
            ],
						"plugins": [` + pluginConf + `]
          }
        ]
			}`))
		})

		It("updates service.path, if the namespace matches", func() {
			data := `{
				"services": [
					{
						"name": "service1",
						"path": "/somepath` + ns + `",
						"routes": [
							{
								"name": "route1",
								"strip_path": false,
								"paths": [
									"/one"
								]
							},{
								"name": "route2",
								"strip_path": false,
								"paths": [
									"~/xyz/two$"
								]
							}
						]
					}
				]
			}`

			config := toYaml(data)
			selectors, err := yamlbasics.NewSelectorSet(nil)
			if err != nil {
				panic(err)
			}
			namespace.Apply(config, selectors, ns)

			Expect(toString(config)).To(MatchJSON(`{
        "services": [
          {
            "name": "service1",
						"path": "/somepath/",
            "routes": [
              {
                "name": "route1",
								"strip_path": false,
                "paths": [
									"` + ns + `one"
								]
              },
              {
                "name": "route2",
								"strip_path": false,
                "paths": [
									"~` + ns + `xyz/two$"
								]
              }
            ]
          }
        ]
			}`))
		})

		It("attaches a plugin to routes, if not all routes match", func() {
			data := `{
				"services": [
					{
						"name": "service1",
						"routes": [
							{
								"name": "routeA",
								"strip_path": false,
								"hosts": [
									"acme-corp.com"
								]
							},{
								"name": "route1",
								"strip_path": false,
								"paths": [
									"/one"
								]
							}
						]
					}
				]
			}`

			config := toYaml(data)
			selector := "$.services[*].routes[1]" // only matches 1 of the 2 routes
			selectors, err := yamlbasics.NewSelectorSet([]string{selector})
			if err != nil {
				panic(err)
			}
			namespace.Apply(config, selectors, ns)

			Expect(toString(config)).To(MatchJSON(`{
        "services": [
          {
            "name": "service1",
            "routes": [
              {
                "name": "routeA",
								"strip_path": false,
                "hosts": [
									"acme-corp.com"
								]
              },{
                "name": "route1",
								"strip_path": false,
                "paths": [
									"` + ns + `one"
								],
								"plugins": [` + pluginConf + `]
              }
            ]
          }
        ]
			}`))
		})
	})
})
