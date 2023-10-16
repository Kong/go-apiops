package patch

// This file implements the '--selector' and '--value' CLI flags

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
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
//	'--value ["foo"]'     results in appending string "foo" to arrays
func ValidateValuesFlags(values []string) (map[string]interface{}, []string, []interface{}, error) {
	valuesMap := make(map[string]interface{})
	removeArr := make([]string, 0)
	appendArr := make([]interface{}, 0)

	for _, content := range values {
		if strings.HasPrefix(strings.TrimSpace(content), "[") &&
			strings.HasSuffix(strings.TrimSpace(content), "]") {
			// this is an array snippet
			var value interface{}
			err := json.Unmarshal([]byte(content), &value)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("expected '--value' entry to be a valid json array '[ entry1, entry2, ... ]', "+
					"failed parsing json-string in '%s'", content)
			}
			values, _ := jsonbasics.ToArray(value)

			logbasics.Debug("parsed patch-instruction", "array", value)
			appendArr = append(appendArr, values...)
		} else {
			// this is a key-value pair or delete-instruction
			subs := strings.SplitN(content, ":", 2)
			if len(subs) == 1 {
				return nil, nil, nil, fmt.Errorf("expected '--value' entry to have format 'key:json-string', "+
					"or '[ json-array ], got: '%s'", content)
			}

			key := subs[0]
			val := strings.TrimSpace(subs[1])

			var value interface{}
			if val == "" {
				// this is a delete-instruction, so inject the delete marker
				logbasics.Debug("parsed delete-instruction", "key", key)
				removeArr = append(removeArr, key)
			} else {
				// this is a key-value pair, parse the value
				err := json.Unmarshal([]byte(val), &value)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("expected '--value' entry to have format 'key:json-string', "+
						"failed parsing json-string in '%s' (did you forget to wrap a json-string-value in quotes?)",
						content)
				}
				logbasics.Debug("parsed patch-instruction", "key", key, "value", value)
				valuesMap[key] = value
			}
		}
	}

	return valuesMap, removeArr, appendArr, nil
}
