package jsonbasics_test

import (
	. "github.com/kong/go-apiops/filebasics"
	. "github.com/kong/go-apiops/jsonbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("jsonbasics", func() {
	Describe("ToObject", func() {
		It("returns an object", func() {
			data := map[string]interface{}{}
			val, err := ToObject(data)

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(map[string]interface{}{}))
		})

		It("returns an error if nil", func() {
			val, err := ToObject(nil)

			Expect(err).To(MatchError("not an object, but %!t(<nil>)"))
			Expect(val).To(BeNil())
		})

		It("returns an error if string", func() {
			val, err := ToObject("123")

			Expect(err).To(MatchError("not an object, but %!t(string=123)"))
			Expect(val).To(BeNil())
		})
	})

	Describe("ToArray", func() {
		It("returns an array", func() {
			data := []interface{}{}
			val, err := ToArray(data)

			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo([]interface{}{}))
		})

		It("returns an error if nil", func() {
			val, err := ToArray(nil)

			Expect(err).To(MatchError("not an array, but %!t(<nil>)"))
			Expect(val).To(BeNil())
		})

		It("returns an error if string", func() {
			val, err := ToArray("123")

			Expect(err).To(MatchError("not an array, but %!t(string=123)"))
			Expect(val).To(BeNil())
		})
	})

	Describe("GetObjectArrayField", func() {
		It("returns an array with the objects found", func() {
			data := []byte(`{
				"myArray": [
					{ "name": "one" },
					{ "name": "two" }
				]
			}`)
			objArr, err := GetObjectArrayField(MustDeserialize(data), "myArray")

			Expect(err).To(BeNil())
			Expect(objArr).To(BeEquivalentTo([]map[string]interface{}{
				{
					"name": "one",
				},
				{
					"name": "two",
				},
			}))
		})

		It("skips non-objects found", func() {
			data := []byte(`{
				"myArray": [
					123,
					{ "name": "one" },
					true,
					{ "name": "two" },
					[1,2,3]
				]
			}`)
			objArr, err := GetObjectArrayField(MustDeserialize(data), "myArray")

			Expect(err).To(BeNil())
			Expect(objArr).To(BeEquivalentTo([]map[string]interface{}{
				{
					"name": "one",
				},
				{
					"name": "two",
				},
			}))
		})

		It("returns an empty array if the field doesn't exist", func() {
			data := []byte(`{}`)
			objArr, err := GetObjectArrayField(MustDeserialize(data), "myArray")

			Expect(err).To(BeNil())
			Expect(objArr).To(BeEquivalentTo([]map[string]interface{}{}))
		})

		It("returns an empty array if no objects are found", func() {
			data := []byte(`{
				"myArray": [
					123,
					true,
					[1,2,3]
				]
			}`)
			objArr, err := GetObjectArrayField(MustDeserialize(data), "myArray")

			Expect(err).To(BeNil())
			Expect(objArr).To(BeEquivalentTo([]map[string]interface{}{}))
		})

		It("returns an error if the field is not an array", func() {
			data := []byte(`{
				"myArray": "it's a string"
			}`)
			objArr, err := GetObjectArrayField(MustDeserialize(data), "myArray")

			Expect(err).To(MatchError("not an array, but %!t(string=it's a string)"))
			Expect(objArr).To(BeNil())
		})
	})

	Describe("SetObjectArrayField", func() {
		It("sets an array to be recognized as an array again", func() {
			data := []byte(`{
				"myArray": [
					{ "name": "one" }
				]
			}`)
			obj := MustDeserialize(data)
			objArr, err := GetObjectArrayField(obj, "myArray")
			Expect(err).To(BeNil())
			Expect(objArr[0]["name"]).To(Equal("one"))

			// append a new entry and set it in the object
			entry := make(map[string]interface{})
			entry["name"] = "two"
			obj["myArray"] = append(objArr, entry)

			// since the myArray field is now []map[string]interface{} instead of
			// []interface{} it will no longer be recognized as an array
			objArr2, err := GetObjectArrayField(obj, "myArray")
			Expect(err.Error()).To(ContainSubstring("not an array"))
			Expect(objArr2).To(BeNil())

			// Do it again, but use the SetObjectArrayField
			SetObjectArrayField(obj, "myArray", append(objArr, entry))
			// check that it worked
			objArr3, err := GetObjectArrayField(obj, "myArray")
			Expect(err).To(BeNil())
			Expect(objArr3[0]["name"]).To(Equal("one"))
			Expect(objArr3[1]["name"]).To(Equal("two"))
		})
	})

	Describe("RemoveObjectFromArrayByFieldValue", func() {
		PIt("still to do", func() {
		})
	})

	Describe("GetStringField", func() {
		PIt("still to do", func() {
		})
	})

	Describe("GetStringIndex", func() {
		PIt("still to do", func() {
		})
	})

	Describe("GetBoolField", func() {
		PIt("still to do", func() {
		})
	})

	Describe("GetBoolIndex", func() {
		PIt("still to do", func() {
		})
	})

	Describe("DeepCopyObject", func() {
		PIt("still to do", func() {
		})
	})

	Describe("DeepCopyArray", func() {
		PIt("still to do", func() {
		})
	})

	Describe("GetFloat64Index", func() {
		It("returns the float or error", func() {
			data := []interface{}{
				1.5,
				-1.5,
				"text",
			}
			val, err := GetFloat64Index(data, 0)
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(1.5))
			val, err = GetFloat64Index(data, 1)
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(-1.5))
			val, err = GetFloat64Index(data, 2)
			Expect(err).To(MatchError("expected index '2' to be a float"))
			Expect(val).To(BeEquivalentTo(0))
		})
	})

	Describe("GetFloat64Field", func() {
		It("returns the float or error", func() {
			data := map[string]interface{}{
				"one":   1.5,
				"two":   -1.5,
				"three": "text",
			}
			val, err := GetFloat64Field(data, "one")
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(1.5))
			val, err = GetFloat64Field(data, "two")
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(-1.5))
			val, err = GetFloat64Field(data, "three")
			Expect(err).To(MatchError("expected key 'three' to be a float"))
			Expect(val).To(BeEquivalentTo(0))
		})
	})

	Describe("GetInt64Index", func() {
		It("returns the integer or error", func() {
			data := []interface{}{
				1,
				-1,
				1.5,
			}
			val, err := GetInt64Index(data, 0)
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(int64(1)))
			val, err = GetInt64Index(data, 1)
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(int64(-1)))
			val, err = GetInt64Index(data, 2)
			Expect(err).To(MatchError("expected index '2' to be an integer"))
			Expect(val).To(BeEquivalentTo(0))
		})
	})

	Describe("GetInt64Field", func() {
		It("returns the integer or error", func() {
			data := map[string]interface{}{
				"one":   1,
				"two":   -1,
				"three": 1.5,
			}
			val, err := GetInt64Field(data, "one")
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(int64(1)))
			val, err = GetInt64Field(data, "two")
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(int64(-1)))
			val, err = GetInt64Field(data, "three")
			Expect(err).To(MatchError("expected key 'three' to be an integer"))
			Expect(val).To(BeEquivalentTo(0))
		})
	})

	Describe("GetUInt64Index", func() {
		It("Returns the unsigned integer or error", func() {
			data := []interface{}{
				uint64(1),
				uint64(2),
				1.5,
			}
			val, err := GetUInt64Index(data, 0)
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(uint64(1)))
			val, err = GetUInt64Index(data, 1)
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(uint64(2)))
			val, err = GetUInt64Index(data, 2)
			Expect(err).To(MatchError("expected index '2' to be an unsigned integer"))
			Expect(val).To(BeEquivalentTo(0))
		})
	})

	Describe("GetUInt64Field", func() {
		It("returns the unsigned integer or error", func() {
			data := map[string]interface{}{
				"one":   uint64(1),
				"two":   uint64(2),
				"three": 1.5,
			}
			val, err := GetUInt64Field(data, "one")
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(uint64(1)))
			val, err = GetUInt64Field(data, "two")
			Expect(err).To(BeNil())
			Expect(val).To(BeEquivalentTo(uint64(2)))
			val, err = GetUInt64Field(data, "three")
			Expect(err).To(MatchError("expected key 'three' to be an unsigned integer"))
			Expect(val).To(BeEquivalentTo(0))
		})
	})
})
