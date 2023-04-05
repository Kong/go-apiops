package patch

// This file implements the '--selector' and '--value' CLI flags

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
	"gopkg.in/yaml.v3"
)

// ValidateValuesFlags parses the CLI '--values' keys formatted 'key:json-string', into
// a map. The map will hold the parsed JSON value by the key. The second return value is an
// array of field names that is supposed to be deleted form the target.
// Returns an error is value is not a valid JSON string. Important: strings
// must be quoted;
//
//	'--value foo:bar'     is invalid
//	'--value foo:"bar"'   results in string "bar"
//	'--value foo:true'    results in boolean true
//	'--value foo:'        results in deleting key 'foo' if it exists
func ValidateValuesFlags(values []string) (map[string]interface{}, []string, error) {
	valuesMap := make(map[string]interface{})
	removeArr := make([]string, 0)

	for _, content := range values {
		subs := strings.SplitN(content, ":", 2)
		if len(subs) == 1 {
			return nil, nil, fmt.Errorf("expected '--value' entry to have format 'key:json-string', got: '%s'", content)
		}

		key := subs[0]
		val := strings.TrimSpace(subs[1])

		var value interface{}
		if val == "" {
			// this is a delete-instruction, so inject the delete marker
			removeArr = append(removeArr, key)
		} else {
			err := json.Unmarshal([]byte(val), &value)
			if err != nil {
				return nil, nil, fmt.Errorf("expected '--value' entry to have format 'key:json-string', "+
					"failed parsing json-string in '%s' (did you forget to wrap a json-string-value in quotes?)",
					content)
			}
			valuesMap[key] = value
		}
	}

	return valuesMap, removeArr, nil
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
	jsonObject, err := jsonbasics.ToObject(jsonInterface)
	if err != nil {
		panic(err)
	}
	return jsonObject
}
