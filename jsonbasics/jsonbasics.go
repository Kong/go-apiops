package jsonbasics

import (
	"encoding/json"
	"fmt"
)

// ToObject returns the object, or nil+err
func ToObject(obj interface{}) (map[string]interface{}, error) {
	switch result := obj.(type) {
	case map[string]interface{}:
		return result, nil
	default:
		return nil, fmt.Errorf("not an object, but %t", obj)
	}
}

// ToArray returns the array, or nil+err
func ToArray(arr interface{}) ([]interface{}, error) {
	switch result := arr.(type) {
	case []interface{}:
		return result, nil
	}
	return nil, fmt.Errorf("not an array, but %t", arr)
}

// GetObjectArrayField returns a new slice containing all objects in the referenced field.
// If the field is not an array, it returns an error.
// If the field doesn't exist it returns an empty array.
// Any entry in the array that is not an object will be omitted from the returned slice.
func GetObjectArrayField(object map[string]interface{}, fieldName string) ([]map[string]interface{}, error) {
	target := object[fieldName]
	if target == nil {
		return make([]map[string]interface{}, 0), nil
	}

	arr, err := ToArray(target)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(arr))
	j := 0
	for _, expectedObject := range arr {
		service, err := ToObject(expectedObject)
		if err == nil {
			result[j] = service
			j++
		}
	}

	return result[:j], nil
}

// RemoveObjectFromArrayByFieldValue returns a slice in which objects that
// match the field value are removed. Returns; new slice, # of removals, err.
// occurrences determines the maximum number of items to remove, use -1 for unlimited.
// Only returns an error if inArr is given (non-nil), and it is not an array/slice. But even
// then it still returns the input value, so it is transparent.
func RemoveObjectFromArrayByFieldValue(inArr interface{},
	fieldName string, fieldValue string, occurrences int,
) (interface{}, int, error) {
	if occurrences < -1 {
		panic(fmt.Sprintf("occurrences cannot be smaller than -1, got %d", occurrences))
	}

	if inArr == nil || occurrences == 0 {
		return inArr, 0, nil
	}

	arr, err := ToArray(inArr)
	if err != nil {
		return inArr, 0, err
	}

	if occurrences == -1 {
		occurrences = len(arr)
	}

	targetIdx := 0
	count := 0
	for _, entry := range arr {
		isMatch := false
		obj, err := ToObject(entry)
		if err == nil {
			strValue, err := GetStringField(obj, fieldName)
			if err == nil && strValue == fieldValue && occurrences > 0 {
				isMatch = true
			}
		}

		if !isMatch {
			arr[targetIdx] = entry
			targetIdx++
			occurrences--
			count++
		}
	}

	for i := targetIdx; i < len(arr); i++ {
		arr[i] = nil
	}

	if targetIdx == 0 {
		return make([]interface{}, 0), 0, nil
	}

	return arr[:targetIdx], count, nil
}

func GetStringField(object map[string]interface{}, fieldName string) (string, error) {
	value := object[fieldName]
	switch result := value.(type) {
	case string:
		return result, nil
	}
	return "", fmt.Errorf("expected key '%s' to be a string, got %t", fieldName, value)
}

func GetStringIndex(arr []interface{}, index int) (string, error) {
	value := arr[index]
	switch result := value.(type) {
	case string:
		return result, nil
	}
	return "", fmt.Errorf("expected index '%d' to be a string, got %t", index, value)
}

func GetBoolField(object map[string]interface{}, fieldName string) (bool, error) {
	value := object[fieldName]
	switch result := value.(type) {
	case bool:
		return result, nil
	}
	return false, fmt.Errorf("expected key '%s' to be a boolean", fieldName)
}

func GetBoolIndex(arr []interface{}, index int) (bool, error) {
	value := arr[index]
	switch result := value.(type) {
	case bool:
		return result, nil
	}
	return false, fmt.Errorf("expected index '%d' to be a boolean", index)
}

// DeepCopy implements a poor man's deepcopy by jsonify/de-jsonify
func DeepCopy(data *map[string]interface{}) *map[string]interface{} {
	var dataCopy map[string]interface{}
	serialized, _ := json.Marshal(data)
	_ = json.Unmarshal(serialized, &dataCopy)
	return &dataCopy
}
