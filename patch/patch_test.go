package patch_test

import (
	. "github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/patch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Patch", func() {
	Describe("validating --value flags", func() {
		It("validates a number", func() {
			val, err := patch.ValidateValuesFlags([]string{"key1:1", "key2:2"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": 1.0,
				"key2": 2.0,
			}))
		})

		It("validates a string", func() {
			val, err := patch.ValidateValuesFlags([]string{"key1:\"hi\"", "key2:\"there\""})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": "hi",
				"key2": "there",
			}))
		})

		It("validates a boolean", func() {
			val, err := patch.ValidateValuesFlags([]string{"key1:true", "key2:false"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": true,
				"key2": false,
			}))
		})

		It("validates an object", func() {
			val, err := patch.ValidateValuesFlags([]string{"key1:{\"hello\": 123}", "key2:false"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": map[string]interface{}{
					"hello": 123.0,
				},
				"key2": false,
			}))
		})

		It("validates an array", func() {
			val, err := patch.ValidateValuesFlags([]string{"key1:[true]", "key2:false"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": []interface{}{true},
				"key2": false,
			}))
		})

		It("validates an empty value (delete the key)", func() {
			val, err := patch.ValidateValuesFlags([]string{"key1:", "key2:"})

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"key1": patch.DeleteMarker,
				"key2": patch.DeleteMarker,
			}))
		})

		Describe("returns error on", func() {
			It("missing ':'", func() {
				val, err := patch.ValidateValuesFlags([]string{"key1:true", "key2_false"})

				Expect(err).To(MatchError("expected '--value' entry to have format 'key:json-string', got: 'key2_false'"))
				Expect(val).To(BeNil())
			})

			It("invalid JSON", func() {
				val, err := patch.ValidateValuesFlags([]string{"key1:true", "key2:{not valid this stuff...}"})

				Expect(err).To(MatchError("expected '--value' entry to have format 'key:json-string', failed parsing " +
					"json-string in 'key2:{not valid this stuff...}' (did you forget to wrap a json-string-value in quotes?)"))
				Expect(val).To(BeNil())
			})
		})

		Describe("allows for", func() {
			It("multiple ':' characters (splits only by first one)", func() {
				val, err := patch.ValidateValuesFlags([]string{"key1:\":::\""})

				Expect(err).To(BeNil())
				Expect(val).To(BeEquivalentTo(map[string]interface{}{
					"key1": ":::",
				}))
			})
		})
	})

	Describe("validating --selector flags", func() {
		It("returns error on bad JSONpath", func() {
			res, err := patch.ApplyValues(nil, "bad JSONpath", nil)
			Expect(res).To(BeNil())
			Expect(err).To(MatchError("invalid character ' ' at position 3, following \"bad\""))
		})
	})

	Describe("Applying values", func() {
		applyUpdates := func(data []byte, selector string, valueFlags []string) []byte {
			jsonData := MustDeserialize(&data)
			parsedValues, err := patch.ValidateValuesFlags(valueFlags)
			Expect(err).To(BeNil())

			updated, err := patch.ApplyValues(jsonData, selector, parsedValues)
			Expect(err).To(BeNil())

			result := MustSerialize(updated, false)
			return *result
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
