package patch

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

// ValidateValuesFlags parses the CLI '--values' keys formatted 'key:json-string', into
// a map. The map will hold the parsed JSON value by the key. If the value is 'nil' then
// the key is supposed to be deleted from the target object.
// Returns an error is value is not a valid JSON string. Important: strings
// must be quoted;
//
//	'--value foo:bar'     is invalid
//	'--value foo:"bar"'   results in string "bar"
//	'--value foo:true'    results in boolean true
//	'--value foo:'        results in deleting key 'foo' if it exists
func ValidateValuesFlags(values []string) (map[string]interface{}, error) {
	valuesMap := make(map[string]interface{})
	for _, content := range values {
		subs := strings.SplitN(content, ":", 2)
		if len(subs) == 1 {
			return nil, fmt.Errorf("expected '--value' entry to have format 'key:json-string', got: '%s'", content)
		}

		key := subs[0]
		val := strings.TrimSpace(subs[1])

		var value interface{}
		if val != "" {
			err := json.Unmarshal([]byte(val), &value)
			if err != nil {
				return nil, fmt.Errorf("expected '--value' entry to have format 'key:json-string', "+
					"failed parsing json-string in '%s' (did you forget to wrap a json-string-value in quotes?)",
					content)
			}
		}
		valuesMap[key] = value
	}

	return valuesMap, nil
}

// MustApplyPatches is identical to `ApplyPatches` except that it will panic instead
// of returning an error.
func MustApplyPatches(data map[string]interface{}, patchFiles []string) map[string]interface{} {
	result, err := ApplyPatches(data, patchFiles)
	if err != nil {
		panic(err)
	}
	return result
}

func ApplyPatches(data map[string]interface{}, patchFiles []string) (map[string]interface{}, error) {
	println(patchFiles)
	println(data)
	return nil, nil
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

func convertToYamlNode(data interface{}) *yaml.Node {
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

func convertToJSONInterface(data *yaml.Node) *interface{} {
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

func convertToJSONobject(data *yaml.Node) map[string]interface{} {
	jsonInterface := *convertToJSONInterface(data)
	jsonObject, err := jsonbasics.ToObject(jsonInterface)
	if err != nil {
		panic(err)
	}
	return jsonObject
}

//
//
//  End of workaround code
//
//

// MustApplyValues is identical to `ApplyValues` except that it will panic instead
// of returning an error.
func MustApplyValues(data map[string]interface{}, selector string, values map[string]interface{},
) map[string]interface{} {
	result, err := ApplyValues(data, selector, values)
	if err != nil {
		panic(err)
	}
	return result
}

func ApplyValues(data map[string]interface{}, selector string, values map[string]interface{},
) (map[string]interface{}, error) {
	// first validat the JSONpath, since all others have been validated
	pointer, err := yamlpath.NewPath(selector)
	if err != nil {
		return nil, err
	}

	yamlData := convertToYamlNode(data)
	nodes, err := pointer.Find(yamlData)
	if err != nil {
		return nil, err
	}

	// 'nodes' is an array of nodes matching the selector
	for _, node := range nodes {
		// since we're updating object fields, we'll skip anything that is
		// not a JSONobject
		if node.Kind == yaml.MappingNode {
			// So this Node is a JSONobject that we need to update

			// keep track of the fields we already processed
			handledFields := make(map[string]bool)

			// a mapping node has 2 entries for each key-value pair in its
			// node.Content array
			for i := 0; i < len(node.Content); {
				keyNode := node.Content[i]
				key := keyNode.Value

				newData, found := values[key]
				if found {
					if newData != nil {
						// we have an updated value for this key, set it
						node.Content[i+1] = convertToYamlNode(newData)
						i = i + 2 // move pointer forward
					} else {
						// delete the entry
						node.Content = append(node.Content[:i], node.Content[i+2:]...)
						// Note: not moving pointer forward, since we deleted elements
					}
					handledFields[key] = true
				} else {
					// no update, just move to next
					i = i + 2
				}
			}

			// update any field not handled yet (wasn't in the original object)
			for fieldName, newValue := range values {
				if !handledFields[fieldName] {
					keyNode := yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: fieldName,
						Style: yaml.DoubleQuotedStyle,
					}
					valueNode := convertToYamlNode(newValue)
					node.Content = append(node.Content, &keyNode, valueNode)
				}
			}
		}
	}

	return convertToJSONobject(yamlData), nil
}
