package yamlbasics_test

import (
	. "github.com/kong/go-apiops/yamlbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("yamlbasics", func() {
	Describe("FromObject", func() {
		It("returns an object node", func() {
			data := map[string]interface{}{
				"name": "myName",
			}
			val, err := FromObject(data)
			Expect(err).To(BeNil())
			Expect(val.Kind).To(BeEquivalentTo(yaml.MappingNode))
			Expect(val.Content[0].Value).To(BeEquivalentTo("name"))
			Expect(val.Content[1].Value).To(BeEquivalentTo("myName"))
		})

		It("returns an error if nil", func() {
			val, err := FromObject(nil)

			Expect(err).To(MatchError("not an object, but <nil>"))
			Expect(val).To(BeNil())
		})
	})

	Describe("ToObject", func() {
		It("returns an object", func() {
			data := NewObject()
			SetFieldValue(data, "name", NewString("myName"))
			val, err := ToObject(data)

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{
				"name": "myName",
			}))
		})

		It("returns an error if nil", func() {
			val, err := ToObject(nil)

			Expect(err).To(MatchError("data is not a mapping node/object"))
			Expect(val).To(BeNil())
		})

		It("returns an error if string", func() {
			val, err := ToObject(NewString("123"))

			Expect(err).To(MatchError("data is not a mapping node/object"))
			Expect(val).To(BeNil())
		})
	})

	Describe("ToArray", func() {
		It("returns an array", func() {
			data := NewArray()
			Append(data, NewString("one"))
			val, err := ToArray(data)

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo([]interface{}{
				"one",
			}))
		})

		It("returns an error if nil", func() {
			val, err := ToArray(nil)

			Expect(err).To(MatchError("data is not a sequence node/array"))
			Expect(val).To(BeNil())
		})
	})

	Describe("GetFieldValue", func() {
		It("returns a node if found", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)
			d2 := NewString("yourName")
			SetFieldValue(data, "name2", d2)
			d3 := NewString("hisName")
			SetFieldValue(data, "name3", d3)

			Expect(GetFieldValue(data, "name1")).To(Equal(d1))
			Expect(GetFieldValue(data, "name2")).To(Equal(d2))
			Expect(GetFieldValue(data, "name3")).To(Equal(d3))
		})

		It("returns a nil if not found", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)

			Expect(GetFieldValue(data, "name2")).To(BeNil())
		})

		It("panics if not an object/map", func() {
			data := NewString("myName")

			Expect(func() {
				GetFieldValue(data, "name")
			}).To(Panic())
		})
	})

	Describe("RemoveField", func() {
		It("removes fields", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)
			d2 := NewString("yourName")
			SetFieldValue(data, "name2", d2)
			d3 := NewString("hisName")
			SetFieldValue(data, "name3", d3)

			RemoveField(data, "name1")
			Expect(GetFieldValue(data, "name1")).To(BeNil())
			Expect(GetFieldValue(data, "name2")).To(Equal(d2))
			Expect(GetFieldValue(data, "name3")).To(Equal(d3))

			RemoveField(data, "name2")
			Expect(GetFieldValue(data, "name2")).To(BeNil())
			Expect(GetFieldValue(data, "name3")).To(Equal(d3))

			RemoveField(data, "name3")
			Expect(GetFieldValue(data, "name3")).To(BeNil())

			Expect(data.Content).To(BeEmpty())
		})

		It("ignores non-existing fields", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)
			d2 := NewString("yourName")
			SetFieldValue(data, "name2", d2)
			d3 := NewString("hisName")
			SetFieldValue(data, "name3", d3)

			RemoveField(data, "doesn't exist")
			Expect(GetFieldValue(data, "name1")).To(Equal(d1))
			Expect(GetFieldValue(data, "name2")).To(Equal(d2))
			Expect(GetFieldValue(data, "name3")).To(Equal(d3))

			Expect(len(data.Content)).To(Equal(6))
		})
	})

	Describe("SetFieldValue", func() {
		It("Adds values", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)

			Expect(GetFieldValue(data, "name1")).To(Equal(d1))
		})

		It("removes a field if value == nil", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)
			Expect(GetFieldValue(data, "name1")).To(Equal(d1))

			SetFieldValue(data, "name1", nil)
			Expect(GetFieldValue(data, "name1")).To(BeNil())
		})

		It("allows setting a non-existing field to nil", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)
			Expect(GetFieldValue(data, "name1")).To(Equal(d1))

			SetFieldValue(data, "name2", nil)
			Expect(GetFieldValue(data, "name2")).To(BeNil())

			Expect(len(data.Content)).To(Equal(2))
		})

		It("Overwrites an existing key", func() {
			data := NewObject()
			d1 := NewString("myName")
			SetFieldValue(data, "name1", d1)
			Expect(GetFieldValue(data, "name1")).To(Equal(d1))

			d2 := NewString("yourName")
			SetFieldValue(data, "name1", d2)
			Expect(GetFieldValue(data, "name1")).To(Equal(d2))

			Expect(len(data.Content)).To(Equal(2))
		})
	})

	Describe("Append", func() {
		It("adds array entries", func() {
			data := NewArray()
			d1 := NewString("myName")
			Append(data, d1)
			d2 := NewString("yourName")
			Append(data, d2)
			d3 := NewString("hisName")
			Append(data, d3)

			Expect(data.Content).To(HaveLen(3))
			Expect(data.Content[0]).To(Equal(d1))
			Expect(data.Content[1]).To(Equal(d2))
			Expect(data.Content[2]).To(Equal(d3))
		})

		It("returns an error if the target is not an array", func() {
			data := NewString("myName")
			d1 := NewString("yourName")
			err := Append(data, d1)
			Expect(err).To(MatchError("targetArray is not a sequence node/array"))
		})

		It("returns no error if not appending anything", func() {
			data := NewArray()
			err := Append(data)
			Expect(err).To(BeNil())
		})

		It("returns an error if trying to append a nil", func() {
			data := NewArray()
			err := Append(data, nil)
			Expect(err).To(MatchError("value at index 0 is nil"))
		})

		It("appending a nil does not change the array", func() {
			data := NewArray()
			d1 := NewString("myName")
			Append(data, d1)
			err := Append(data, nil)
			Expect(err).To(MatchError("value at index 0 is nil"))
			Expect(data.Content).To(HaveLen(1))
			Expect(data.Content[0]).To(Equal(d1))
		})
	})

	Describe("AppendSlice", func() {
		It("adds array entries", func() {
			data := NewArray()
			d1 := NewString("myName")
			d2 := NewString("yourName")
			d3 := NewString("hisName")
			err := AppendSlice(data, []*yaml.Node{d1, d2, d3})

			Expect(err).To(BeNil())
			Expect(data.Content).To(HaveLen(3))
			Expect(data.Content[0]).To(Equal(d1))
			Expect(data.Content[1]).To(Equal(d2))
			Expect(data.Content[2]).To(Equal(d3))
		})

		It("returns an error if the target is not an array", func() {
			data := NewString("myName")
			d1 := NewString("myName")
			d2 := NewString("yourName")
			d3 := NewString("hisName")
			err := AppendSlice(data, []*yaml.Node{d1, d2, d3})
			Expect(err).To(MatchError("targetArray is not a sequence node/array"))
		})

		It("returns no error if not appending anything", func() {
			data := NewArray()
			err := AppendSlice(data, []*yaml.Node{})
			Expect(err).To(BeNil())
		})

		It("returns an error if trying to append a nil", func() {
			data := NewArray()
			d1 := NewString("myName")
			// d2 := NewString("yourName")
			d3 := NewString("hisName")
			err := AppendSlice(data, []*yaml.Node{d1, nil, d3})
			Expect(err).To(MatchError("value at index 1 is nil"))
		})

		It("appending a nil does not change the array", func() {
			data := NewArray()
			d1 := NewString("myName")
			Append(data, d1)
			d2 := NewString("yourName")
			d3 := NewString("hisName")
			err := AppendSlice(data, []*yaml.Node{d2, nil, d3})
			Expect(err).To(MatchError("value at index 1 is nil"))
			Expect(data.Content).To(HaveLen(1))
			Expect(data.Content[0]).To(Equal(d1))
		})
	})

	Describe("Search", func() {
		var hits []string

		newSearcher := func() func(*yaml.Node) (bool, error) {
			hits = make([]string, 0)
			return func(node *yaml.Node) (bool, error) {
				hits = append(hits, node.Value)
				return true, nil
			}
		}

		It("panics if targetArray is nil", func() {
			Expect(func() { Search(nil, newSearcher()) }).To(Panic())
		})

		It("panics if targetArray is not an array", func() {
			Expect(func() { Search(NewObject(), newSearcher()) }).To(Panic())
		})

		It("gets hit with each entry", func() {
			data := NewArray()
			Append(data, NewString("myName"), NewString("yourName"), NewString("hisName"))
			next := Search(data, newSearcher())
			for {
				node, _, _ := next()
				if node == nil {
					break
				}
			}

			Expect(hits).To(BeEquivalentTo([]string{"myName", "yourName", "hisName"}))
		})

		It("doesn't fail on an empty array", func() {
			data := NewArray()
			next := Search(data, newSearcher())
			for {
				node, _, _ := next()
				if node == nil {
					break
				}
			}

			Expect(hits).To(BeEquivalentTo([]string{}))
		})

		It("doesn't get hit by nil values", func() {
			data := NewArray()
			Append(data, NewString("myName"), NewString("yourName"), NewString("hisName"))
			data.Content[1] = nil // simulate a nil value
			next := Search(data, newSearcher())
			for {
				node, _, _ := next()
				if node == nil {
					break
				}
			}

			Expect(hits).To(BeEquivalentTo([]string{"myName", "hisName"}))
		})

		It("handles adding/removing values", func() {
			data := NewArray()
			Append(data, NewString("myName"), NewString("yourName"), NewString("hisName"))
			next := Search(data, newSearcher())
			for {
				node, _, _ := next()
				if data.Content[1] != nil {
					data.Content[1] = nil                   // simulate a nil value (after 1st call)
					data.Content[0] = NewString("injected") // replace first value
				}
				if node == nil {
					break
				}
			}

			Expect(hits).To(BeEquivalentTo([]string{"myName", "injected", "hisName"}))
		})
	})

	Describe("CopyNode", func() {
		It("doesn't panic on nil", func() {
			Expect(func() { CopyNode(nil) }).ToNot(Panic())
			Expect(CopyNode(nil)).To(BeNil())
		})

		It("Copies a scalar node", func() {
			node := NewString("myName")
			duplicate := CopyNode(node)
			Expect(duplicate).ToNot(BeNil())
			Expect(duplicate).ToNot(BeIdenticalTo(node))
			Expect(duplicate).To(Equal(node))
		})

		It("Copies a non-scalar node", func() {
			node := NewObject()
			SetFieldValue(node, "key1", NewString("myName"))
			SetFieldValue(node, "key2", NewString("yourName"))
			duplicate := CopyNode(node)
			Expect(duplicate).ToNot(BeNil())
			Expect(len(duplicate.Content)).To(Equal(4))

			node.Content[0].Value = "key3" // change the original
			Expect(duplicate.Content[0].Value).To(Equal("key1"))
			Expect(duplicate.Content[1].Value).To(Equal("myName"))
			Expect(duplicate.Content[2].Value).To(Equal("key2"))
			Expect(duplicate.Content[3].Value).To(Equal("yourName"))
		})
	})
})
