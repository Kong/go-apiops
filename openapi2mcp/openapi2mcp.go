package openapi2mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/openapi2kong"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	openapibase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const (
	// MCP proxy modes
	ModeConversionListener = "conversion-listener"
	ModeConversion         = "conversion"
)

// O2MOptions defines the options for an OpenAPI to MCP conversion operation
type O2MOptions struct {
	// Array of tags to mark all generated entities with, taken from 'x-kong-tags' if omitted.
	Tags []string
	// Base document name, will be taken from x-kong-name, or info.title (for UUID generation!)
	DocName string
	// Namespace for UUID generation, defaults to DNS namespace for UUID v5
	UUIDNamespace uuid.UUID
	// Skip ID generation (UUIDs)
	SkipID bool
	// MCP proxy mode: "conversion" or "conversion-listener"
	Mode string
	// Custom path prefix for the MCP route (default: /{service-name}-mcp)
	PathPrefix string
	// Also generate direct (non-MCP) routes for API access
	IncludeDirectRoute bool
	// Ignore security errors (unsupported schemes, missing x-kong-mcp-acl extension)
	IgnoreSecurityErrors bool
}

// setDefaults sets the defaults for the OpenAPI2MCP operation.
func (opts *O2MOptions) setDefaults() {
	var emptyUUID uuid.UUID

	if bytes.Equal(emptyUUID[:], opts.UUIDNamespace[:]) {
		opts.UUIDNamespace = uuid.NameSpaceDNS
	}

	if opts.Mode == "" {
		opts.Mode = ModeConversionListener
	}
}

// toKebabCase converts a string to kebab-case
// Handles camelCase, PascalCase, snake_case, and existing kebab-case
func toKebabCase(s string) string {
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

// getXKongComponents will return a map of the '/components/x-kong/' object.
func getXKongComponents(doc v3.Document) (*map[string]interface{}, error) {
	var components map[string]interface{}

	if doc.Components == nil || doc.Components.Extensions == nil {
		return &map[string]interface{}{}, nil
	}

	xKongComponents, ok := doc.Components.Extensions.Get("x-kong")
	if !ok || xKongComponents == nil {
		return &components, nil
	}

	xKongComponentsBytes, err := convertYamlNodeToBytes(xKongComponents)
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

// convertYamlNodeToBytes converts a YAML node to bytes
func convertYamlNodeToBytes(node *yaml.Node) ([]byte, error) {
	var data interface{}
	err := node.Decode(&data)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(data)
}

// getXKongObject returns specified 'key' from the extension properties if available.
func getXKongObject(
	extensions *orderedmap.Map[string, *yaml.Node],
	key string,
	components *map[string]interface{},
) ([]byte, error) {
	if extensions == nil {
		return nil, nil
	}

	xKongObject, ok := extensions.Get(key)
	if !ok || xKongObject == nil {
		return nil, nil
	}

	xKongObjectBytes, err := convertYamlNodeToBytes(xKongObject)
	if err != nil {
		return nil, fmt.Errorf("expected '%s' to be a YAML object", key)
	}

	var jsonBlob interface{}
	_ = yaml.Unmarshal(xKongObjectBytes, &jsonBlob)
	jsonObject, err := jsonbasics.ToObject(jsonBlob)
	if err != nil {
		return nil, fmt.Errorf("expected '%s' to be a JSON/YAML object", key)
	}

	object, err := dereferenceJSONObject(jsonObject, components)
	if err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

// dereferenceJSONObject resolves $ref pointers within x-kong extensions
func dereferenceJSONObject(
	value map[string]interface{},
	components *map[string]interface{},
) (map[string]interface{}, error) {
	var pointer string

	switch value["$ref"].(type) {
	case nil:
		return value, nil
	case string:
		pointer = value["$ref"].(string)
		if !strings.HasPrefix(pointer, "#/components/x-kong/") {
			return nil, fmt.Errorf("all 'x-kong-...' references must be at '#/components/x-kong/...'")
		}
	default:
		return nil, fmt.Errorf("expected '$ref' pointer to be a string")
	}

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

// getRouteDefaults returns a JSON string containing the route defaults
func getRouteDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-route-defaults", components)
}

// getMCPProxyConfig returns the x-kong-mcp-proxy override config
func getMCPProxyConfig(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-mcp-proxy", components)
}

// getMCPACLConfig reads the x-kong-mcp-acl extension from a security scheme and returns the
// ACL configuration (acl_attribute_type, access_token_claim_field).
func getMCPACLConfig(scheme *v3.SecurityScheme) (map[string]interface{}, error) {
	if scheme == nil || scheme.Extensions == nil {
		return nil, nil
	}

	node, ok := scheme.Extensions.Get("x-kong-mcp-acl")
	if !ok || node == nil {
		return nil, nil
	}

	nodeBytes, err := convertYamlNodeToBytes(node)
	if err != nil {
		return nil, fmt.Errorf("expected 'x-kong-mcp-acl' to be a YAML object: %w", err)
	}

	var aclConfig map[string]interface{}
	err = yaml.Unmarshal(nodeBytes, &aclConfig)
	if err != nil {
		return nil, fmt.Errorf("expected 'x-kong-mcp-acl' to be a YAML object: %w", err)
	}

	return aclConfig, nil
}

// getMCPDefaultACL reads the x-kong-mcp-default-acl extension from document-level extensions
// and returns the default ACL array (e.g. [{scope: "tools", allow: ["flights:read"]}]).
func getMCPDefaultACL(extensions *orderedmap.Map[string, *yaml.Node]) ([]interface{}, error) {
	if extensions == nil {
		return nil, nil
	}

	node, ok := extensions.Get("x-kong-mcp-default-acl")
	if !ok || node == nil {
		return nil, nil
	}

	nodeBytes, err := convertYamlNodeToBytes(node)
	if err != nil {
		return nil, fmt.Errorf("expected 'x-kong-mcp-default-acl' to be a YAML array: %w", err)
	}

	var defaultACL []interface{}
	err = yaml.Unmarshal(nodeBytes, &defaultACL)
	if err != nil {
		return nil, fmt.Errorf("expected 'x-kong-mcp-default-acl' to be a YAML array: %w", err)
	}

	return defaultACL, nil
}

// getOperationACL extracts ACL scopes from an operation's security requirements.
// If the operation has no security field, it inherits from document-level security.
// Returns a map with "allow" key containing sorted scope strings, or nil if no security applies.
func getOperationACL(
	operationSecurity []*openapibase.SecurityRequirement,
	docSecurity []*openapibase.SecurityRequirement,
	doc v3.Document,
	ignoreSecurityErrors bool,
) (map[string]interface{}, error) {
	// Determine effective security: operation-level overrides document-level
	security := operationSecurity
	if security == nil {
		// No operation-level security; inherit from document
		security = docSecurity
	}

	if len(security) == 0 {
		// No security requirements at all
		return nil, nil
	}

	// Check for explicit empty security (security: []) which opts out
	if len(security) == 1 && security[0].ContainsEmptyRequirement {
		return nil, nil
	}

	if len(security) > 1 {
		// Multiple security requirements represent OR logic
		if ignoreSecurityErrors {
			return nil, nil
		}
		return nil, fmt.Errorf("only a single security requirement is supported per operation for MCP ACL generation")
	}

	requirement := security[0].Requirements
	if requirement == nil || requirement.Len() == 0 {
		return nil, nil
	}

	if requirement.Len() > 1 {
		// Multiple schemes within one requirement is AND logic
		if ignoreSecurityErrors {
			return nil, nil
		}
		return nil, fmt.Errorf("only a single security scheme per requirement is supported for MCP ACL generation")
	}

	// Extract the single scheme name and scopes
	reqPair := requirement.First()
	schemeName := reqPair.Key()
	scopes := reqPair.Value()

	// Validate the scheme exists and is oauth2
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		if ignoreSecurityErrors {
			return nil, nil
		}
		return nil, fmt.Errorf("no security schemes defined in components")
	}

	scheme, ok := doc.Components.SecuritySchemes.Get(schemeName)
	if !ok || scheme == nil {
		if ignoreSecurityErrors {
			return nil, nil
		}
		return nil, fmt.Errorf("security scheme '%s' not found in components/securitySchemes", schemeName)
	}

	if strings.ToLower(scheme.Type) != "oauth2" {
		if ignoreSecurityErrors {
			return nil, nil
		}
		return nil, fmt.Errorf("only 'oauth2' security schemes are supported for MCP ACL generation, got '%s'", scheme.Type)
	}

	// Validate the scheme has x-kong-mcp-acl extension
	aclConfig, err := getMCPACLConfig(scheme)
	if err != nil {
		return nil, err
	}
	if aclConfig == nil {
		if ignoreSecurityErrors {
			return nil, nil
		}
		return nil, fmt.Errorf("oauth2 security scheme '%s' is missing the 'x-kong-mcp-acl' extension", schemeName)
	}

	if len(scopes) == 0 {
		return nil, nil
	}

	// Sort scopes for deterministic output
	sortedScopes := make([]string, len(scopes))
	copy(sortedScopes, scopes)
	sort.Strings(sortedScopes)

	return map[string]interface{}{
		"allow": sortedScopes,
	}, nil
}

// findACLSecurityScheme looks through the document's security schemes for the first oauth2 scheme
// that has an x-kong-mcp-acl extension. Returns the scheme and its ACL config, or nil if none found.
func findACLSecurityScheme(doc v3.Document) (*v3.SecurityScheme, map[string]interface{}, error) {
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return nil, nil, nil
	}

	for pair := doc.Components.SecuritySchemes.First(); pair != nil; pair = pair.Next() {
		scheme := pair.Value()
		if scheme == nil || strings.ToLower(scheme.Type) != "oauth2" {
			continue
		}

		aclConfig, err := getMCPACLConfig(scheme)
		if err != nil {
			return nil, nil, err
		}
		if aclConfig != nil {
			return scheme, aclConfig, nil
		}
	}

	return nil, nil, nil
}

// getExtensionBool returns a boolean value from an extension
func getExtensionBool(extensions *orderedmap.Map[string, *yaml.Node], key string) (bool, error) {
	if extensions == nil {
		return false, nil
	}

	node, ok := extensions.Get(key)
	if !ok || node == nil {
		return false, nil
	}

	var value bool
	err := yaml.Unmarshal([]byte(node.Value), &value)
	if err != nil {
		return false, fmt.Errorf("expected '%s' to be a boolean: %w", key, err)
	}

	return value, nil
}

// getExtensionString returns a string value from an extension
func getExtensionString(extensions *orderedmap.Map[string, *yaml.Node], key string) (string, error) {
	if extensions == nil {
		return "", nil
	}

	node, ok := extensions.Get(key)
	if !ok || node == nil {
		return "", nil
	}

	var value string
	err := yaml.Unmarshal([]byte(node.Value), &value)
	if err != nil {
		return "", fmt.Errorf("expected '%s' to be a string: %w", key, err)
	}

	return value, nil
}

// simplifySchema simplifies an OpenAPI schema to essential properties
func simplifySchema(schema *openapibase.Schema) map[string]interface{} {
	if schema == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Handle type
	if len(schema.Type) > 0 {
		result["type"] = schema.Type[0]
	}

	// For objects, include properties and required
	if result["type"] == "object" && schema.Properties != nil {
		props := make(map[string]interface{})
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			propSchema := pair.Value().Schema()
			props[pair.Key()] = simplifySchema(propSchema)
		}
		result["properties"] = props

		if len(schema.Required) > 0 {
			result["required"] = schema.Required
		}
	}

	// For arrays, include items
	if result["type"] == "array" && schema.Items != nil && schema.Items.A != nil {
		result["items"] = simplifySchema(schema.Items.A.Schema())
	}

	return result
}

// buildParameters builds the parameters array for an MCP tool
func buildParameters(params []*v3.Parameter) []map[string]interface{} {
	if len(params) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(params))
	for _, param := range params {
		if param == nil {
			continue
		}

		p := map[string]interface{}{
			"name":     param.Name,
			"in":       param.In,
			"required": param.Required != nil && *param.Required,
		}

		if param.Description != "" {
			p["description"] = param.Description
		}

		if param.Schema != nil {
			schema := param.Schema.Schema()
			if schema != nil {
				p["schema"] = simplifySchema(schema)
			}
		}

		result = append(result, p)
	}

	return result
}

// buildRequestBody builds the request_body object for an MCP tool
func buildRequestBody(rb *v3.RequestBody) map[string]interface{} {
	if rb == nil {
		return nil
	}

	result := map[string]interface{}{}

	if rb.Required != nil {
		result["required"] = *rb.Required
	}

	if rb.Content != nil {
		content := make(map[string]interface{})
		for pair := rb.Content.First(); pair != nil; pair = pair.Next() {
			mediaType := pair.Key()
			mediaTypeObj := pair.Value()

			mediaContent := make(map[string]interface{})
			if mediaTypeObj.Schema != nil {
				schema := mediaTypeObj.Schema.Schema()
				if schema != nil {
					mediaContent["schema"] = simplifySchema(schema)
				}
			}
			content[mediaType] = mediaContent
		}
		result["content"] = content
	}

	return result
}

// buildMCPTool builds an MCP tool definition from an OAS operation
func buildMCPTool(
	path string,
	method string,
	operation *v3.Operation,
	pathParams []*v3.Parameter,
	acl map[string]interface{},
) (map[string]interface{}, error) {
	// Get tool name: x-kong-mcp-tool-name > operationId
	toolName, err := getExtensionString(operation.Extensions, "x-kong-mcp-tool-name")
	if err != nil {
		return nil, err
	}
	if toolName == "" {
		toolName = toKebabCase(operation.OperationId)
	}
	if toolName == "" {
		// Fallback: generate from method + path
		toolName = toKebabCase(strings.ToLower(method) + "-" + strings.ReplaceAll(path, "/", "-"))
	}

	// Get tool description: x-kong-mcp-tool-description > description > summary
	toolDesc, err := getExtensionString(operation.Extensions, "x-kong-mcp-tool-description")
	if err != nil {
		return nil, err
	}
	if toolDesc == "" {
		if operation.Description != "" {
			toolDesc = operation.Description
		} else if operation.Summary != "" {
			toolDesc = operation.Summary
		}
	}

	tool := map[string]interface{}{
		"name":   toolName,
		"method": strings.ToUpper(method),
		"path":   path,
	}

	if toolDesc != "" {
		tool["description"] = toolDesc
	}

	// Add annotations with title
	if operation.Summary != "" {
		tool["annotations"] = map[string]interface{}{
			"title": operation.Summary,
		}
	}

	// Merge path-level and operation-level parameters
	allParams := make([]*v3.Parameter, 0)
	paramNames := make(map[string]bool)

	// Operation params take precedence
	for _, p := range operation.Parameters {
		if p != nil {
			allParams = append(allParams, p)
			paramNames[p.Name] = true
		}
	}

	// Add path params that aren't overridden
	for _, p := range pathParams {
		if p != nil && !paramNames[p.Name] {
			allParams = append(allParams, p)
		}
	}

	if len(allParams) > 0 {
		tool["parameters"] = buildParameters(allParams)
	}

	// Add request body
	if operation.RequestBody != nil {
		tool["request_body"] = buildRequestBody(operation.RequestBody)
	}

	// Add ACL if provided
	if acl != nil {
		tool["acl"] = acl
	}

	return tool, nil
}

// MustConvert is the same as Convert, but will panic if an error is returned.
func MustConvert(content []byte, opts O2MOptions) map[string]interface{} {
	result, err := Convert(content, opts)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

// Convert converts an OpenAPI spec to a Kong declarative file with MCP configuration.
func Convert(content []byte, opts O2MOptions) (map[string]interface{}, error) {
	opts.setDefaults()
	logbasics.Debug("received OpenAPI2MCP options", "options", opts)

	// convert to openapi2kong options
	o2kOpts := openapi2kong.O2kOptions{
		Tags:          opts.Tags,
		DocName:       opts.DocName,
		UUIDNamespace: opts.UUIDNamespace,
		SkipID:        opts.SkipID,
	}

	// generate the base Kong configuration
	result, err := openapi2kong.Convert(content, o2kOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate base Kong configuration: %w", err)
	}

	// Load and parse the OAS file to get the v3 model
	openapiDoc, err := libopenapi.NewDocument(content)
	if err != nil {
		return nil, fmt.Errorf("error parsing OAS3 file: [%w]", err)
	}
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.IgnoreArrayCircularReferences = true
	docConfig.IgnorePolymorphicCircularReferences = true
	openapiDoc.SetConfiguration(docConfig)
	v3Model, errs := openapiDoc.BuildV3Model()
	if errs != nil {
		logbasics.Error(errs, "error while building v3 document model")
		return nil, fmt.Errorf("cannot create v3 model from document: %w", errs)
	}
	var doc v3.Document
	if v3Model != nil {
		doc = v3Model.Model
	}

	// get the main service
	services, ok := result["services"].([]interface{})
	if !ok || len(services) == 0 {
		return nil, fmt.Errorf("no services generated")
	}
	docService, ok := services[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("generated service is not a valid object")
	}
	docBaseName := docService["name"].(string)

	// handle routes
	if !opts.IncludeDirectRoute {
		docService["routes"] = make([]interface{}, 0)
	}
	routes := docService["routes"].([]interface{})

	// get kong components and defaults
	kongComponents, err := getXKongComponents(doc)
	if err != nil {
		return nil, err
	}
	docRouteDefaults, err := getRouteDefaults(doc.Extensions, kongComponents)
	if err != nil {
		return nil, err
	}

	// Detect ACL security configuration
	_, aclConfig, err := findACLSecurityScheme(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to read security scheme ACL config: %w", err)
	}

	// Read default ACL from document-level extension
	var defaultACL []interface{}
	if aclConfig != nil {
		defaultACL, err = getMCPDefaultACL(doc.Extensions)
		if err != nil {
			return nil, fmt.Errorf("failed to read default ACL: %w", err)
		}
	}

	// Build MCP tools from all operations
	tools := make([]interface{}, 0)
	if doc.Paths != nil {
		allPaths := doc.Paths.PathItems
		sortedPaths := make([]string, 0, allPaths.Len())
		path := allPaths.First()
		for path != nil {
			sortedPaths = append(sortedPaths, path.Key())
			path = path.Next()
		}
		sort.Strings(sortedPaths)

		for _, pathKey := range sortedPaths {
			pathItem, ok := allPaths.Get(pathKey)
			if !ok {
				continue
			}

			operations := pathItem.GetOperations()
			sortedMethods := make([]string, 0, operations.Len())
			method := operations.First()
			for method != nil {
				sortedMethods = append(sortedMethods, method.Key())
				method = method.Next()
			}
			sort.Strings(sortedMethods)

			for _, methodKey := range sortedMethods {
				operation, ok := operations.Get(methodKey)
				if !ok {
					continue
				}

				excluded, err := getExtensionBool(operation.Extensions, "x-kong-mcp-exclude")
				if err != nil {
					return nil, err
				}
				if excluded {
					continue
				}

				var toolACL map[string]interface{}
				if aclConfig != nil {
					toolACL, err = getOperationACL(operation.Security, doc.Security, doc, opts.IgnoreSecurityErrors)
					if err != nil {
						return nil, fmt.Errorf("failed to get ACL for %s %s: %w", methodKey, pathKey, err)
					}
				}

				tool, err := buildMCPTool(pathKey, methodKey, operation, pathItem.Parameters, toolACL)
				if err != nil {
					return nil, fmt.Errorf("failed to build MCP tool for %s %s: %w", methodKey, pathKey, err)
				}
				tools = append(tools, tool)
			}
		}
	}

	// Build the MCP route
	mcpRouteName := docBaseName + "-mcp"
	mcpRoutePath := opts.PathPrefix
	if mcpRoutePath == "" {
		mcpRoutePath = "/" + mcpRouteName
	}

	mcpRoute := make(map[string]interface{})
	if docRouteDefaults != nil {
		_ = json.Unmarshal(docRouteDefaults, &mcpRoute)
		delete(mcpRoute, "service")
	}

	if !opts.SkipID {
		mcpRoute["id"] = uuid.NewSHA1(opts.UUIDNamespace, []byte(mcpRouteName+".route")).String()
	}
	mcpRoute["name"] = mcpRouteName
	mcpRoute["paths"] = []string{mcpRoutePath}
	mcpRoute["tags"] = docService["tags"]

	// Build ai-mcp-proxy plugin config
	mcpPluginConfig := map[string]interface{}{
		"mode":  opts.Mode,
		"tools": tools,
	}

	if aclConfig != nil {
		if v, ok := aclConfig["acl_attribute_type"]; ok {
			mcpPluginConfig["acl_attribute_type"] = v
		}
		if v, ok := aclConfig["access_token_claim_field"]; ok {
			mcpPluginConfig["access_token_claim_field"] = v
		}
	}
	if defaultACL != nil {
		mcpPluginConfig["default_acl"] = defaultACL
	}

	mcpProxyOverride, err := getMCPProxyConfig(doc.Extensions, kongComponents)
	if err != nil {
		return nil, err
	}
	if mcpProxyOverride != nil {
		var override map[string]interface{}
		_ = json.Unmarshal(mcpProxyOverride, &override)
		for k, v := range override {
			if k != "tools" {
				mcpPluginConfig[k] = v
			}
		}
	}

	mcpPlugin := map[string]interface{}{
		"name":   "ai-mcp-proxy",
		"config": mcpPluginConfig,
	}

	if !opts.SkipID {
		mcpPlugin["id"] = uuid.NewSHA1(opts.UUIDNamespace, []byte(mcpRouteName+".plugin.ai-mcp-proxy")).String()
	}
	mcpPlugin["tags"] = docService["tags"]

	mcpRoute["plugins"] = []interface{}{mcpPlugin}

	// Add MCP route to service
	routes = append([]interface{}{mcpRoute}, routes...)
	docService["routes"] = routes

	if upstreams, ok := result["upstreams"].([]interface{}); ok {
		if len(upstreams) == 0 {
			delete(result, "upstreams")
		}
	}

	return result, nil
}
