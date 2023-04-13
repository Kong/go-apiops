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
	Describe("UpdateRoute", func() {
		It("updates plain paths", func() {
			data := `{ "paths": [
				"/one",
				"/two"
			]}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(`{
				"paths": [
					"/prefix/one",
					"/prefix/two"
				],
				"strip_path": true,
				"strip_prefix": "/prefix"
			}`))
		})

		It("updates regex paths", func() {
			data := `{ "paths": [
				"~/one$",
				"~/two$"
			]}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(`{
				"paths": [
					"~/prefix/one$",
					"~/prefix/two$"
				],
				"strip_path": true,
				"strip_prefix": "/prefix"
			}`))
		})

		It("doesn't set 'strip_prefix' if 'strip_path' was already set", func() {
			data := `{
				"paths": [
					"/one",
					"/two"
				],
				"strip_path": true
			}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(`{
				"paths": [
					"/prefix/one",
					"/prefix/two"
				],
				"strip_path": true
			}`))
		})

		It("keeps stripping existing prefixes (plain)", func() {
			data := `{
				"paths": [
					"/xyz/one",
					"/xyz/two"
				],
				"strip_prefix": "/xyz"
			}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(`{
				"paths": [
					"/prefix/xyz/one",
					"/prefix/xyz/two"
				],
				"strip_path": true,
				"strip_prefix": "/prefix/xyz"
			}`))
		})

		It("keeps stripping existing prefixes (regex)", func() {
			data := `{
				"paths": [
					"~/xyz/one$",
					"~/xyz/two$"
				],
				"strip_prefix": "/xyz"
			}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(`{
				"paths": [
					"~/prefix/xyz/one$",
					"~/prefix/xyz/two$"
				],
				"strip_path": true,
				"strip_prefix": "/prefix/xyz"
			}`))
		})

		It("skips routes where not all paths entries match", func() {
			data := `{ "paths": [
				"/one",
				"/two"
			]}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/one", "/prefix") // matches only "/one"

			Expect(toString(route)).To(MatchJSON(data))
		})

		It("skips routes with non-scalar paths-entries", func() {
			data := `{ "paths": [
				"/one",
				"/two",
				{ "this": "is an object" }
			]}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(data))
		})

		It("skips routes with paths as a non-array", func() {
			data := `{ "paths": {}}`

			route := toYaml(data)
			namespace.UpdateRoute(route, "/", "/prefix")

			Expect(toString(route)).To(MatchJSON(data))
		})
	})

	Describe("Apply", func() {
		It("handles all routes in a yamlNode", func() {
			data := `{
				"routes": [
					{
						"name": "route1",
						"paths": [
							"/one",
							"/two"
						]
					},{
						"name": "route2",
						"paths": [
							"~/xyz/one$",
							"~/xyz/two$"
						]
					}
				]
			}`

			config := toYaml(data)
			namespace.Apply(config, "/", "/prefix")

			Expect(toString(config)).To(MatchJSON(`{
				"routes": [
					{
						"name": "route1",
						"paths": [
							"/prefix/one",
							"/prefix/two"
						],
						"strip_path": true,
						"strip_prefix": "/prefix"
					},{
						"name": "route2",
						"paths": [
							"~/prefix/xyz/one$",
							"~/prefix/xyz/two$"
						],
						"strip_path": true,
						"strip_prefix": "/prefix"
					}
				]
			}`))
		})
	})
})
