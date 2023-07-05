// This package provides some basic functions for working with yaml nodes.
// The assumption is to never directly encode/decode yaml. Instead, we'll
// convert to/from interface{}.
package yamlbasics

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

//
//
//  parsing
//
//

// FromObject converts the given map[string]interface{} to an yaml node (map).
func FromObject(data map[string]interface{}) (*yaml.Node, error) {
	if data == nil {
		return nil, errors.New("not an object, but <nil>")
	}
	encData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var yNode yaml.Node
	err = yaml.Unmarshal(encData, &yNode)
	if err != nil {
		return nil, err
	}
	if yNode.Kind == yaml.DocumentNode {
		return yNode.Content[0], nil
	}
	return &yNode, nil
}

// ToObject converts the given yaml node to a map[string]interface{}.
func ToObject(data *yaml.Node) (map[string]interface{}, error) {
	if data == nil || data.Kind != yaml.MappingNode {
		return nil, errors.New("data is not a mapping node/object")
	}

	encData, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	var jsonData interface{}
	err = json.Unmarshal(encData, &jsonData)
	if err != nil {
		return nil, err
	}
	return jsonData.(map[string]interface{}), nil
}

// ToArray converts the given yaml node to a []interface{}.
func ToArray(data *yaml.Node) ([]interface{}, error) {
	if data == nil || data.Kind != yaml.SequenceNode {
		return nil, errors.New("data is not a sequence node/array")
	}

	encData, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	var jsonData interface{}
	err = json.Unmarshal(encData, &jsonData)
	if err != nil {
		return nil, err
	}
	return jsonData.([]interface{}), nil
}

// CopyNode creates a deep copy of the given node.
func CopyNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	nodeCopy := *node
	nodeCopy.Alias = nil // TODO: for now assume we do not use aliases
	nodeCopy.Content = nil
	for _, child := range node.Content {
		nodeCopy.Content = append(nodeCopy.Content, CopyNode(child))
	}

	return &nodeCopy
}

//
//
//  Handling objects and fields
//
//

// NewObject creates a new object node.
func NewObject() *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.MappingNode,
		Tag:   "!!map",
		Style: yaml.FlowStyle,
	}
}

// NewString creates a new string node.
func NewString(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
		Style: yaml.DoubleQuotedStyle,
	}
}

// FindFieldKeyIndex returns the index of the Node that contains the object-Key in the
// targets Content array. If the key is not found, it returns -1.
func FindFieldKeyIndex(targetObject *yaml.Node, key string) int {
	if targetObject.Kind != yaml.MappingNode {
		panic("targetObject is not a mapping node/object")
	}

	for i := 0; i < len(targetObject.Content); i += 2 {
		if targetObject.Content[i].Value == key {
			return i
		}
	}

	return -1
}

// FindFieldValueIndex returns the index of the Node that contains the object-Value in the
// targets Content array. If the value is not found, it returns -1.
func FindFieldValueIndex(targetObject *yaml.Node, key string) int {
	i := FindFieldKeyIndex(targetObject, key)
	if i != -1 {
		i++
	}

	return i
}

// RemoveFieldByIdx removes the key (by its index) and its value from the targetObject.
func RemoveFieldByIdx(targetObject *yaml.Node, idx int) {
	if idx < 0 || idx >= len(targetObject.Content) {
		panic("idx out of bounds")
	}
	targetObject.Content = append(targetObject.Content[:idx], targetObject.Content[idx+2:]...)
}

// RemoveField removes the given key and its value from the targetObject if it exists.
func RemoveField(targetObject *yaml.Node, key string) {
	if i := FindFieldKeyIndex(targetObject, key); i != -1 {
		RemoveFieldByIdx(targetObject, i)
	}
}

// GetFieldValue returns the value of the given key in the targetObject.
// If the key is not found, then nil is returned.
func GetFieldValue(targetObject *yaml.Node, key string) *yaml.Node {
	i := FindFieldValueIndex(targetObject, key)
	if i == -1 {
		return nil
	}
	return targetObject.Content[i]
}

// SetFieldValue sets/overwrites the value of the given key in the targetObject to the
// given value. If value is nil, then the key is removed from the targetObject if it exists.
func SetFieldValue(targetObject *yaml.Node, key string, value *yaml.Node) {
	i := FindFieldKeyIndex(targetObject, key)
	if i == -1 {
		// key not found, so field doesn't exist yet
		if value == nil {
			// nothing to do
			return
		}
		// add the field
		targetObject.Content = append(targetObject.Content, NewString(key), value)
		return
	}

	// key found, so field exists
	if value == nil {
		// remove the field
		RemoveFieldByIdx(targetObject, i)
		return
	}
	targetObject.Content[i+1] = value
}

//
//
//  Handling objects and fields
//
//

// NewArray creates a new array node.
func NewArray() *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.SequenceNode,
		Tag:   "!!seq",
		Style: yaml.FlowStyle,
	}
}

// Append adds the given values to the end of the targetArray. If no values are given,
// then nothing is done.
// If targetArray is nil or not a sequence node, then an error is returned.
// If any of the values are nil, then an error is returned (and the array remains unchanged).
func Append(targetArray *yaml.Node, values ...*yaml.Node) error {
	if targetArray == nil || targetArray.Kind != yaml.SequenceNode {
		return errors.New("targetArray is not a sequence node/array")
	}
	for i, value := range values {
		if value == nil {
			return fmt.Errorf("value at index %d is nil", i)
		}
	}

	targetArray.Content = append(targetArray.Content, values...)
	return nil
}

// AppendSlice appends all entries in a slice to the end of the targetArray.
// If targetArray is nil or not a sequence node, then an error is returned.
// If the slice is nil, then nothing is done.
// If any of the values in the slice are nil, then an error is returned (and the array remains unchanged).
func AppendSlice(targetArray *yaml.Node, values []*yaml.Node) error {
	if targetArray == nil || targetArray.Kind != yaml.SequenceNode {
		return errors.New("targetArray is not a sequence node/array")
	}
	if values == nil {
		return nil
	}

	err := Append(targetArray, values...)
	if err != nil {
		return err
	}
	return nil
}

// YamlArrayMatcher is a type of function passed to Search. To match a node against
// the search criteria, the function should return true if it matches.
type YamlArrayMatcher func(*yaml.Node) (bool, error)

// YamlArrayIterator is a type of function returned by Search. On each call, it returns
// the next matching node in the targetArray. If no more matches are found, then nil is
// returned. The second return value is the index of the node in the targetArray.
type YamlArrayIterator func() (*yaml.Node, int, error)

// Search returns a YamlArrayIterator function that can be called repeatedly to find the next matching
// node in the targetArray. If no more matches are found, then nil is returned.
// The search is resilient against changing the targetArray while searching.
// The YamlArrayMatcher function is called with each (non-nil) node in the targetArray.
// If the match function
// returns true, then the node is returned. If the match function returns false, then
// the next node is checked. If the match function returns an error, then the search
// is aborted and the error is returned (calling again returns the same error).
func Search(targetArray *yaml.Node, match YamlArrayMatcher) YamlArrayIterator {
	if targetArray == nil || targetArray.Kind != yaml.SequenceNode {
		panic("targetArray is not a sequence node/array")
	}

	refs := make(map[*yaml.Node]bool)
	done := false
	var err error
	var idx int

	return func() (*yaml.Node, int, error) {
		if !done {
			for i := 0; i < len(targetArray.Content); i++ {
				res := targetArray.Content[i]
				if res != nil && !refs[res] {
					var matched bool
					matched, err = match(res)
					if err != nil {
						done = true
						idx = i
						return nil, idx, err
					}
					if matched {
						refs[res] = true
						return res, i, nil
					}
				}
			}
			done = true
		}
		return nil, idx, err
	}
}
