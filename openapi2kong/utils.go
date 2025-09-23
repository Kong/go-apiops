package openapi2kong

import (
	"fmt"
	"strings"

	"github.com/kong/go-slugify"
	"gopkg.in/yaml.v3"
)

// Slugify converts a name to a valid Kong name by removing and replacing unallowed characters
// and sanitizing non-latin characters. Multiple inputs will be concatenated using '_'.
func Slugify(insoCompat bool, name ...string) string {
	var (
		slugifier *slugify.Slugifier
		concatBy  string
	)
	if insoCompat {
		slugifier = (&slugify.Slugifier{}).ToLower(false).InvalidChar("_").WordSeparator("_")
		slugifier.AllowedSet("a-zA-Z0-9\\-")
		concatBy = "-"
	} else {
		slugifier = (&slugify.Slugifier{}).ToLower(true).InvalidChar("-").WordSeparator("-")
		concatBy = "_"
	}

	for i, elem := range name {
		name[i] = slugifier.Slugify(elem)
	}

	// drop empty strings from the array
	for i := 0; i < len(name); i++ {
		if name[i] == "" {
			name = append(name[:i], name[i+1:]...)
			i--
		}
	}

	return strings.Join(name, concatBy)
}

// sanitizeRegexCapture will remove illegal characters from the path-variable name.
// The returned name will be valid for PCRE regex captures; Alphanumeric + '_', starting
// with [a-zA-Z].
func sanitizeRegexCapture(varName string, insoCompat bool) string {
	var regexName *slugify.Slugifier
	if insoCompat {
		regexName = (&slugify.Slugifier{}).ToLower(false).InvalidChar("_").WordSeparator("_")
	} else {
		regexName = (&slugify.Slugifier{}).ToLower(true).InvalidChar("_").WordSeparator("_")
	}
	return regexName.Slugify(varName)
}

func dereferenceJSONObject(
	value map[string]interface{},
	components *map[string]interface{},
) (map[string]interface{}, error) {
	var pointer string

	switch value["$ref"].(type) {
	case nil: // it is not a reference, so return the object
		return value, nil

	case string: // it is a json pointer
		pointer = value["$ref"].(string)
		if !strings.HasPrefix(pointer, "#/components/x-kong/") {
			return nil, fmt.Errorf("all 'x-kong-...' references must be at '#/components/x-kong/...'")
		}

	default: // bad pointer
		return nil, fmt.Errorf("expected '$ref' pointer to be a string")
	}

	// walk the tree to find the reference
	segments := strings.Split(pointer, "/")
	path := "#/components/x-kong"
	result := components

	for i := 3; i < len(segments); i++ {
		segment := segments[i]
		path = path + "/" + segment

		switch (*result)[segment].(type) {
		case nil:
			return nil, fmt.Errorf("reference '%s' not found", pointer)
		case map[string]interface{}:
			target := (*result)[segment].(map[string]interface{})
			result = &target
		default:
			return nil, fmt.Errorf("expected '%s' to be a JSON object", path)
		}
	}

	return *result, nil
}

func convertYamlNodeToBytes(node *yaml.Node) ([]byte, error) {
	var data interface{}
	err := node.Decode(&data)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(data)
}

// crossProduct computes the Cartesian product of the input slices.
// The slices can be of any type
func crossProduct(slices ...[]any) [][]any {
	if len(slices) == 0 {
		return [][]any{{}}
	}

	result := [][]any{{}}
	for _, slice := range slices {
		var next [][]any
		for _, prefix := range result {
			for _, elem := range slice {
				newTuple := append([]any{}, prefix...)
				newTuple = append(newTuple, elem)
				next = append(next, newTuple)
			}
		}
		result = next
	}

	return result
}
