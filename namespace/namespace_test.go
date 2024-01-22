package namespace_test

import (
	"strings"

	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/namespace"
	"github.com/kong/go-apiops/yamlbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
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
			err := namespace.CheckNamespace("/prefix/")
			Expect(err).ToNot(HaveOccurred())
		})
		It("appends a post-fix /", func() {
			err := namespace.CheckNamespace("/prefix")
			Expect(err).ToNot(HaveOccurred())
		})
		It("rejects a namespace without a leading /", func() {
			err := namespace.CheckNamespace("prefix/")
			Expect(err).To(HaveOccurred())
		})
		It("rejects a namespace with only a leading /", func() {
			err := namespace.CheckNamespace("/")
			Expect(err).To(HaveOccurred())
		})
		It("rejects a namespace with empty segment; //", func() {
			err := namespace.CheckNamespace("//")
			Expect(err).To(HaveOccurred())
		})
	})

	DescribeTable("UpdateSinglePathString",
		func(path, ns, expected string) {
			Expect(namespace.UpdateSinglePathString(path, ns)).To(Equal(expected))
		},
		// Test inputs:
		// "current-route-path", "namespace to apply", "final path after applying namespace"

		// plain
		Entry(nil, "/", "/namespace", "/namespace"),
		Entry(nil, "/one", "/namespace", "/namespace/one"),
		Entry(nil, "/one/", "/namespace", "/namespace/one/"),
		// regex
		Entry(nil, "~/$", "/namespace", "~/namespace$"),
		Entry(nil, "~/", "/namespace", "~/namespace"),
		Entry(nil, "~/one$", "/namespace", "~/namespace/one$"),
		Entry(nil, "~/one/$", "/namespace", "~/namespace/one/$"),

		// same but now a namespace with a trailing slash
		// plain
		Entry(nil, "/", "/namespace/", "/namespace/"), // different!!
		Entry(nil, "/one", "/namespace/", "/namespace/one"),
		Entry(nil, "/one/", "/namespace/", "/namespace/one/"),
		// regex
		Entry(nil, "~/$", "/namespace/", "~/namespace/$"), // different!!
		Entry(nil, "~/", "/namespace/", "~/namespace/"),   // different!!
		Entry(nil, "~/one$", "/namespace/", "~/namespace/one$"),
		Entry(nil, "~/one/$", "/namespace/", "~/namespace/one/$"),
	)

	DescribeTable("PreFunctions plugin Lua code",
		func(upstream_uri, ns, expected string) {
			stripFunc := namespace.GetLuaStripFunction(ns)
			L := lua.NewState()
			defer L.Close()
			err := L.DoString(`
			  local ngx = {
					var = {
						upstream_uri = "` + upstream_uri + `"
					}
				}

			` + stripFunc + `
			return ngx.var.upstream_uri`)
			Expect(err).ToNot(HaveOccurred())
			Expect(L.GetTop()).To(Equal(1))
			Expect(L.Get(1).String()).To(Equal(expected))
		},

		// Test inputs:
		// "upstream_uri set by Kong", "namespace as applied", "upstream_uri after stripping namespace"

		// namespace without trailing slash
		Entry(nil, "/namespace", "/namespace", "/"),
		Entry(nil, "/namespace/", "/namespace", "/"),
		Entry(nil, "/namespace/more", "/namespace", "/more"),
		// namespace with trailing slash
		// Entry(nil, "/namespace", "/namespace/", "/"), // TODO: cannot happen, incoming path will not match
		Entry(nil, "/namespace/", "/namespace/", "/"),
		Entry(nil, "/namespace/more", "/namespace/", "/more"),

		// same again, but now with a "service.path" set (added by kong in front of the namespace)
		// namespace without trailing slash
		Entry(nil, "/service/namespace", "/namespace", "/service"),
		Entry(nil, "/service/namespace/", "/namespace", "/service/"),
		Entry(nil, "/service/namespace/more", "/namespace", "/service/more"),
		// namespace with trailing slash
		// Entry(nil, "/service/namespace", "/namespace/", "/service/"), // TODO: cannot happen, incoming path will not match
		Entry(nil, "/service/namespace/", "/namespace/", "/service/"),
		Entry(nil, "/service/namespace/more", "/namespace/", "/service/more"),
	)

	Describe("UpdateRoute", func() {
		ns := "/namespace"
		err := namespace.CheckNamespace(ns)
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
						"/namespace"
					]
				}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("updates route with no paths, namespace trailing /", func() {
				data := `{
					"strip_path": true
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, ns+"/")

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
						"/namespace"
					]
				}`))
				Expect(needsStripping).To(BeTrue())
			})
			It("updates route with no paths, namespace trailing /", func() {
				data := `{
					"strip_path": false,
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, ns+"/")

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
		pluginObj, _ := yamlbasics.ToObject(namespace.GetPreFunctionPlugin(ns))
		pluginConf := string(filebasics.MustSerialize(pluginObj, filebasics.OutputFormatJSON))

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

		Describe("updates service.path, if the namespace matches", func() {
			Describe("path-prefix with trailing slash", func() {
				It("service.path with trailing slash", func() {
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
				It("service.path without trailing slash", func() {
					data := `{
						"services": [
							{
								"name": "service1",
								"path": "/somepath` + strings.TrimSuffix(ns, "/") + `",
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
			})
			Describe("path-prefix without trailing slash", func() {
				It("service.path with trailing slash", func() {
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
					namespace.Apply(config, selectors, strings.TrimSuffix(ns, "/"))

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
				It("service.path without trailing slash", func() {
					data := `{
						"services": [
							{
								"name": "service1",
								"path": "/somepath` + strings.TrimSuffix(ns, "/") + `",
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
					namespace.Apply(config, selectors, strings.TrimSuffix(ns, "/"))

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
			})
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
