package openapi2mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-slugify"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	openapibase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const (
	formatVersionKey   = "_format_version"
	formatVersionValue = "3.0"

	// MCP proxy modes
	ModeConversionListener = "conversion-listener"
	ModeConversion         = "conversion"

	// Default schemes
	httpScheme  = "http"
	httpsScheme = "https"
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

// Slugify converts a name to a valid Kong name by removing and replacing unallowed characters
// and sanitizing non-latin characters. Multiple inputs will be concatenated using '_'.
func Slugify(name ...string) string {
	slugifier := (&slugify.Slugifier{}).ToLower(true).InvalidChar("-").WordSeparator("-")
	concatBy := "_"

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

// getKongTags returns the provided tags or if nil, then the `x-kong-tags` property,
// validated to be a string array.
func getKongTags(doc v3.Document, tagsProvided []string) ([]string, error) {
	if tagsProvided != nil {
		return tagsProvided, nil
	}

	if doc.Extensions == nil {
		return make([]string, 0), nil
	}

	kongTags, ok := doc.Extensions.Get("x-kong-tags")
	if !ok {
		return make([]string, 0), nil
	}

	resultArray := make([]string, len(kongTags.Content))
	for i, v := range kongTags.Content {
		var tagsValue interface{}
		err := yaml.Unmarshal([]byte(v.Value), &tagsValue)
		if err != nil {
			return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings: %w", err)
		}

		switch tag := tagsValue.(type) {
		case string:
			resultArray[i] = tag
		default:
			return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings")
		}
	}

	return resultArray, nil
}

// getKongName returns the `x-kong-name` property, validated to be a string
func getKongName(extensions *orderedmap.Map[string, *yaml.Node]) (string, error) {
	if extensions == nil {
		return "", nil
	}

	xKongName, ok := extensions.Get("x-kong-name")
	if !ok {
		return "", nil
	}

	var name string
	err := yaml.Unmarshal([]byte(xKongName.Value), &name)
	if err != nil {
		return "", fmt.Errorf("expected 'x-kong-name' to be a string: %w", err)
	}

	return name, nil
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

// getServiceDefaults returns a JSON string containing the service defaults
func getServiceDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-service-defaults", components)
}

// getRouteDefaults returns a JSON string containing the route defaults
func getRouteDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-route-defaults", components)
}

// getUpstreamDefaults returns a JSON string containing the upstream defaults
func getUpstreamDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-upstream-defaults", components)
}

// getMCPProxyConfig returns the x-kong-mcp-proxy override config
func getMCPProxyConfig(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-mcp-proxy", components)
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

// getPluginsList returns a list of plugins from x-kong-plugin-* extensions
func getPluginsList(
	extensions *orderedmap.Map[string, *yaml.Node],
	uuidNamespace uuid.UUID,
	baseName string,
	components *map[string]interface{},
	tags []string,
	skipID bool,
) ([]*map[string]interface{}, error) {
	plugins := make(map[string]*map[string]interface{})

	if extensions == nil {
		return make([]*map[string]interface{}, 0), nil
	}

	extension := extensions.First()
	for extension != nil {
		extensionName := extension.Key()
		if strings.HasPrefix(extensionName, "x-kong-plugin-") {
			pluginName := strings.TrimPrefix(extensionName, "x-kong-plugin-")

			jsonstr, err := getXKongObject(extensions, extensionName, components)
			if err != nil {
				return nil, err
			}

			var pluginConfig map[string]interface{}
			err = json.Unmarshal(jsonstr, &pluginConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON object for '%s': %w", extensionName, err)
			}

			pluginConfig["name"] = pluginName
			if !skipID {
				pluginConfig["id"] = uuid.NewSHA1(uuidNamespace, []byte(baseName+".plugin."+pluginName)).String()
			}
			pluginConfig["tags"] = tags

			delete(pluginConfig, "service")
			delete(pluginConfig, "route")

			plugins[pluginName] = &pluginConfig
		}

		extension = extension.Next()
	}

	// Sort plugins by name for deterministic output
	sortedNames := make([]string, 0, len(plugins))
	for pluginName := range plugins {
		sortedNames = append(sortedNames, pluginName)
	}
	sort.Strings(sortedNames)

	sorted := make([]*map[string]interface{}, len(plugins))
	for i, pluginName := range sortedNames {
		sorted[i] = plugins[pluginName]
	}
	return sorted, nil
}

// parseServerUris parses the server URIs after rendering template variables
func parseServerUris(servers []*v3.Server) ([]*url.URL, error) {
	var targets []*url.URL

	if len(servers) == 0 {
		uriObject, _ := url.ParseRequestURI("/")
		targets = make([]*url.URL, 1)
		targets[0] = uriObject
	} else {
		targets = make([]*url.URL, len(servers))

		for i, server := range servers {
			uriString := server.URL

			pair := server.Variables.First()
			for pair != nil {
				name := pair.Key()
				svar := pair.Value()
				uriString = strings.ReplaceAll(uriString, "{"+name+"}", svar.Default)
				pair = pair.Next()
			}

			uriObject, err := url.ParseRequestURI(uriString)
			if err != nil {
				return targets, fmt.Errorf("failed to parse uri '%s'; %w", uriString, err)
			}

			if uriObject.Path == "" {
				uriObject.Path = "/"
			}

			targets[i] = uriObject
		}
	}

	return targets, nil
}

// setServerDefaults sets the scheme and port if missing
func setServerDefaults(targets []*url.URL, schemeDefault string) {
	for _, target := range targets {
		if target.Host == "" {
			target.Host = "localhost"
		}

		if target.Scheme == "" {
			switch target.Port() {
			case "80":
				target.Scheme = httpScheme
			case "443":
				target.Scheme = httpsScheme
			default:
				target.Scheme = schemeDefault
			}
		}

		if target.Host != "" && target.Port() == "" {
			if target.Scheme == httpScheme {
				target.Host = target.Host + ":80"
			}
			if target.Scheme == httpsScheme {
				target.Host = target.Host + ":443"
			}
		}
	}
}

// createKongUpstream creates a new upstream entity
func createKongUpstream(
	baseName string,
	servers []*v3.Server,
	upstreamDefaults []byte,
	tags []string,
	uuidNamespace uuid.UUID,
	skipID bool,
) (map[string]interface{}, error) {
	var upstream map[string]interface{}

	if upstreamDefaults != nil {
		_ = json.Unmarshal(upstreamDefaults, &upstream)
	} else {
		upstream = make(map[string]interface{})
	}

	upstreamName := baseName + ".upstream"
	if !skipID {
		upstream["id"] = uuid.NewSHA1(uuidNamespace, []byte(upstreamName)).String()
	}
	upstream["name"] = upstreamName
	upstream["tags"] = tags

	targets, err := parseServerUris(servers)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upstream: %w", err)
	}

	setServerDefaults(targets, httpsScheme)

	upstreamTargets := make([]map[string]interface{}, len(targets))
	for i, target := range targets {
		t := make(map[string]interface{})
		t["target"] = target.Host
		t["tags"] = tags
		upstreamTargets[i] = t
	}
	upstream["targets"] = upstreamTargets

	return upstream, nil
}

// CreateKongService creates a new Kong service entity, and optional upstream
func CreateKongService(
	baseName string,
	servers []*v3.Server,
	serviceDefaults []byte,
	upstreamDefaults []byte,
	tags []string,
	uuidNamespace uuid.UUID,
	skipID bool,
) (map[string]interface{}, map[string]interface{}, error) {
	var (
		service  map[string]interface{}
		upstream map[string]interface{}
	)

	if serviceDefaults != nil {
		_ = json.Unmarshal(serviceDefaults, &service)
	} else {
		service = make(map[string]interface{})
	}

	if !skipID {
		service["id"] = uuid.NewSHA1(uuidNamespace, []byte(baseName+".service")).String()
	}
	service["name"] = baseName
	service["tags"] = tags
	service["plugins"] = make([]interface{}, 0)
	service["routes"] = make([]interface{}, 0)

	targets, err := parseServerUris(servers)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create service: %w", err)
	}

	scheme := httpsScheme
	if service["protocol"] != nil {
		scheme = service["protocol"].(string)
	}
	setServerDefaults(targets, scheme)

	if service["protocol"] == nil {
		scheme = targets[0].Scheme
		service["protocol"] = scheme
	}
	if service["path"] == nil {
		service["path"] = targets[0].Path
	}
	if service["port"] == nil {
		if targets[0].Port() != "" {
			parsedPort, err := strconv.ParseUint(targets[0].Port(), 10, 16)
			if err != nil {
				return nil, nil, err
			}
			service["port"] = parsedPort
		} else {
			if scheme != httpScheme {
				service["port"] = 443
			} else {
				service["port"] = 80
			}
		}
	}

	if service["host"] == nil {
		if len(targets) == 1 && upstreamDefaults == nil {
			service["host"] = targets[0].Hostname()
		} else {
			upstream, err = createKongUpstream(baseName, servers, upstreamDefaults, tags, uuidNamespace, skipID)
			if err != nil {
				return nil, nil, err
			}
			service["host"] = upstream["name"]
		}
	}

	return service, upstream, nil
}

// simplifySchema simplifies an OpenAPI schema to essential properties
func simplifySchema(schema *openapibase.Schema) map[string]interface{} {
	if schema == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Handle type
	if schema.Type != nil && len(schema.Type) > 0 {
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

	// Set up output document
	result := make(map[string]interface{})
	result[formatVersionKey] = formatVersionValue
	services := make([]interface{}, 0)
	upstreams := make([]interface{}, 0)

	var (
		err            error
		doc            v3.Document
		kongComponents *map[string]interface{}
		kongTags       []string

		docBaseName         string
		docServers          []*v3.Server
		docServiceDefaults  []byte
		docUpstreamDefaults []byte
		docRouteDefaults    []byte
		docService          map[string]interface{}
		docUpstream         map[string]interface{}
	)

	// Load and parse the OAS file
	openapiDoc, err := libopenapi.NewDocument(content)
	if err != nil {
		return nil, fmt.Errorf("error parsing OAS3 file: [%w]", err)
	}

	// Configure document options
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.IgnoreArrayCircularReferences = true
	docConfig.IgnorePolymorphicCircularReferences = true
	openapiDoc.SetConfiguration(docConfig)

	v3Model, errs := openapiDoc.BuildV3Model()
	if errs != nil {
		logbasics.Error(errs, "error while building v3 document model")
		return nil, fmt.Errorf("cannot create v3 model from document: %w", errs)
	}

	if v3Model != nil {
		doc = v3Model.Model
	}

	// Collect tags
	if kongTags, err = getKongTags(doc, opts.Tags); err != nil {
		return nil, err
	}

	// Set document level elements
	docServers = doc.Servers

	// Determine document name
	docBaseName = opts.DocName
	if docBaseName == "" {
		if docBaseName, err = getKongName(doc.Extensions); err != nil {
			return nil, err
		}
		if docBaseName == "" {
			if doc.Info != nil && doc.Info.Title != "" {
				docBaseName = doc.Info.Title
			} else {
				id, err := uuid.NewRandom()
				if err != nil {
					return nil, fmt.Errorf("failed to generate UUID: %w", err)
				}
				docBaseName = id.String()
			}
		}
	}
	docBaseName = Slugify(docBaseName)

	if kongComponents, err = getXKongComponents(doc); err != nil {
		return nil, err
	}

	// Get defaults
	if docServiceDefaults, err = getServiceDefaults(doc.Extensions, kongComponents); err != nil {
		return nil, err
	}
	if docUpstreamDefaults, err = getUpstreamDefaults(doc.Extensions, kongComponents); err != nil {
		return nil, err
	}
	if docRouteDefaults, err = getRouteDefaults(doc.Extensions, kongComponents); err != nil {
		return nil, err
	}

	// Create the Kong service and optional upstream
	docService, docUpstream, err = CreateKongService(
		docBaseName,
		docServers,
		docServiceDefaults,
		docUpstreamDefaults,
		kongTags,
		opts.UUIDNamespace,
		opts.SkipID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service/upstream: %w", err)
	}

	services = append(services, docService)
	if docUpstream != nil {
		upstreams = append(upstreams, docUpstream)
	}

	// Get document-level plugins
	docPlugins, err := getPluginsList(doc.Extensions, opts.UUIDNamespace, docBaseName, kongComponents, kongTags, opts.SkipID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugins list: %w", err)
	}
	docService["plugins"] = docPlugins

	// Check for paths
	if doc.Paths == nil {
		return nil, fmt.Errorf("must have `.paths` in the root of the document")
	}

	// Build MCP tools from all operations
	tools := make([]interface{}, 0)

	// Create sorted array of paths for deterministic output
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

			// Check if operation is excluded
			excluded, err := getExtensionBool(operation.Extensions, "x-kong-mcp-exclude")
			if err != nil {
				return nil, err
			}
			if excluded {
				continue
			}

			tool, err := buildMCPTool(pathKey, methodKey, operation, pathItem.Parameters)
			if err != nil {
				return nil, fmt.Errorf("failed to build MCP tool for %s %s: %w", methodKey, pathKey, err)
			}

			tools = append(tools, tool)
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
	mcpRoute["tags"] = kongTags

	// Build ai-mcp-proxy plugin config
	mcpPluginConfig := map[string]interface{}{
		"mode":  opts.Mode,
		"tools": tools,
	}

	// Check for x-kong-mcp-proxy override
	mcpProxyOverride, err := getMCPProxyConfig(doc.Extensions, kongComponents)
	if err != nil {
		return nil, err
	}
	if mcpProxyOverride != nil {
		var override map[string]interface{}
		_ = json.Unmarshal(mcpProxyOverride, &override)
		// Merge override into config (override takes precedence except for tools)
		for k, v := range override {
			if k != "tools" { // Don't override generated tools
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
	mcpPlugin["tags"] = kongTags

	mcpRoute["plugins"] = []interface{}{mcpPlugin}

	// Add MCP route to service
	routes := docService["routes"].([]interface{})
	routes = append(routes, mcpRoute)

	// Optionally add direct routes (non-MCP)
	if opts.IncludeDirectRoute {
		directRoutes, err := buildDirectRoutes(doc, docBaseName, docRouteDefaults, kongTags, opts.UUIDNamespace, opts.SkipID)
		if err != nil {
			return nil, err
		}
		routes = append(routes, directRoutes...)
	}

	docService["routes"] = routes

	// Build result
	result["services"] = services
	if len(upstreams) > 0 {
		result["upstreams"] = upstreams
	}

	return result, nil
}

// buildDirectRoutes builds traditional Kong routes for direct API access (non-MCP)
func buildDirectRoutes(
	doc v3.Document,
	baseName string,
	routeDefaults []byte,
	tags []string,
	uuidNamespace uuid.UUID,
	skipID bool,
) ([]interface{}, error) {
	routes := make([]interface{}, 0)

	if doc.Paths == nil {
		return routes, nil
	}

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

			// Build route name
			routeName := baseName
			if operation.OperationId != "" {
				routeName = baseName + "_" + Slugify(operation.OperationId)
			} else {
				routeName = baseName + "_" + Slugify(pathKey) + "_" + strings.ToLower(methodKey)
			}

			route := make(map[string]interface{})
			if routeDefaults != nil {
				_ = json.Unmarshal(routeDefaults, &route)
				delete(route, "service")
			}

			if !skipID {
				route["id"] = uuid.NewSHA1(uuidNamespace, []byte(routeName+".route")).String()
			}

			// Convert path parameters to regex
			convertedPath := pathKey
			charsToEscape := []string{"(", ")", ".", "+", "?", "*", "[", "$"}
			for _, char := range charsToEscape {
				convertedPath = strings.ReplaceAll(convertedPath, char, "\\"+char)
			}

			re := regexp.MustCompile(`{([^}]+)}`)
			regexPriority := 200
			if matches := re.FindAllStringSubmatch(convertedPath, -1); matches != nil {
				regexPriority = 100
				for _, match := range matches {
					varName := match[1]
					captureName := strings.ReplaceAll(strings.ToLower(varName), "-", "_")
					regexMatch := "(?<" + captureName + ">[^#?/]+)"
					placeHolder := "{" + varName + "}"
					convertedPath = strings.Replace(convertedPath, placeHolder, regexMatch, 1)
				}
			}

			route["name"] = routeName
			route["paths"] = []string{"~" + convertedPath + "$"}
			route["methods"] = []string{strings.ToUpper(methodKey)}
			route["tags"] = tags
			route["regex_priority"] = regexPriority
			route["strip_path"] = false
			route["plugins"] = make([]interface{}, 0)

			routes = append(routes, route)
		}
	}

	return routes, nil
}
