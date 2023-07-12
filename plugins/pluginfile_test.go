package plugins_test

import (
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/plugins"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin files", func() {
	Describe("AddPluginPatch", func() {
		Describe("Parse", func() {
			It("parses a valid patch", func() {
				data := filebasics.MustDeserialize([]byte(`
					{
						"selectors": [
							"$.services[*]"
						],
						"overwrite": true,
						"plugins": [{
							"name": "my-plugin"
						}]
					}`))

				var pp plugins.AddPluginPatch
				err := pp.Parse(data, "test")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(pp.Selectors).To(Equal([]string{"$.services[*]"}))
				Expect(pp.Overwrite).To(BeTrue())
				Expect(pp.Plugins).To(Equal([]map[string]interface{}{
					{"name": "my-plugin"},
				}))
			})

			It("sets proper defaults", func() {
				data := filebasics.MustDeserialize([]byte(`{}`))

				var pp plugins.AddPluginPatch
				err := pp.Parse(data, "test")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(pp.Selectors).To(Equal([]string{}))
				Expect(pp.Overwrite).To(BeFalse())
				Expect(pp.Plugins).To(Equal([]map[string]interface{}{}))
			})

			It("validates selectors", func() {
				data := filebasics.MustDeserialize([]byte(`
					{
						"selectors": ["this is not a jsonpath"]
					}`))

				var pp plugins.AddPluginPatch
				err := pp.Parse(data, "test")
				Expect(err).Should(HaveOccurred())
			})
		})

		Describe("Apply", func() {
			It("applies a patch", func() {
				data := jsonbasics.ConvertToYamlNode(filebasics.MustDeserialize([]byte(`
					{
						"services": [
							{
								"name": "my-service",
								"plugins": [
									{ "name": "my-plugin" }
								]
							}
						]
					}`)))

				var pp plugins.AddPluginPatch
				err := pp.Parse(filebasics.MustDeserialize([]byte(`
					{
						"selectors": ["$.services[*]"],
						"overwrite": true,
						"plugins": [{
							"name": "my-other-plugin"
						}]
					}`)), "test")
				Expect(err).ShouldNot(HaveOccurred())

				err = pp.Apply(data)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(filebasics.MustSerialize(jsonbasics.ConvertToJSONobject(data), "JSON")).To(MatchJSON(`
					{
						"services": [
							{
								"name": "my-service",
								"plugins": [
									{ "name": "my-plugin" },
									{ "name": "my-other-plugin" }
								]
							}
						]
					}`))
			})
		})
	})
})
