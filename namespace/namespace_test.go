package namespace_test

import (
	"github.com/kong/go-apiops/namespace"
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

	Describe("CheckPrefix", func() {
		It("validates a plain prefix", func() {
			prefix, err := namespace.CheckPrefix("/prefix/")
			Expect(err).ToNot(HaveOccurred())
			Expect(prefix).To(Equal("/prefix/"))
		})
		It("rejects a prefix without a leading /", func() {
			_, err := namespace.CheckPrefix("prefix/")
			Expect(err).To(HaveOccurred())
		})
		It("rejects a regex prefix with a leading ~", func() {
			_, err := namespace.CheckPrefix("~/prefix/")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("UpdateSinglePathString", func() {
		ns, err := namespace.CheckNamespace("/namespace")
		if err != nil {
			panic(err)
		}
		It("updates plain paths", func() {
			Expect(namespace.UpdateSinglePathString("/one", "", ns)).To(Equal("/namespace/one"))
			Expect(namespace.UpdateSinglePathString("/one", "/", ns)).To(Equal("/namespace/one"))
			Expect(namespace.UpdateSinglePathString("/one", "/one", ns)).To(Equal("/namespace/one"))
			Expect(namespace.UpdateSinglePathString("/one", "/two", ns)).To(Equal("/one"))
		})
		It("updates empty paths", func() {
			Expect(namespace.UpdateSinglePathString("/", "", ns)).To(Equal("/namespace/"))
			Expect(namespace.UpdateSinglePathString("/", "/", ns)).To(Equal("/namespace/"))
			Expect(namespace.UpdateSinglePathString("/", "/one", ns)).To(Equal("/"))
		})
		It("updates regex paths", func() {
			Expect(namespace.UpdateSinglePathString("~/demo/(?<something>[^#?/]+)/else$", "",
				ns)).To(Equal("~/namespace/demo/(?<something>[^#?/]+)/else$"))
			Expect(namespace.UpdateSinglePathString("~/demo/(?<something>[^#?/]+)/else$", "/",
				ns)).To(Equal("~/namespace/demo/(?<something>[^#?/]+)/else$"))
			Expect(namespace.UpdateSinglePathString("~/demo/(?<something>[^#?/]+)/else$", "/demo",
				ns)).To(Equal("~/namespace/demo/(?<something>[^#?/]+)/else$"))
			Expect(namespace.UpdateSinglePathString("~/demo/(?<something>[^#?/]+)/else$", "/two",
				ns)).To(Equal("~/demo/(?<something>[^#?/]+)/else$"))
		})
	})

	Describe("UpdateRoute", func() {
		ns, err := namespace.CheckNamespace("/namespace")
		if err != nil {
			panic(err)
		}

		Describe("strip_path == true", func() {
			It("updates plain paths, matching all, no prefix", func() {
				data := `{
					"strip_path": true,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": true,
					"paths": [
						"/namespace/one",
						"/namespace/two"
					]
				}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("updates plain paths, matching all", func() {
				data := `{
					"strip_path": true,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "/", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": true,
					"paths": [
							"/namespace/one",
							"/namespace/two"
						]
					}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("updates plain paths, matching some", func() {
				data := `{
					"strip_path": true,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "/one", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": true,
					"paths": [
						"/namespace/one",
						"/two"
					]
				}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("doesn't update route with no paths, prefix '/'", func() {
				data := `{
					"strip_path": true
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "/", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": true
				}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("updates route with no paths; no prefix", func() {
				data := `{
					"strip_path": true
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "", ns)

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
			It("updates plain paths, matching all, no prefix", func() {
				data := `{
					"strip_path": false,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": false,
					"paths": [
						"/namespace/one",
						"/namespace/two"
					]
				}`))
				Expect(needsStripping).To(BeTrue())
			})
			It("updates plain paths, matching all", func() {
				data := `{
					"strip_path": false,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "/", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": false,
					"paths": [
							"/namespace/one",
							"/namespace/two"
						]
					}`))
				Expect(needsStripping).To(BeTrue())
			})
			It("updates plain paths, matching some", func() {
				data := `{
					"strip_path": false,
					"paths": [
						"/one",
						"/two"
					]}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "/one", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": false,
					"paths": [
						"/namespace/one",
						"/two"
					]
				}`))
				Expect(needsStripping).To(BeTrue())
			})
			It("doesn't update route with no paths, prefix '/'", func() {
				data := `{
					"strip_path": false,
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "/", ns)

				Expect(toString(route)).To(MatchJSON(`{
					"strip_path": false
				}`))
				Expect(needsStripping).To(BeFalse())
			})
			It("updates route with no paths; no prefix", func() {
				data := `{
					"strip_path": false,
				}`

				route := toYaml(data)
				needsStripping := namespace.UpdateRoute(route, "", ns)

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

		It("handles plain and regex (no prefix-match)", func() {
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
			namespace.Apply(config, "", ns)

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

		It("handles routes without paths, strip=true (no prefix-match)", func() {
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
			namespace.Apply(config, "", ns)

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

		It("handles routes without paths, strip=false (no prefix-match)", func() {
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
			namespace.Apply(config, "", ns)

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

		It("handles plain and regex, strip=true (prefix-match = '/')", func() {
			data := `{
				"routes": [
					{
						"name": "route1",
						"strip_path": true,
						"paths": [
							"/one",
							"/two"
						]
					},{
						"name": "route2",
						"strip_path": true,
						"paths": [
							"~/xyz/one$",
							"~/xyz/two$"
						]
					}
				]
			}`

			config := toYaml(data)
			namespace.Apply(config, "/", ns)

			Expect(toString(config)).To(MatchJSON(`{
				"routes": [
					{
						"name": "route1",
						"strip_path": true,
						"paths": [
							"` + ns + `one",
							"` + ns + `two"
						]
					},
					{
						"name": "route2",
						"strip_path": true,
						"paths": [
							"~` + ns + `xyz/one$",
							"~` + ns + `xyz/two$"
						]
					}
				]
			}`))
		})

		It("handles plain and regex, strip=false (prefix-match = '/')", func() {
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
			namespace.Apply(config, "/", ns)

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
					},
					{
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

		It("doesn't handle routes without paths (prefix-match = '/')", func() {
			data := `{
				"routes": [
					{
						"name": "routeA",
						"hosts": [
							"acme-corp.com"
						]
					}
				]
			}`

			config := toYaml(data)
			namespace.Apply(config, "/", ns)

			Expect(toString(config)).To(MatchJSON(`{
				"routes": [
					{
						"name": "routeA",
						"hosts": [
							"acme-corp.com"
						]
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
			namespace.Apply(config, "", ns) // the "" prefix will match all routes

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
			namespace.Apply(config, "/", ns) // the "/" prefix will not match routeA

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
