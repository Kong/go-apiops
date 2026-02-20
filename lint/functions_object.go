package lint

import (
	"encoding/json"
	"fmt"
	"strings"
)

// coreOr checks that at least one of the specified properties is defined on an object.
// Options:
//   - properties: array of property names to check (required, minimum 2)
func coreOr(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return []string{"value must be an object for 'or' check"}
	}

	obj, ok := targetVal.(map[string]interface{})
	if !ok {
		return []string{fmt.Sprintf("value must be an object for 'or' check, got %T", targetVal)}
	}

	properties, ok := getStringSliceOption(opts, "properties")
	if !ok || len(properties) < 2 {
		return []string{"'or' function requires 'properties' option with at least 2 entries"}
	}

	for _, prop := range properties {
		if _, exists := obj[prop]; exists {
			return nil // at least one is defined
		}
	}

	if len(properties) > 4 {
		shortProps := properties[:3]
		count := fmt.Sprintf("%d other properties must be defined", len(properties)-3)
		return []string{
			fmt.Sprintf(`at least one of "%s" or %s`, strings.Join(shortProps, `" or "`), count),
		}
	}
	return []string{
		fmt.Sprintf(`at least one of "%s" must be defined`, strings.Join(properties, `" or "`)),
	}
}

// coreXor checks that exactly one of the specified properties is defined on an object.
// Options:
//   - properties: array of property names to check (required, minimum 2)
func coreXor(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return []string{"value must be an object for 'xor' check"}
	}

	obj, ok := targetVal.(map[string]interface{})
	if !ok {
		return []string{fmt.Sprintf("value must be an object for 'xor' check, got %T", targetVal)}
	}

	properties, ok := getStringSliceOption(opts, "properties")
	if !ok || len(properties) < 2 {
		return []string{"'xor' function requires 'properties' option with at least 2 entries"}
	}

	var results []string

	// Find which properties are defined
	var defined []string
	for _, prop := range properties {
		if _, exists := obj[prop]; exists {
			defined = append(defined, prop)
		}
	}

	if len(defined) == 0 {
		if len(properties) > 4 {
			shortProps := properties[:3]
			count := fmt.Sprintf("%d other properties must be defined", len(properties)-3)
			results = append(results,
				fmt.Sprintf(`at least one of "%s" or %s`, strings.Join(shortProps, `" or "`), count))
		} else {
			results = append(results,
				fmt.Sprintf(`at least one of "%s" must be defined`, strings.Join(properties, `" or "`)))
		}
	}

	if len(defined) > 1 {
		results = append(results,
			fmt.Sprintf(`just one of "%s" must be defined`, strings.Join(defined, `" and "`)))
	}

	return results
}

// coreTypedEnum checks that when a schema has both 'type' and 'enum', all enum values
// match the declared type.
func coreTypedEnum(targetVal interface{}, _ map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	obj, ok := targetVal.(map[string]interface{})
	if !ok {
		return nil
	}

	typeRaw, hasType := obj["type"]
	enumRaw, hasEnum := obj["enum"]
	if !hasType || !hasEnum {
		return nil
	}

	typeName, ok := typeRaw.(string)
	if !ok {
		return nil
	}

	enumArr, ok := enumRaw.([]interface{})
	if !ok {
		return nil
	}

	var results []string
	for i, enumVal := range enumArr {
		if !matchesJSONSchemaType(enumVal, typeName) {
			results = append(results,
				fmt.Sprintf("enum value at index %d (%v) does not match type %q", i, enumVal, typeName))
		}
	}

	return results
}

// matchesJSONSchemaType checks if a value matches a JSON Schema type name.
func matchesJSONSchemaType(val interface{}, typeName string) bool {
	if val == nil {
		return typeName == "null"
	}

	switch typeName {
	case "string":
		_, ok := val.(string)
		return ok
	case "number":
		_, ok := toFloat64(val)
		return ok
	case "integer":
		f, ok := toFloat64(val)
		if !ok {
			return false
		}
		return f == float64(int64(f))
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "array":
		_, ok := val.([]interface{})
		return ok
	case "object":
		_, ok := val.(map[string]interface{})
		return ok
	case "null":
		return val == nil
	default:
		return false
	}
}

// coreUnreferencedReusableObject identifies objects that are defined in a reusable location
// but never referenced via $ref in the document.
// Options:
//   - reusableObjectsLocation: a JSON pointer to the location of reusable objects
//     (e.g., "#/definitions", "#/components/schemas") (required)
//
// The targetVal should be the object at the reusableObjectsLocation (the value that `given` points to).
// The function needs access to the full document to scan for $ref references, which is passed
// via a special key "__document__" in the opts map (injected by the engine).
func coreUnreferencedReusableObject(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	obj, ok := targetVal.(map[string]interface{})
	if !ok {
		return nil
	}

	location, ok := getStringOption(opts, "reusableObjectsLocation")
	if !ok {
		return []string{"unreferencedReusableObject requires 'reusableObjectsLocation' option"}
	}

	// Get the full document from the special options key injected by the engine
	doc := opts["__document__"]

	// Collect all $ref values from the entire document
	refs := make(map[string]bool)
	collectRefs(doc, refs)

	// Check each key in the reusable objects location
	var results []string
	for name := range obj {
		refPath := location + "/" + name
		if !refs[refPath] {
			results = append(results,
				fmt.Sprintf("potential orphaned reusable object: %s", refPath))
		}
	}

	return results
}

// collectRefs walks the document tree and collects all $ref string values.
func collectRefs(val interface{}, refs map[string]bool) {
	switch v := val.(type) {
	case map[string]interface{}:
		if ref, ok := v["$ref"]; ok {
			if refStr, ok := ref.(string); ok {
				refs[refStr] = true
			}
		}
		for _, child := range v {
			collectRefs(child, refs)
		}
	case []interface{}:
		for _, item := range v {
			collectRefs(item, refs)
		}
	}
}

// coreSchema validates a value against a JSON Schema.
// Options:
//   - schema: a JSON Schema object (required)
//   - dialect: the JSON Schema draft to use (optional, default "auto")
//   - allErrors: if true, return all errors; otherwise only the first (optional, default false)
//
// This is a placeholder for JSON Schema validation - implemented in functions_schema.go.
// It uses encoding/json-based validation via the santhosh-tekuri/jsonschema library.
func coreSchema(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	schemaRaw, ok := opts["schema"]
	if !ok {
		return []string{"schema function requires 'schema' option"}
	}

	allErrors := getBoolOption(opts, "allErrors")

	return validateJSONSchema(targetVal, schemaRaw, allErrors)
}

// validateJSONSchema validates a value against a JSON Schema using basic type checking.
// This is a self-contained implementation that handles the most common JSON Schema keywords
// without requiring an external dependency.
func validateJSONSchema(targetVal interface{}, schema interface{}, allErrors bool) []string {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return []string{"schema must be a JSON Schema object"}
	}

	var results []string

	// Type check
	if schemaType, ok := schemaMap["type"]; ok {
		typeStr, isStr := schemaType.(string)
		if isStr {
			if !matchesJSONSchemaType(targetVal, typeStr) {
				msg := fmt.Sprintf("value must be of type %q", typeStr)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
	}

	// Enum check
	if enumRaw, ok := schemaMap["enum"]; ok {
		if enumArr, ok := enumRaw.([]interface{}); ok {
			found := false
			targetJSON := toJSON(targetVal)
			for _, allowed := range enumArr {
				if toJSON(allowed) == targetJSON {
					found = true
					break
				}
			}
			if !found {
				msg := "value must be one of the enum values"
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
	}

	// String validations
	if strVal, ok := targetVal.(string); ok {
		if minLen, ok := getFloat64FromMap(schemaMap, "minLength"); ok {
			if float64(len(strVal)) < minLen {
				msg := fmt.Sprintf("string length must be >= %g", minLen)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		if maxLen, ok := getFloat64FromMap(schemaMap, "maxLength"); ok {
			if float64(len(strVal)) > maxLen {
				msg := fmt.Sprintf("string length must be <= %g", maxLen)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		if pattern, ok := schemaMap["pattern"]; ok {
			if patStr, ok := pattern.(string); ok {
				re, err := compilePattern(patStr)
				if err == nil && !re.MatchString(strVal) {
					msg := fmt.Sprintf("string does not match pattern %q", patStr)
					if !allErrors {
						return []string{msg}
					}
					results = append(results, msg)
				}
			}
		}
	}

	// Numeric validations
	if numVal, isNum := toFloat64(targetVal); isNum {
		if minimum, ok := getFloat64FromMap(schemaMap, "minimum"); ok {
			if numVal < minimum {
				msg := fmt.Sprintf("value must be >= %g", minimum)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		if maximum, ok := getFloat64FromMap(schemaMap, "maximum"); ok {
			if numVal > maximum {
				msg := fmt.Sprintf("value must be <= %g", maximum)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
	}

	// Array validations
	if arrVal, ok := targetVal.([]interface{}); ok {
		if minItems, ok := getFloat64FromMap(schemaMap, "minItems"); ok {
			if float64(len(arrVal)) < minItems {
				msg := fmt.Sprintf("array must have >= %g items", minItems)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		if maxItems, ok := getFloat64FromMap(schemaMap, "maxItems"); ok {
			if float64(len(arrVal)) > maxItems {
				msg := fmt.Sprintf("array must have <= %g items", maxItems)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		// Validate items schema
		if itemsSchema, ok := schemaMap["items"]; ok {
			for i, item := range arrVal {
				itemResults := validateJSONSchema(item, itemsSchema, allErrors)
				for _, msg := range itemResults {
					results = append(results, fmt.Sprintf("items[%d]: %s", i, msg))
					if !allErrors {
						return results
					}
				}
			}
		}
	}

	// Object validations
	if objVal, ok := targetVal.(map[string]interface{}); ok {
		if minProps, ok := getFloat64FromMap(schemaMap, "minProperties"); ok {
			if float64(len(objVal)) < minProps {
				msg := fmt.Sprintf("object must have >= %g properties", minProps)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		if maxProps, ok := getFloat64FromMap(schemaMap, "maxProperties"); ok {
			if float64(len(objVal)) > maxProps {
				msg := fmt.Sprintf("object must have <= %g properties", maxProps)
				if !allErrors {
					return []string{msg}
				}
				results = append(results, msg)
			}
		}
		// Required properties
		if requiredRaw, ok := schemaMap["required"]; ok {
			if requiredArr, ok := requiredRaw.([]interface{}); ok {
				for _, req := range requiredArr {
					if reqStr, ok := req.(string); ok {
						if _, exists := objVal[reqStr]; !exists {
							msg := fmt.Sprintf("object is missing required property %q", reqStr)
							if !allErrors {
								return []string{msg}
							}
							results = append(results, msg)
						}
					}
				}
			}
		}
		// Properties schemas
		if propsRaw, ok := schemaMap["properties"]; ok {
			if propsMap, ok := propsRaw.(map[string]interface{}); ok {
				for propName, propSchema := range propsMap {
					if propVal, exists := objVal[propName]; exists {
						propResults := validateJSONSchema(propVal, propSchema, allErrors)
						for _, msg := range propResults {
							results = append(results, fmt.Sprintf("property %q: %s", propName, msg))
							if !allErrors {
								return results
							}
						}
					}
				}
			}
		}
	}

	return results
}

func getFloat64FromMap(m map[string]interface{}, key string) (float64, bool) {
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	return toFloat64Cast(val)
}

func toFloat64Cast(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func toJSON(val interface{}) string {
	b, _ := json.Marshal(val)
	return string(b)
}
