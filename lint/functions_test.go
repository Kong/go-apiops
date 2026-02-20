package lint_test

import (
	. "github.com/kong/go-apiops/lint"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Core Functions", func() {
	// Helper: get the core function by name
	getFunc := func(name string) CoreFunction {
		fn, err := GetCoreFunction(name)
		Expect(err).ToNot(HaveOccurred())
		return fn
	}

	Describe("truthy", func() {
		It("passes for a non-empty string", func() {
			fn := getFunc("truthy")
			Expect(fn("hello", nil)).To(BeEmpty())
		})

		It("passes for true", func() {
			fn := getFunc("truthy")
			Expect(fn(true, nil)).To(BeEmpty())
		})

		It("passes for non-zero number", func() {
			fn := getFunc("truthy")
			Expect(fn(42, nil)).To(BeEmpty())
			Expect(fn(float64(3.14), nil)).To(BeEmpty())
		})

		It("passes for a map", func() {
			fn := getFunc("truthy")
			Expect(fn(map[string]interface{}{"key": "val"}, nil)).To(BeEmpty())
		})

		It("passes for a slice", func() {
			fn := getFunc("truthy")
			Expect(fn([]interface{}{"a"}, nil)).To(BeEmpty())
		})

		It("fails for nil", func() {
			fn := getFunc("truthy")
			Expect(fn(nil, nil)).To(HaveLen(1))
		})

		It("fails for false", func() {
			fn := getFunc("truthy")
			Expect(fn(false, nil)).To(HaveLen(1))
		})

		It("fails for empty string", func() {
			fn := getFunc("truthy")
			Expect(fn("", nil)).To(HaveLen(1))
		})

		It("fails for zero", func() {
			fn := getFunc("truthy")
			Expect(fn(0, nil)).To(HaveLen(1))
			Expect(fn(float64(0), nil)).To(HaveLen(1))
		})
	})

	Describe("falsy", func() {
		It("passes for nil", func() {
			fn := getFunc("falsy")
			Expect(fn(nil, nil)).To(BeEmpty())
		})

		It("passes for false", func() {
			fn := getFunc("falsy")
			Expect(fn(false, nil)).To(BeEmpty())
		})

		It("passes for empty string", func() {
			fn := getFunc("falsy")
			Expect(fn("", nil)).To(BeEmpty())
		})

		It("passes for zero", func() {
			fn := getFunc("falsy")
			Expect(fn(0, nil)).To(BeEmpty())
		})

		It("fails for true", func() {
			fn := getFunc("falsy")
			Expect(fn(true, nil)).To(HaveLen(1))
		})

		It("fails for a non-empty string", func() {
			fn := getFunc("falsy")
			Expect(fn("hello", nil)).To(HaveLen(1))
		})

		It("fails for a non-zero number", func() {
			fn := getFunc("falsy")
			Expect(fn(42, nil)).To(HaveLen(1))
		})
	})

	Describe("defined", func() {
		It("passes for any non-nil value", func() {
			fn := getFunc("defined")
			Expect(fn("hello", nil)).To(BeEmpty())
			Expect(fn(false, nil)).To(BeEmpty())
			Expect(fn(0, nil)).To(BeEmpty())
			Expect(fn("", nil)).To(BeEmpty())
		})

		It("fails for nil", func() {
			fn := getFunc("defined")
			Expect(fn(nil, nil)).To(HaveLen(1))
		})
	})

	Describe("undefined", func() {
		It("passes for nil", func() {
			fn := getFunc("undefined")
			Expect(fn(nil, nil)).To(BeEmpty())
		})

		It("fails for any non-nil value", func() {
			fn := getFunc("undefined")
			Expect(fn("hello", nil)).To(HaveLen(1))
			Expect(fn(false, nil)).To(HaveLen(1))
			Expect(fn(0, nil)).To(HaveLen(1))
		})
	})

	Describe("pattern", func() {
		It("passes when value matches the match pattern", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"match": "^hello"}
			Expect(fn("hello world", opts)).To(BeEmpty())
		})

		It("fails when value does not match the match pattern", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"match": "^goodbye"}
			result := fn("hello world", opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("does not match"))
		})

		It("passes when value does not match the notMatch pattern", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"notMatch": "^goodbye"}
			Expect(fn("hello world", opts)).To(BeEmpty())
		})

		It("fails when value matches the notMatch pattern", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"notMatch": "^hello"}
			result := fn("hello world", opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("must not match"))
		})

		It("handles delimited patterns with flags", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"match": "/^HELLO/i"}
			Expect(fn("hello world", opts)).To(BeEmpty())
		})

		It("handles case-insensitive notMatch", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"notMatch": "/^x-/i"}
			result := fn("X-Custom", opts)
			Expect(result).To(HaveLen(1))
		})

		It("handles both match and notMatch together", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"match": "^[a-z]", "notMatch": "bad"}
			Expect(fn("good", opts)).To(BeEmpty())

			result := fn("bad", opts)
			Expect(result).To(HaveLen(1)) // matches notMatch
		})

		It("returns nil for nil input", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"match": ".*"}
			Expect(fn(nil, opts)).To(BeEmpty())
		})

		It("coerces non-string values to string", func() {
			fn := getFunc("pattern")
			opts := map[string]interface{}{"match": "^42$"}
			Expect(fn(42, opts)).To(BeEmpty())
		})
	})

	Describe("casing", func() {
		DescribeTable("validates casing types",
			func(value string, casingType string, shouldPass bool) {
				fn := getFunc("casing")
				opts := map[string]interface{}{"type": casingType}
				result := fn(value, opts)
				if shouldPass {
					Expect(result).To(BeEmpty(), "expected %q to pass %s casing", value, casingType)
				} else {
					Expect(result).To(HaveLen(1), "expected %q to fail %s casing", value, casingType)
				}
			},
			// flat
			Entry("flat: verylongname", "verylongname", "flat", true),
			Entry("flat: VeryLongName fails", "VeryLongName", "flat", false),
			Entry("flat: with digits", "name123", "flat", true),
			// camel
			Entry("camel: veryLongName", "veryLongName", "camel", true),
			Entry("camel: VeryLongName fails", "VeryLongName", "camel", false),
			Entry("camel: single word", "name", "camel", true),
			// pascal
			Entry("pascal: VeryLongName", "VeryLongName", "pascal", true),
			Entry("pascal: veryLongName fails", "veryLongName", "pascal", false),
			// kebab
			Entry("kebab: very-long-name", "very-long-name", "kebab", true),
			Entry("kebab: VeryLongName fails", "VeryLongName", "kebab", false),
			// cobol
			Entry("cobol: VERY-LONG-NAME", "VERY-LONG-NAME", "cobol", true),
			Entry("cobol: very-long-name fails", "very-long-name", "cobol", false),
			// snake
			Entry("snake: very_long_name", "very_long_name", "snake", true),
			Entry("snake: VeryLongName fails", "VeryLongName", "snake", false),
			// macro
			Entry("macro: VERY_LONG_NAME", "VERY_LONG_NAME", "macro", true),
			Entry("macro: very_long_name fails", "very_long_name", "macro", false),
		)

		It("handles disallowDigits option", func() {
			fn := getFunc("casing")
			opts := map[string]interface{}{"type": "flat", "disallowDigits": true}
			Expect(fn("abc", opts)).To(BeEmpty())
			Expect(fn("abc123", opts)).To(HaveLen(1))
		})

		It("handles separator option", func() {
			fn := getFunc("casing")
			opts := map[string]interface{}{
				"type": "pascal",
				"separator": map[string]interface{}{
					"char": "-",
				},
			}
			Expect(fn("X-YourMighty-Header", opts)).To(BeEmpty())
		})

		It("handles separator with allowLeading", func() {
			fn := getFunc("casing")
			opts := map[string]interface{}{
				"type": "kebab",
				"separator": map[string]interface{}{
					"char":         "/",
					"allowLeading": true,
				},
			}
			Expect(fn("/some-path", opts)).To(BeEmpty())
		})

		It("returns nil for empty string", func() {
			fn := getFunc("casing")
			opts := map[string]interface{}{"type": "camel"}
			Expect(fn("", opts)).To(BeEmpty())
		})

		It("returns nil for nil input", func() {
			fn := getFunc("casing")
			opts := map[string]interface{}{"type": "camel"}
			Expect(fn(nil, opts)).To(BeEmpty())
		})
	})

	Describe("alphabetical", func() {
		It("passes for a sorted simple array", func() {
			fn := getFunc("alphabetical")
			Expect(fn([]interface{}{"a", "b", "c"}, nil)).To(BeEmpty())
		})

		It("fails for an unsorted simple array", func() {
			fn := getFunc("alphabetical")
			result := fn([]interface{}{"b", "a", "c"}, nil)
			Expect(result).To(HaveLen(1))
		})

		It("passes for a sorted array of objects with keyedBy", func() {
			fn := getFunc("alphabetical")
			arr := []interface{}{
				map[string]interface{}{"name": "alpha"},
				map[string]interface{}{"name": "beta"},
				map[string]interface{}{"name": "gamma"},
			}
			opts := map[string]interface{}{"keyedBy": "name"}
			Expect(fn(arr, opts)).To(BeEmpty())
		})

		It("fails for an unsorted array of objects with keyedBy", func() {
			fn := getFunc("alphabetical")
			arr := []interface{}{
				map[string]interface{}{"name": "gamma"},
				map[string]interface{}{"name": "alpha"},
				map[string]interface{}{"name": "beta"},
			}
			opts := map[string]interface{}{"keyedBy": "name"}
			result := fn(arr, opts)
			Expect(result).To(HaveLen(1))
		})

		It("passes for a single-element array", func() {
			fn := getFunc("alphabetical")
			Expect(fn([]interface{}{"only"}, nil)).To(BeEmpty())
		})

		It("passes for an empty array", func() {
			fn := getFunc("alphabetical")
			Expect(fn([]interface{}{}, nil)).To(BeEmpty())
		})

		It("returns nil for nil input", func() {
			fn := getFunc("alphabetical")
			Expect(fn(nil, nil)).To(BeEmpty())
		})
	})

	Describe("enumeration", func() {
		It("passes when value is in the allowed set", func() {
			fn := getFunc("enumeration")
			opts := map[string]interface{}{
				"values": []interface{}{"users", "articles", "categories"},
			}
			Expect(fn("users", opts)).To(BeEmpty())
		})

		It("fails when value is not in the allowed set", func() {
			fn := getFunc("enumeration")
			opts := map[string]interface{}{
				"values": []interface{}{"users", "articles", "categories"},
			}
			result := fn("admin", opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("does not match"))
		})

		It("handles numeric values", func() {
			fn := getFunc("enumeration")
			opts := map[string]interface{}{
				"values": []interface{}{1, 2, 3},
			}
			Expect(fn(2, opts)).To(BeEmpty())
		})

		It("returns nil for nil input", func() {
			fn := getFunc("enumeration")
			opts := map[string]interface{}{
				"values": []interface{}{"a"},
			}
			Expect(fn(nil, opts)).To(BeEmpty())
		})
	})

	Describe("length", func() {
		It("passes when string length is within bounds", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"min": float64(1), "max": float64(10)}
			Expect(fn("hello", opts)).To(BeEmpty())
		})

		It("fails when string is too short", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"min": float64(10)}
			result := fn("hi", opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("shorter"))
		})

		It("fails when string is too long", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"max": float64(3)}
			result := fn("hello", opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("longer"))
		})

		It("checks array length", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"min": float64(2)}
			Expect(fn([]interface{}{"a", "b", "c"}, opts)).To(BeEmpty())
			result := fn([]interface{}{"a"}, opts)
			Expect(result).To(HaveLen(1))
		})

		It("checks object property count", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"min": float64(2)}
			obj := map[string]interface{}{"a": 1, "b": 2}
			Expect(fn(obj, opts)).To(BeEmpty())
			result := fn(map[string]interface{}{"a": 1}, opts)
			Expect(result).To(HaveLen(1))
		})

		It("checks numeric value directly", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"min": float64(10), "max": float64(100)}
			Expect(fn(float64(50), opts)).To(BeEmpty())
			result := fn(float64(5), opts)
			Expect(result).To(HaveLen(1))
		})

		It("returns nil for nil input", func() {
			fn := getFunc("length")
			opts := map[string]interface{}{"min": float64(1)}
			Expect(fn(nil, opts)).To(BeEmpty())
		})
	})

	Describe("or", func() {
		It("passes when at least one property is defined", func() {
			fn := getFunc("or")
			obj := map[string]interface{}{"title": "My API"}
			opts := map[string]interface{}{
				"properties": []interface{}{"title", "description"},
			}
			Expect(fn(obj, opts)).To(BeEmpty())
		})

		It("passes when multiple properties are defined", func() {
			fn := getFunc("or")
			obj := map[string]interface{}{"title": "My API", "description": "desc"}
			opts := map[string]interface{}{
				"properties": []interface{}{"title", "description"},
			}
			Expect(fn(obj, opts)).To(BeEmpty())
		})

		It("fails when no properties are defined", func() {
			fn := getFunc("or")
			obj := map[string]interface{}{"other": "value"}
			opts := map[string]interface{}{
				"properties": []interface{}{"title", "description"},
			}
			result := fn(obj, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("at least one"))
		})

		It("handles more than 4 properties in error message", func() {
			fn := getFunc("or")
			obj := map[string]interface{}{}
			opts := map[string]interface{}{
				"properties": []interface{}{"a", "b", "c", "d", "e"},
			}
			result := fn(obj, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("other properties"))
		})
	})

	Describe("xor", func() {
		It("passes when exactly one property is defined", func() {
			fn := getFunc("xor")
			obj := map[string]interface{}{"value": "foo"}
			opts := map[string]interface{}{
				"properties": []interface{}{"value", "externalValue"},
			}
			Expect(fn(obj, opts)).To(BeEmpty())
		})

		It("fails when no properties are defined", func() {
			fn := getFunc("xor")
			obj := map[string]interface{}{"other": "foo"}
			opts := map[string]interface{}{
				"properties": []interface{}{"value", "externalValue"},
			}
			result := fn(obj, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("at least one"))
		})

		It("fails when multiple properties are defined", func() {
			fn := getFunc("xor")
			obj := map[string]interface{}{"value": "foo", "externalValue": "bar"}
			opts := map[string]interface{}{
				"properties": []interface{}{"value", "externalValue"},
			}
			result := fn(obj, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("just one"))
		})
	})

	Describe("typedEnum", func() {
		It("passes when all enum values match the type", func() {
			fn := getFunc("typedEnum")
			obj := map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"a", "b", "c"},
			}
			Expect(fn(obj, nil)).To(BeEmpty())
		})

		It("fails when enum values don't match the type", func() {
			fn := getFunc("typedEnum")
			obj := map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"a", 42, "c"},
			}
			result := fn(obj, nil)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("does not match type"))
		})

		It("handles integer type", func() {
			fn := getFunc("typedEnum")
			obj := map[string]interface{}{
				"type": "integer",
				"enum": []interface{}{float64(1), float64(2), float64(3)},
			}
			Expect(fn(obj, nil)).To(BeEmpty())
		})

		It("detects non-integer in integer enum", func() {
			fn := getFunc("typedEnum")
			obj := map[string]interface{}{
				"type": "integer",
				"enum": []interface{}{float64(1), 3.14, float64(3)},
			}
			result := fn(obj, nil)
			Expect(result).To(HaveLen(1))
		})

		It("handles boolean type", func() {
			fn := getFunc("typedEnum")
			obj := map[string]interface{}{
				"type": "boolean",
				"enum": []interface{}{true, false},
			}
			Expect(fn(obj, nil)).To(BeEmpty())
		})

		It("returns nil when type or enum is missing", func() {
			fn := getFunc("typedEnum")
			Expect(fn(map[string]interface{}{"type": "string"}, nil)).To(BeEmpty())
			Expect(fn(map[string]interface{}{"enum": []interface{}{"a"}}, nil)).To(BeEmpty())
		})

		It("returns nil for nil input", func() {
			fn := getFunc("typedEnum")
			Expect(fn(nil, nil)).To(BeEmpty())
		})
	})

	Describe("schema", func() {
		It("passes when value matches schema type", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "string",
				},
			}
			Expect(fn("hello", opts)).To(BeEmpty())
		})

		It("fails when value doesn't match schema type", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "string",
				},
			}
			result := fn(42, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("type"))
		})

		It("validates array with minItems", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type":     "array",
					"minItems": float64(2),
				},
			}
			Expect(fn([]interface{}{1, 2, 3}, opts)).To(BeEmpty())
			result := fn([]interface{}{1}, opts)
			Expect(result).To(HaveLen(1))
		})

		It("validates object with required properties", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type":     "object",
					"required": []interface{}{"name"},
				},
			}
			Expect(fn(map[string]interface{}{"name": "test"}, opts)).To(BeEmpty())
			result := fn(map[string]interface{}{"other": "test"}, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("required"))
		})

		It("validates string minLength", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type":      "string",
					"minLength": float64(5),
				},
			}
			Expect(fn("hello", opts)).To(BeEmpty())
			result := fn("hi", opts)
			Expect(result).To(HaveLen(1))
		})

		It("validates numeric minimum", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type":    "number",
					"minimum": float64(10),
				},
			}
			Expect(fn(float64(15), opts)).To(BeEmpty())
			result := fn(float64(5), opts)
			Expect(result).To(HaveLen(1))
		})

		It("returns nil for nil input", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{"type": "string"},
			}
			Expect(fn(nil, opts)).To(BeEmpty())
		})

		It("validates items schema in array", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
					},
				},
			}
			arr := []interface{}{
				map[string]interface{}{"name": "a"},
				map[string]interface{}{"name": "b"},
			}
			Expect(fn(arr, opts)).To(BeEmpty())
		})

		It("validates enum in schema", func() {
			fn := getFunc("schema")
			opts := map[string]interface{}{
				"schema": map[string]interface{}{
					"enum": []interface{}{"a", "b", "c"},
				},
			}
			Expect(fn("a", opts)).To(BeEmpty())
			result := fn("d", opts)
			Expect(result).To(HaveLen(1))
		})
	})

	Describe("unreferencedReusableObject", func() {
		It("identifies unreferenced objects", func() {
			fn := getFunc("unreferencedReusableObject")
			target := map[string]interface{}{
				"User":    map[string]interface{}{"type": "object"},
				"Orphan":  map[string]interface{}{"type": "object"},
				"Address": map[string]interface{}{"type": "object"},
			}
			opts := map[string]interface{}{
				"reusableObjectsLocation": "#/definitions",
				"__document__": map[string]interface{}{
					"definitions": target,
					"paths": map[string]interface{}{
						"/users": map[string]interface{}{
							"get": map[string]interface{}{
								"responses": map[string]interface{}{
									"200": map[string]interface{}{
										"schema": map[string]interface{}{
											"$ref": "#/definitions/User",
										},
									},
								},
							},
						},
						"/address": map[string]interface{}{
							"get": map[string]interface{}{
								"responses": map[string]interface{}{
									"200": map[string]interface{}{
										"schema": map[string]interface{}{
											"$ref": "#/definitions/Address",
										},
									},
								},
							},
						},
					},
				},
			}
			result := fn(target, opts)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(ContainSubstring("Orphan"))
		})

		It("returns nil when all objects are referenced", func() {
			fn := getFunc("unreferencedReusableObject")
			target := map[string]interface{}{
				"User": map[string]interface{}{"type": "object"},
			}
			opts := map[string]interface{}{
				"reusableObjectsLocation": "#/definitions",
				"__document__": map[string]interface{}{
					"definitions": target,
					"paths": map[string]interface{}{
						"/users": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/definitions/User",
							},
						},
					},
				},
			}
			Expect(fn(target, opts)).To(BeEmpty())
		})

		It("returns nil for nil input", func() {
			fn := getFunc("unreferencedReusableObject")
			Expect(fn(nil, nil)).To(BeEmpty())
		})
	})

	Describe("GetCoreFunction", func() {
		It("returns an error for unknown functions", func() {
			_, err := GetCoreFunction("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown core function"))
		})
	})
})
