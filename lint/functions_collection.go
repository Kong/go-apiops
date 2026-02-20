package lint

import (
	"fmt"
	"sort"
	"strings"
)

// coreAlphabetical checks that an array or object keys are in alphabetical order.
// Options:
//   - keyedBy: when sorting an array of objects, which key to use for comparison (optional)
func coreAlphabetical(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	keyedBy, _ := getStringOption(opts, "keyedBy")

	switch v := targetVal.(type) {
	case []interface{}:
		return checkArrayAlphabetical(v, keyedBy)
	case map[string]interface{}:
		return checkObjectKeysAlphabetical(v)
	default:
		return []string{fmt.Sprintf("value must be an array or object for alphabetical check, got %T", targetVal)}
	}
}

func checkArrayAlphabetical(arr []interface{}, keyedBy string) []string {
	if len(arr) < 2 {
		return nil
	}

	if keyedBy != "" {
		// Extract the keyed values from array of objects
		values := make([]string, 0, len(arr))
		for _, item := range arr {
			obj, ok := item.(map[string]interface{})
			if !ok {
				return []string{"property must be an object"}
			}
			keyVal, ok := obj[keyedBy]
			if ok {
				values = append(values, fmt.Sprintf("%v", keyVal))
			}
		}
		return checkStringsAlphabetical(values)
	}

	// Simple array of values
	values := make([]string, 0, len(arr))
	for _, item := range arr {
		values = append(values, fmt.Sprintf("%v", item))
	}
	return checkStringsAlphabetical(values)
}

func checkStringsAlphabetical(values []string) []string {
	for i := 0; i < len(values)-1; i++ {
		if strings.Compare(values[i], values[i+1]) > 0 {
			return []string{
				"properties must follow the alphabetical order",
			}
		}
	}
	return nil
}

func checkObjectKeysAlphabetical(obj map[string]interface{}) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	if len(keys) < 2 {
		return nil
	}

	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	for i := range keys {
		if keys[i] != sorted[i] {
			return []string{"properties must follow the alphabetical order"}
		}
	}
	return nil
}

// coreEnumeration checks that a value exists in a set of allowed values.
// Options:
//   - values: array of allowed values (required)
func coreEnumeration(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	valuesRaw, ok := opts["values"]
	if !ok {
		return []string{"enumeration function requires 'values' option"}
	}

	values, ok := valuesRaw.([]interface{})
	if !ok {
		return []string{"enumeration 'values' must be an array"}
	}

	targetStr := fmt.Sprintf("%v", targetVal)
	for _, allowed := range values {
		if fmt.Sprintf("%v", allowed) == targetStr {
			return nil
		}
	}

	allowedStrs := make([]string, len(values))
	for i, v := range values {
		allowedStrs[i] = fmt.Sprintf("%v", v)
	}
	return []string{
		fmt.Sprintf("`%s` does not match any of the allowed values: %s",
			targetStr, strings.Join(allowedStrs, ", ")),
	}
}

// coreLength checks the length of a string, array, or object key count, or a numeric value,
// against min/max constraints.
// Options:
//   - min: minimum length/count/value (optional, but at least one of min/max required)
//   - max: maximum length/count/value (optional, but at least one of min/max required)
func coreLength(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	var value float64

	switch v := targetVal.(type) {
	case string:
		value = float64(len(v))
	case []interface{}:
		value = float64(len(v))
	case map[string]interface{}:
		value = float64(len(v))
	default:
		if numVal, ok := toFloat64(targetVal); ok {
			value = numVal
		} else {
			return []string{fmt.Sprintf("cannot determine length of %T", targetVal)}
		}
	}

	var results []string

	if minVal, ok := getFloat64Option(opts, "min"); ok {
		if value < minVal {
			results = append(results, fmt.Sprintf("must not be shorter than %g", minVal))
		}
	}

	if maxVal, ok := getFloat64Option(opts, "max"); ok {
		if value > maxVal {
			results = append(results, fmt.Sprintf("must not be longer than %g", maxVal))
		}
	}

	return results
}
