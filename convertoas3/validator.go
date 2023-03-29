package convertoas3

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/kong/go-apiops/jsonbasics"
	uuid "github.com/satori/go.uuid"
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
func generateParameterSchema(operation *openapi3.Operation) *[]map[string]interface{} {
	parameters := operation.Parameters
	if parameters == nil {
		return nil
	}

	if len(parameters) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, len(parameters))
	i := 0
	for _, parameterRef := range parameters {
		paramValue := parameterRef.Value

		var explode bool
		if paramValue.Explode == nil {
			explode = false
		} else {
			explode = *paramValue.Explode
		}

		if paramValue != nil {
			paramConf := make(map[string]interface{})
			paramConf["explode"] = explode
			paramConf["in"] = paramValue.In
			if paramValue.In == "path" {
				paramConf["name"] = sanitizeRegexCapture(paramValue.Name)
			} else {
				paramConf["name"] = paramValue.Name
			}
			paramConf["required"] = paramValue.Required
			paramConf["style"] = getDefaultParamStyle(paramValue.Style, paramValue.In)

			schema := extractSchema(paramValue.Schema)
			if schema != "" {
				paramConf["schema"] = schema
			}

			result[i] = paramConf
			i++
		}
	}

	return &result
}

// generateBodySchema returns the given schema if there is one, a generated
// schema if it was specified, or "" if there is none.
func generateBodySchema(operation *openapi3.Operation) string {
	requestBody := operation.RequestBody
	if requestBody == nil {
		return ""
	}

	requestBodyValue := requestBody.Value
	if requestBodyValue == nil {
		return ""
	}

	content := requestBodyValue.Content
	if content == nil {
		return ""
	}

	for contentType, content := range content {
		if strings.Contains(strings.ToLower(contentType), "application/json") {
			return extractSchema((*content).Schema)
		}
	}

	return ""
}

// generateContentTypes returns an array of allowed content types. nil if none.
// Returned array will be sorted by name for deterministic comparisons.
func generateContentTypes(operation *openapi3.Operation) *[]string {
	requestBody := operation.RequestBody
	if requestBody == nil {
		return nil
	}

	requestBodyValue := requestBody.Value
	if requestBodyValue == nil {
		return nil
	}

	content := requestBodyValue.Content
	if content == nil {
		return nil
	}

	if len(content) == 0 {
		return nil
	}

	list := make([]string, len(content))
	i := 0
	for contentType := range content {
		list[i] = contentType
		i++
	}
	sort.Strings(list)

	return &list
}

// generateValidatorPlugin generates the validator plugin configuration, based
// on the JSON snippet, and the OAS inputs. This can return nil
func generateValidatorPlugin(configJSON []byte, operation *openapi3.Operation,
	uuidNamespace uuid.UUID,
	baseName string,
) *map[string]interface{} {
	if len(configJSON) == 0 {
		return nil
	}

	var pluginConfig map[string]interface{}
	_ = json.Unmarshal(configJSON, &pluginConfig)

	// create a new ID here based on the operation
	pluginConfig["id"] = createPluginID(uuidNamespace, baseName, pluginConfig)

	config, _ := jsonbasics.ToObject(pluginConfig["config"])
	if config == nil {
		config = make(map[string]interface{})
		pluginConfig["config"] = config
	}

	if config["parameter_schema"] == nil {
		parameterSchema := generateParameterSchema(operation)
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
					return nil
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

	return &pluginConfig
}
