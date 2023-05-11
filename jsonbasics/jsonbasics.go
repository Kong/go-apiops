package jsonbasics

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ToObject returns the object, if it was one, or nil+err.
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

// GetObjectArrayField returns a new slice containing all objects from the array referenced by fieldName.
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
	for _, expectedObject := range arr {
		obj, err := ToObject(expectedObject)
		if err == nil {
			result = append(result, obj)
		}
	}

	return result, nil
}

// SetObjectArrayField sets an array in a parsed json object. This ensure it is of type
// []interface{}, such that a next call to GetObjectArrayField will work.
// If 'objectArray' is nil, then the field is deleted from the object.
func SetObjectArrayField(object map[string]interface{}, fieldName string, objectArray []map[string]interface{}) {
	if objectArray == nil {
		delete(object, fieldName)
		return
	}

	arr := make([]interface{}, len(objectArray))
	for i, obj := range objectArray {
		arr[i] = obj
	}
	object[fieldName] = arr
}

// GetStringArrayField returns a new slice containing all strings from the array referenced by fieldName.
// If the field is not an array, it returns an error.
// If the field doesn't exist it returns an empty array.
// Any entry in the array that is not a string will be omitted from the returned slice.
func GetStringArrayField(object map[string]interface{}, fieldName string) ([]string, error) {
	target := object[fieldName]
	if target == nil {
		return make([]string, 0), nil
	}

	arr, err := ToArray(target)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(arr))
	for _, expectedString := range arr {
		obj, ok := expectedString.(string)
		if ok {
			result = append(result, obj)
		}
	}

	return result, nil
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

// GetBoolField returns a boolean from an object field. Returns an error if the field
// is not a boolean, or is not found.
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

// DeepCopyObject implements a poor man's deepcopy by jsonify/de-jsonify
func DeepCopyObject(data *map[string]interface{}) *map[string]interface{} {
	var dataCopy map[string]interface{}
	serialized, _ := json.Marshal(data)
	_ = json.Unmarshal(serialized, &dataCopy)
	return &dataCopy
}

// DeepCopyArray implements a poor man's deepcopy by jsonify/de-jsonify
func DeepCopyArray(data *[]interface{}) *[]interface{} {
	var dataCopy []interface{}
	serialized, _ := json.Marshal(data)
	_ = json.Unmarshal(serialized, &dataCopy)
	return &dataCopy
}

//
//
//  Start of workaround code
//
//

// The JSONpath lib does not parse to interface{} types, but uses its own struct,
// yaml.Node. Hence anything read by the filebasics helpers must be converted back
// and forth, so we serialize and deserialze the data in an extra round-trip to get
// it into the proper structures.
//
// In library code, this takes a performance hit, in the CLI it's a lesser concern.

func ConvertToYamlNode(data interface{}) *yaml.Node {
	encData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	var yNode yaml.Node
	err = yaml.Unmarshal(encData, &yNode)
	if err != nil {
		panic(err)
	}
	if yNode.Kind == yaml.DocumentNode {
		return yNode.Content[0]
	}
	return &yNode
}

func ConvertToJSONInterface(data *yaml.Node) *interface{} {
	encData, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}
	var jsonData interface{}
	err = json.Unmarshal(encData, &jsonData)
	if err != nil {
		panic(err)
	}
	return &jsonData
}

func ConvertToJSONobject(data *yaml.Node) map[string]interface{} {
	jsonInterface := *ConvertToJSONInterface(data)
	jsonObject, err := ToObject(jsonInterface)
	if err != nil {
		panic(err)
	}
	return jsonObject
}
