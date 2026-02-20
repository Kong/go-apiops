package lint

import (
	"fmt"
)

// CoreFunction is the signature for all built-in Spectral core functions.
// It receives:
//   - targetVal: the value selected by the JSONPath expression (and optional field).
//     May be nil if the target was not found (equivalent to JS undefined).
//   - opts: the functionOptions from the rule definition. May be nil.
//
// It returns a slice of messages describing violations. An empty/nil slice means the value passes.
type CoreFunction func(targetVal interface{}, opts map[string]interface{}) []string

// coreFunctions is the registry of all built-in Spectral core functions.
var coreFunctions = map[string]CoreFunction{
	"truthy":                     coreTruthy,
	"falsy":                      coreFalsy,
	"defined":                    coreDefined,
	"undefined":                  coreUndefined,
	"pattern":                    corePattern,
	"casing":                     coreCasing,
	"alphabetical":               coreAlphabetical,
	"enumeration":                coreEnumeration,
	"length":                     coreLength,
	"or":                         coreOr,
	"xor":                        coreXor,
	"schema":                     coreSchema,
	"typedEnum":                  coreTypedEnum,
	"unreferencedReusableObject": coreUnreferencedReusableObject,
}

// GetCoreFunction returns the core function with the given name, or an error
// if no such function exists.
func GetCoreFunction(name string) (CoreFunction, error) {
	fn, ok := coreFunctions[name]
	if !ok {
		return nil, fmt.Errorf("unknown core function: %q", name)
	}
	return fn, nil
}

// isTruthy returns true if the value would be considered "truthy" in JavaScript.
// False, empty string, zero, and nil are falsy. Everything else is truthy.
func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	default:
		return true
	}
}

// getStringOption extracts a string option from the functionOptions map.
func getStringOption(opts map[string]interface{}, key string) (string, bool) {
	if opts == nil {
		return "", false
	}
	val, ok := opts[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// getStringSliceOption extracts a string slice option from the functionOptions map.
func getStringSliceOption(opts map[string]interface{}, key string) ([]string, bool) {
	if opts == nil {
		return nil, false
	}
	val, ok := opts[key]
	if !ok {
		return nil, false
	}
	switch v := val.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			result = append(result, s)
		}
		return result, true
	case []string:
		return v, true
	default:
		return nil, false
	}
}

// getFloat64Option extracts a float64 option from the functionOptions map.
func getFloat64Option(opts map[string]interface{}, key string) (float64, bool) {
	if opts == nil {
		return 0, false
	}
	val, ok := opts[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// getBoolOption extracts a bool option from the functionOptions map.
// Returns the value if found and is a bool, or false otherwise.
func getBoolOption(opts map[string]interface{}, key string) bool {
	if opts == nil {
		return false
	}
	val, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b
}

// toFloat64 converts a numeric value to float64.
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}
