package openapitools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-slugify"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// ToKebabCase converts a string to kebab-case
// Handles camelCase, PascalCase, snake_case, and existing kebab-case
func ToKebabCase(s string) string {
	// First, replace underscores and spaces with hyphens
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")

	// Insert hyphens before uppercase letters (for camelCase/PascalCase)
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	s = re.ReplaceAllString(s, "${1}-${2}")

	// Convert to lowercase
	s = strings.ToLower(s)

	// Remove any double hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

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

// SanitizeRegexCapture will remove illegal characters from the path-variable name.
// The returned name will be valid for PCRE regex captures; Alphanumeric + '_', starting
// with [a-zA-Z].
func SanitizeRegexCapture(varName string, insoCompat bool) string {
	var regexName *slugify.Slugifier
	if insoCompat {
		regexName = (&slugify.Slugifier{}).ToLower(false).InvalidChar("_").WordSeparator("_")
	} else {
		regexName = (&slugify.Slugifier{}).ToLower(true).InvalidChar("_").WordSeparator("_")
	}
	return regexName.Slugify(varName)
}

// DereferenceJSONObject will dereference a JSON object.
func DereferenceJSONObject(
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

// ConvertYamlNodeToBytes will convert a yaml node to bytes.
func ConvertYamlNodeToBytes(node *yaml.Node) ([]byte, error) {
	var data interface{}
	err := node.Decode(&data)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(data)
}

// CrossProduct computes the Cartesian product of the input slices.
// The slices can be of any type
func CrossProduct(slices ...[]any) [][]any {
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

// GetXKongComponents returns a map of the '/components/x-kong/' object. If
// the extension is not there it will return an empty map. If the entry is not a
// yaml object, it will return an error.
func GetXKongComponents(doc v3.Document) (*map[string]interface{}, error) {
	var components map[string]interface{}

	if doc.Components == nil || doc.Components.Extensions == nil {
		return &map[string]interface{}{}, nil
	}

	xKongComponents, ok := doc.Components.Extensions.Get("x-kong")

	if !ok || xKongComponents == nil {
		return &components, nil
	}

	xKongComponentsBytes, err := ConvertYamlNodeToBytes(xKongComponents)
	if err != nil {
		return nil, fmt.Errorf("expected '/components/x-kong' to be a YAML object")
	}

	var xKong interface{}
	_ = yaml.Unmarshal(xKongComponentsBytes, &xKong)
	components, err = jsonbasics.ToObject(xKong)
	if err != nil {
		return nil, fmt.Errorf("expected '/components/x-kong' to be a JSON/YAML object")
	}

	return &components, nil
}

// GetXKongObject returns specified 'key' from the extension properties if available.
// Returns nil if it wasn't found, an error if it wasn't an object or couldn't be
// dereferenced. The returned object will be json encoded again.
func GetXKongObject(
	extensions *orderedmap.Map[string, *yaml.Node],
	key string, components *map[string]interface{},
) ([]byte, error) {
	if extensions == nil {
		return nil, nil
	}

	xKongObject, ok := extensions.Get(key)
	if !ok || xKongObject == nil {
		return nil, nil
	}

	xKongObjectBytes, err := ConvertYamlNodeToBytes(xKongObject)
	if err != nil {
		return nil, fmt.Errorf("expected '%s' to be a YAML object", key)
	}

	var jsonBlob interface{}
	_ = yaml.Unmarshal(xKongObjectBytes, &jsonBlob)
	jsonObject, err := jsonbasics.ToObject(jsonBlob)
	if err != nil {
		return nil, fmt.Errorf("expected '%s' to be a JSON/YAML object", key)
	}

	object, err := DereferenceJSONObject(jsonObject, components)
	if err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

// GetRouteDefaults returns a JSON string containing the route defaults
func GetRouteDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return GetXKongObject(extensions, "x-kong-route-defaults", components)
}
