package patch_test

import (
	. "github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/patch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Patch", func() {
	Describe("Parse", func() {
		It("parses a valid patch", func() {
			jsonData := []byte(`{
				"selectors": ["$"],
				"values": {
					"field1": "value1"
				},
				"remove": [ "field2" ]
			}`)
			data := MustDeserialize(jsonData)

			var patch patch.DeckPatch
			err := patch.Parse(data, "breadcrumb-text")

			Expect(err).To(BeNil())
			Expect(patch.SelectorSources).To(BeEquivalentTo([]string{"$"}))
			Expect(patch.Selectors).ToNot(BeNil())
			Expect(patch.Values).To(BeEquivalentTo(map[string]interface{}{
				"field1": "value1",
			}))
			Expect(patch.Remove).To(BeEquivalentTo([]string{
				"field2",
			}))
		})

		It("fails on non-string-array selector", func() {
			jsonData := []byte(`{
				"selectors": 123
			}`)
			data := MustDeserialize(jsonData)

			var patch patch.DeckPatch
			err := patch.Parse(data, "file1.yml:patches[1]")

			Expect(err).To(MatchError("file1.yml:patches[1].selectors is not a string-array"))
		})

		It("fails on bad selector", func() {
			jsonData := []byte(`{
				"selectors": ["not valid"]
			}`)
			data := MustDeserialize(jsonData)

			var patch patch.DeckPatch
			err := patch.Parse(data, "file1.yml:patches[1]")

			Expect(err).To(MatchError("file1.yml:patches[1].selectors[0] is not a valid JSONpath " +
				"expression; invalid character ' ' at position 3, following \"not\""))
		})

		It("fails on non-object 'values'", func() {
			jsonData := []byte(`{
				"selectors": ["$"],
				"values": 123
			}`)
			data := MustDeserialize(jsonData)

			var patch patch.DeckPatch
			err := patch.Parse(data, "file1.yml:patches[1]")

			Expect(err).To(MatchError("file1.yml:patches[1].values is not an object"))
		})

		It("fails on non-array 'remove'", func() {
			jsonData := []byte(`{
				"selectors": ["$"],
				"remove": 123
			}`)
			data := MustDeserialize(jsonData)

			var patch patch.DeckPatch
			err := patch.Parse(data, "file1.yml:patches[1]")

			Expect(err).To(MatchError("file1.yml:patches[1].remove is not an array"))
		})

		It("fails on changing and removing the same field", func() {
			jsonData := []byte(`{
				"selectors": ["$"],
				"values": {
					"field1": "value1"
				},
				"remove": [ "field1" ]
			}`)
			data := MustDeserialize(jsonData)

			var patch patch.DeckPatch
			err := patch.Parse(data, "file1.yml:patches[1]")

			Expect(err).To(MatchError("file1.yml:patches[1] is trying to change and remove 'field1' at the same time"))
		})
	})

	Describe("validating --value flags", func() {
		It("validates a number", func() {
			val, rem, err := patch.ValidateValuesFlags([]string{"key1:1", "key2:2"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": 1.0,
				"key2": 2.0,
			}))
			Expect(rem).To(BeEquivalentTo([]string{}))
		})

		It("validates a string", func() {
			val, rem, err := patch.ValidateValuesFlags([]string{"key1:\"hi\"", "key2:\"there\""})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": "hi",
				"key2": "there",
			}))
			Expect(rem).To(BeEquivalentTo([]string{}))
		})

		It("validates a boolean", func() {
			val, rem, err := patch.ValidateValuesFlags([]string{"key1:true", "key2:false"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": true,
				"key2": false,
			}))
			Expect(rem).To(BeEquivalentTo([]string{}))
		})

		It("validates an object", func() {
			val, rem, err := patch.ValidateValuesFlags([]string{"key1:{\"hello\": 123}", "key2:false"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": map[string]interface{}{
					"hello": 123.0,
				},
				"key2": false,
			}))
			Expect(rem).To(BeEquivalentTo([]string{}))
		})

		It("validates an array", func() {
			val, rem, err := patch.ValidateValuesFlags([]string{"key1:[true]", "key2:false"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": []interface{}{true},
				"key2": false,
			}))
			Expect(rem).To(BeEquivalentTo([]string{}))
		})

		It("validates an empty value (delete the key)", func() {
			val, rem, err := patch.ValidateValuesFlags([]string{"key1:", "key2:"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{}))
			Expect(rem).To(BeEquivalentTo([]string{"key1", "key2"}))
		})

		Describe("returns error on", func() {
			It("missing ':'", func() {
				val, rem, err := patch.ValidateValuesFlags([]string{"key1:true", "key2_false"})

				Expect(err).To(MatchError("expected '--value' entry to have format 'key:json-string', got: 'key2_false'"))
				Expect(val).To(BeNil())
				Expect(rem).To(BeNil())
			})

			It("invalid JSON", func() {
				val, rem, err := patch.ValidateValuesFlags([]string{"key1:true", "key2:{not valid this stuff...}"})

				Expect(err).To(MatchError("expected '--value' entry to have format 'key:json-string', failed parsing " +
					"json-string in 'key2:{not valid this stuff...}' (did you forget to wrap a json-string-value in quotes?)"))
				Expect(val).To(BeNil())
				Expect(rem).To(BeNil())
			})
		})

		Describe("allows for", func() {
			It("multiple ':' characters (splits only by first one)", func() {
				val, rem, err := patch.ValidateValuesFlags([]string{"key1:\":::\""})

				Expect(err).To(BeNil())
				Expect(val).To(BeEquivalentTo(map[string]interface{}{
					"key1": ":::",
				}))
				Expect(rem).To(BeEquivalentTo([]string{}))
			})
		})
	})

	Describe("validating --selector flags", func() {
		It("returns error on bad JSONpath", func() {
			testPatch := patch.DeckPatch{
				SelectorSources: []string{"bad JSONpath"},
				Values:          nil,
				Remove:          []string{"test"},
			}
			data := []byte(`{}`)
			err := testPatch.ApplyToNodes(jsonbasics.ConvertToYamlNode(MustDeserialize(data)))
			Expect(err).To(MatchError("selector 'bad JSONpath' is not a valid JSONpath expression; " +
				"invalid character ' ' at position 3, following \"bad\""))
		})
	})

	Describe("Applying values", func() {
		applyUpdates := func(data []byte, selector string, valueFlags []string) []byte {
			jsonData := MustDeserialize(data)
			parsedValues, remove, err := patch.ValidateValuesFlags(valueFlags)
			Expect(err).To(BeNil())

			testPatch := patch.DeckPatch{
				SelectorSources: []string{selector},
				Values:          parsedValues,
				Remove:          remove,
			}

			yamlNode := jsonbasics.ConvertToYamlNode(jsonData)
			err = testPatch.ApplyToNodes(yamlNode)
			Expect(err).To(BeNil())

			updated := jsonbasics.ConvertToJSONobject(yamlNode)
			result := MustSerialize(updated, OutputFormatJSON)
			return result
		}

		It("to an object", func() {
			data := []byte(`{
				"services": [
					{
						"name": "one",
						"plugins": [
							{ "name": "a" },
						]
					},{
						"name": "two",
						"plugins": [
							{ "name": "b" },
						]
					}
				]
			}`)
			selector := "$..plugins[*]"
			valueFlags := []string{
				"one:\"one\"",
				"name:\"two\"",
			}

			Expect(applyUpdates(data, selector, valueFlags)).To(MatchJSON(`{
				"services": [
					{
						"name": "one",
						"plugins": [
							{
								"name": "two",
								"one": "one"
							}
						]
					},{
						"name": "two",
						"plugins": [
							{
								"name": "two",
								"one": "one"
							}
						]
					}
				]
			}`))
		})

		It("skips non-objects", func() {
			data := []byte(`{
				"plugins": [
					{ "name": "old name" },
					true,
					0,
					["an array"],
					"a string"
				]
			}`)
			selector := "$..plugins[*]"
			valueFlags := []string{
				"name:\"new name\"",
			}

			Expect(applyUpdates(data, selector, valueFlags)).To(MatchJSON(`{
				"plugins": [
					{	"name": "new name" },
					true,
					0,
					["an array"],
					"a string"
				]
			}`))
		})

		It("works on empty objects", func() {
			data := []byte(`{
				"routes": [{}]
			}`)
			selector := "$..routes[*]"
			valueFlags := []string{
				"name:\"new name\"",
			}

			Expect(applyUpdates(data, selector, valueFlags)).To(MatchJSON(`{
				"routes": [
					{	"name": "new name" }
				]
			}`))
		})

		It("deletes an existing key if the value is nil", func() {
			data := []byte(`{
				"routes": [
					{	"name": "new name" }
				]
			}`)
			selector := "$..routes[*]"
			valueFlags := []string{
				"name:", // no value specified, so nil value, and hence delete it
			}

			Expect(applyUpdates(data, selector, valueFlags)).To(MatchJSON(`{
				"routes": [
					{}
				]
			}`))
		})

		It("doesn't insert 'null' when deleting a non-existing key", func() {
			data := []byte(`{
				"upstreams": [
					{	"name": "my name" }
				]
			}`)
			selector := "$..upstreams[*]"
			valueFlags := []string{
				"foobar:", // no value specified, so nil value, and hence delete it
			}

			Expect(applyUpdates(data, selector, valueFlags)).To(MatchJSON(`{
				"upstreams": [
					{ "name": "my name" }
				]
			}`))
		})

		It("can set 'null' if specified", func() {
			data := []byte(`{
				"upstreams": [
					{	"name": "my name" }
				]
			}`)
			selector := "$..upstreams[*]"
			valueFlags := []string{
				"name:null",
			}

			Expect(applyUpdates(data, selector, valueFlags)).To(MatchJSON(`{
				"upstreams": [
					{ "name": null }
				]
			}`))
		})
	})
})
