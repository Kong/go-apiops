package openapi2kong

import (
	"encoding/json"
	"fmt"
	"mime"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

const JSONSchemaVersion = "draft4"

// getDefaultParamStyles returns default styles per OAS parameter-type.
func getDefaultParamStyle(givenStyle string, paramType string) string {
	// should be a constant, but maps cannot be constants
	styles := map[string]string{
		"header": "simple",
		"cookie": "form",
		"query":  "form",
		"path":   "simple",
	}

	if givenStyle == "" {
		return styles[paramType]
	}
	return givenStyle
}

// generateParameterSchema returns the given schema if there is one, a generated
// schema if it was specified, or nil if there is none.
// Parameters include path, query, and headers
func generateParameterSchema(operation *v3.Operation, path *v3.PathItem, insoCompat bool) ([]map[string]interface{}, error) {
	pathParameters := path.Parameters
	operationParameters := operation.Parameters
	if pathParameters == nil && operationParameters == nil {
		return nil, nil
	}

	totalLength := len(pathParameters) + len(operationParameters)
	if totalLength == 0 {
		return nil, nil
	}

	combinedParameters := make([]*v3.Parameter, 0, totalLength)

	for _, pathParam := range pathParameters {
		for _, opParam := range operationParameters {
			// If path parameter and operation parameter share the same name and location
			// operation parameter overrides the path parameter. Thus, if this check passes,
			// Then we add the path param, else we skip it.
			if pathParam.Name != opParam.Name && pathParam.In != opParam.In {
				combinedParameters = append(combinedParameters, pathParam)
			}
		}
	}

	combinedParameters = append(combinedParameters, operationParameters...)

	result := make([]map[string]interface{}, len(combinedParameters))
	i := 0

	for _, parameter := range combinedParameters {
		if parameter != nil {
			style := getDefaultParamStyle(parameter.Style, parameter.In)

			var explode bool
			if parameter.Explode == nil {
				explode = (style == "form") // default to true for form style, false for all others
			} else {
				explode = *parameter.Explode
			}

			paramConf := make(map[string]interface{})
			paramConf["style"] = style
			paramConf["explode"] = explode
			paramConf["in"] = parameter.In
			if parameter.In == "path" {
				paramConf["name"] = sanitizeRegexCapture(parameter.Name, insoCompat)
			} else {
				paramConf["name"] = parameter.Name
			}

			if parameter.Required != nil {
				paramConf["required"] = parameter.Required
			} else {
				paramConf["required"] = false
			}

			schema, schemaMap := extractSchema(parameter.Schema)
			if schema != "" {
				paramConf["schema"] = schema

				_, typeStr, ok := fetchOneOfAndType(schemaMap)
				if ok && typeStr == "" {
					return nil, fmt.Errorf(`parameter schemas for request-validator plugin using oneOf must have a top-level type property`)
				}
			}

			result[i] = paramConf
			i++
		}
	}

	return result, nil
}

func parseMediaType(mediaType string) (string, string, error) {
	parsedMediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(parsedMediaType, "/")
	return parts[0], parts[1], nil
}

// generateBodySchema returns the given schema if there is one, a generated
// schema if it was specified, or "" if there is none.
func generateBodySchema(operation *v3.Operation) string {
	requestBody := operation.RequestBody
	if requestBody == nil {
		return ""
	}

	content := requestBody.Content
	if content == nil {
		return ""
	}

	contentItem := content.First()

	for contentItem != nil {
		contentType := contentItem.Key()
		contentValue := contentItem.Value()

		typ, subtype, err := parseMediaType(contentType)
		if err != nil {
			logbasics.Info("invalid MediaType '" + contentType + "' will be ignored")
			return ""
		}
		if typ == "application" && (subtype == "json" || strings.HasSuffix(subtype, "+json")) {
			schema, _ := extractSchema((*contentValue).Schema)
			return schema
		}

		contentItem = contentItem.Next()
	}

	return ""
}

// generateContentTypes returns an array of allowed content types. nil if none.
// Returned array will be sorted by name for deterministic comparisons.
func generateContentTypes(operation *v3.Operation) []string {
	requestBody := operation.RequestBody
	if requestBody == nil {
		return nil
	}

	content := requestBody.Content
	if content == nil {
		return nil
	}

	if content.Len() == 0 {
		return nil
	}

	list := make([]string, content.Len())
	i := 0
	contentItem := content.First()
	for contentItem != nil && i < len(list) {
		list[i] = contentItem.Key()
		i++
		contentItem = contentItem.Next()
	}
	sort.Strings(list)

	return list
}

// generateValidatorPlugin generates the validator plugin configuration, based
// on the JSON snippet, and the OAS inputs. This can return nil
func generateValidatorPlugin(operationConfigJSON []byte, operation *v3.Operation, path *v3.PathItem,
	uuidNamespace uuid.UUID, baseName string, skipID bool, insoCompat bool,
) (*map[string]interface{}, error) {
	if len(operationConfigJSON) == 0 {
		return nil, nil
	}
	logbasics.Debug("generating validator plugin", "operation", baseName)

	var pluginConfig map[string]interface{}
	_ = json.Unmarshal(operationConfigJSON, &pluginConfig)

	// create a new ID here based on the operation
	if !skipID {
		pluginConfig["id"] = createPluginID(uuidNamespace, baseName, pluginConfig)
	}

	config, _ := jsonbasics.ToObject(pluginConfig["config"])
	if config == nil {
		config = make(map[string]interface{})
		pluginConfig["config"] = config
	}

	if config["parameter_schema"] == nil {
		parameterSchema, err := generateParameterSchema(operation, path, insoCompat)
		if err != nil {
			return nil, err
		}
		if parameterSchema != nil {
			config["parameter_schema"] = parameterSchema
			config["version"] = JSONSchemaVersion
		}
	}

	if config["body_schema"] == nil {
		bodySchema := generateBodySchema(operation)
		if bodySchema != "" {
			config["body_schema"] = bodySchema
			config["version"] = JSONSchemaVersion
		} else {
			if config["parameter_schema"] == nil {
				// neither parameter nor body schema given, there is nothing to validate
				// unless the content-types have been provided by the user
				if config["allowed_content_types"] == nil {
					// also not provided, so really nothing to validate, don't add a plugin
					return nil, nil
				}
				// add an empty schema, which passes everything, but it also activates the
				// content-type check
				config["body_schema"] = "{}"
				config["version"] = JSONSchemaVersion
			}
		}
	}

	if config["allowed_content_types"] == nil {
		contentTypes := generateContentTypes(operation)
		if contentTypes != nil {
			config["allowed_content_types"] = contentTypes
		}
	}

	return &pluginConfig, nil
}

func fetchOneOfAndType(schemaMap map[string]interface{}) ([]interface{}, string, bool) {
	var oneOfSchemaArray []interface{}
	var typeStr string
	var oneOfFound bool

	// Check if oneOf exists at the current level
	if oneOf, ok := schemaMap["oneOf"]; ok {
		if slice, isSlice := oneOf.([]interface{}); isSlice {
			oneOfSchemaArray = slice
			oneOfFound = true
		}
	}

	// Check if type exists at the current level
	if typ, ok := schemaMap["type"]; ok {
		if str, isString := typ.(string); isString {
			typeStr = str
		}
	}

	// If both oneOf and type are found at this level, return them
	if oneOfFound && typeStr != "" {
		return oneOfSchemaArray, typeStr, true
	}

	// Recursively search in nested objects
	for _, value := range schemaMap {
		switch v := value.(type) {
		case map[string]interface{}:
			if slice, str, found := fetchOneOfAndType(v); found {
				return slice, str, true
			}
		case []interface{}:
			for _, item := range v {
				if itemMap, isMap := item.(map[string]interface{}); isMap {
					if slice, str, found := fetchOneOfAndType(itemMap); found {
						return slice, str, true
					}
				}
			}
		}
	}

	// If oneOf is found but type is not, return oneOf with empty type
	if oneOfFound {
		return oneOfSchemaArray, "", true
	}

	// If neither oneOf nor type is found
	return nil, "", false
}
