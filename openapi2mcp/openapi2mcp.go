package openapi2mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/openapi2kong"
	"github.com/kong/go-apiops/openapitools"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	openapibase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const (
	// MCP proxy modes, matching the ai-mcp-proxy plugin's PLUGIN_MODES (kong-ee).
	ModePassthroughListener = "passthrough-listener"
	ModeConversionListener  = "conversion-listener"
	ModeConversionOnly      = "conversion-only"
	ModeListener            = "listener"

	// ModeConversion is a deprecated alias for ModeConversionOnly. It was never
	// a real ai-mcp-proxy mode - callers should use ModeConversionOnly instead.
	ModeConversion = "conversion"
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
	// MCP proxy mode: one of "passthrough-listener", "conversion-listener",
	// "conversion-only", or "listener". "conversion" is accepted as a
	// deprecated alias for "conversion-only".
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
	if opts.Mode == ModeConversion {
		opts.Mode = ModeConversionOnly
	}
}

// getMCPProxyConfig returns the x-kong-mcp-proxy override config
func getMCPProxyConfig(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return openapitools.GetXKongObject(extensions, "x-kong-mcp-proxy", components)
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

	nodeBytes, err := openapitools.ConvertYamlNodeToBytes(node)
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

	nodeBytes, err := openapitools.ConvertYamlNodeToBytes(node)
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

// maxSchemaNodes bounds the schema nodes emitted per parameter/request body, to prevent
// exponential expansion through dense references.
const maxSchemaNodes = 10000

// schemaSimplifier converts OpenAPI schema to self-contained JSON Schema: inlines all $refs
// since ai-mcp-proxy has no OpenAPI doc. Refs keyed by unique text (no file/remote refs).
type schemaSimplifier struct {
	inProgress map[string]bool // refs being expanded, to cut cycles
	nodes      int             // schema nodes emitted so far, truncated ones included
}

// inlineSchema returns the self-contained schema of a parameter or request body.
// logContext holds key/value pairs naming the schema in the truncation log.
func inlineSchema(proxy *openapibase.SchemaProxy, logContext ...interface{}) map[string]interface{} {
	s := &schemaSimplifier{inProgress: make(map[string]bool)}
	result := s.simplify(proxy)
	if s.nodes > maxSchemaNodes {
		logbasics.Info("schema is too large to inline fully, truncated",
			append(logContext, "maxSchemaNodes", maxSchemaNodes)...)
	}
	fillImpliedTypes(result)
	return result
}

// simplify simplifies one schema keeping type, properties, required, items, enum, allOf
// (merged), anyOf/oneOf (as anyOf branches). Types declared only; fillImpliedTypes infers rest.
func (s *schemaSimplifier) simplify(proxy *openapibase.SchemaProxy) map[string]interface{} {
	if proxy == nil {
		return map[string]interface{}{}
	}

	schema := proxy.Schema()
	if schema == nil {
		return map[string]interface{}{}
	}
	s.nodes++

	if proxy.IsReference() {
		ref := proxy.GetReference()
		if s.inProgress[ref] {
			// A recursive type cannot be represented without a reference.
			logbasics.Debug("schema reference cycle detected, truncating", "ref", ref)
			return truncatedSchema(schema)
		}
		s.inProgress[ref] = true
		defer delete(s.inProgress, ref)
	}

	if s.nodes > maxSchemaNodes {
		return truncatedSchema(schema)
	}

	result := map[string]interface{}{}
	setTypes(result, schema.Type)

	if allowsType(schema.Type, "object") {
		if schema.Properties != nil {
			props := make(map[string]interface{})
			for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
				props[pair.Key()] = s.simplify(pair.Value())
			}
			result["properties"] = props
		}
		if len(schema.Required) > 0 {
			result["required"] = schema.Required
		}
	}

	if allowsType(schema.Type, "array") && schema.Items != nil && schema.Items.A != nil {
		result["items"] = s.simplify(schema.Items.A)
	}

	if len(schema.Enum) > 0 {
		result["enum"] = decodeEnum(schema.Enum)
	}

	for _, member := range schema.AllOf {
		mergeAllOfMember(result, s.simplify(member))
	}

	// oneOf is emitted as anyOf. Simplification drops the keywords that keep oneOf
	// branches apart (const, format, pattern, discriminator, ...), so the simplified
	// branches can overlap, and a valid value matching more than one would fail oneOf.
	if len(schema.OneOf) > 0 {
		addAnyOf(result, s.simplifyAll(schema.OneOf))
	}
	if len(schema.AnyOf) > 0 {
		addAnyOf(result, s.simplifyAll(schema.AnyOf))
	}

	return result
}

// simplifyAll simplifies a list of schemas.
func (s *schemaSimplifier) simplifyAll(proxies []*openapibase.SchemaProxy) []interface{} {
	result := make([]interface{}, 0, len(proxies))
	for _, proxy := range proxies {
		result = append(result, s.simplify(proxy))
	}
	return result
}

// truncatedSchema returns a schema with type only (for cycles/size limits). Type: declared
// types intersected, else object if properties, else array if items (includes allOf members).
func truncatedSchema(schema *openapibase.Schema) map[string]interface{} {
	members := []*openapibase.Schema{schema}
	for _, member := range schema.AllOf {
		if memberSchema := member.Schema(); memberSchema != nil {
			members = append(members, memberSchema)
		}
	}

	var types []string
	hasProperties, hasItems := false, false
	for _, member := range members {
		if len(types) == 0 {
			types = member.Type
		} else if common := intersectTypes(types, member.Type); len(common) > 0 {
			types = common
		}
		hasProperties = hasProperties || member.Properties != nil
		hasItems = hasItems || (member.Items != nil && member.Items.A != nil)
	}

	result := map[string]interface{}{}
	setTypes(result, types)
	if len(types) == 0 {
		switch {
		case hasProperties:
			result["type"] = "object"
		case hasItems:
			result["type"] = "array"
		}
	}
	return result
}

// fillImpliedTypes sets type object for properties or array for items when no type declared.
// Runs post-merge, so implied types never conflict (e.g., allOf with both properties and {type: string}).
func fillImpliedTypes(schema map[string]interface{}) {
	if _, hasType := schema["type"]; !hasType {
		if _, hasProps := schema["properties"]; hasProps {
			schema["type"] = "object"
		} else if _, hasItems := schema["items"]; hasItems {
			schema["type"] = "array"
		}
	}

	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for _, prop := range props {
			if propSchema, ok := prop.(map[string]interface{}); ok {
				fillImpliedTypes(propSchema)
			}
		}
	}
	if items, ok := schema["items"].(map[string]interface{}); ok {
		fillImpliedTypes(items)
	}
	for _, keyword := range []string{"allOf", "anyOf"} {
		branches, _ := schema[keyword].([]interface{})
		for _, branch := range branches {
			if branchSchema, ok := branch.(map[string]interface{}); ok {
				fillImpliedTypes(branchSchema)
			}
		}
	}
}

// typesOf returns the types of a simplified schema, nil when its type is unconstrained.
func typesOf(schema map[string]interface{}) []string {
	switch schemaType := schema["type"].(type) {
	case string:
		return []string{schemaType}
	case []string:
		return schemaType
	}
	return nil
}

// setTypes sets the type of a simplified schema: a string for one type, a list for several.
// "null" is kept, so OAS 3.1 nullable types such as ["string", "null"] still accept null.
func setTypes(schema map[string]interface{}, types []string) {
	switch len(types) {
	case 0:
		delete(schema, "type")
	case 1:
		schema["type"] = types[0]
	default:
		schema["type"] = types
	}
}

// allowsType reports whether types admit the given type; no types admit any.
func allowsType(types []string, schemaType string) bool {
	return len(types) == 0 || slices.Contains(types, schemaType)
}

// intersectTypes returns the types allowed by both lists, in first-list order.
// integer is a subset of number, so number and integer intersect to integer.
func intersectTypes(first, second []string) []string {
	result := make([]string, 0, len(first))
	for _, schemaType := range first {
		common := ""
		switch {
		case slices.Contains(second, schemaType):
			common = schemaType
		case schemaType == "number" && slices.Contains(second, "integer"),
			schemaType == "integer" && slices.Contains(second, "number"):
			common = "integer"
		}
		if common != "" && !slices.Contains(result, common) {
			result = append(result, common)
		}
	}
	return result
}

// markUnsatisfiable records allOf member contradictions via "not": {} so no values are accepted.
func markUnsatisfiable(schema map[string]interface{}, reason string, keysAndValues ...interface{}) {
	logbasics.Debug("allOf members contradict each other, no value satisfies the schema: "+reason,
		keysAndValues...)
	schema["not"] = map[string]interface{}{}
}

// mergeAllOfMember merges src into dst per allOf: types/enums intersected (contradictions unsatisfiable),
// properties/required unioned, nested schemas merged recursively. Drops keywords ruled out by merged type.
func mergeAllOfMember(dst, src map[string]interface{}) {
	if srcTypes := typesOf(src); len(srcTypes) > 0 {
		if dstTypes := typesOf(dst); len(dstTypes) == 0 {
			setTypes(dst, srcTypes)
		} else if common := intersectTypes(dstTypes, srcTypes); len(common) > 0 {
			setTypes(dst, common)
		} else {
			markUnsatisfiable(dst, "conflicting types", "types", dstTypes, "conflicting", srcTypes)
		}
	}

	if srcProps, ok := src["properties"].(map[string]interface{}); ok {
		dstProps, ok := dst["properties"].(map[string]interface{})
		if !ok {
			dstProps = make(map[string]interface{})
			dst["properties"] = dstProps
		}
		for key, srcProp := range srcProps {
			if dstProp, exists := dstProps[key]; exists {
				mergeSubschema(dstProp, srcProp)
			} else {
				dstProps[key] = srcProp
			}
		}
	}

	if srcRequired, ok := src["required"].([]string); ok {
		dstRequired, _ := dst["required"].([]string)
		dst["required"] = unionRequired(dstRequired, srcRequired)
	}

	if srcItems, ok := src["items"]; ok {
		if dstItems, hasItems := dst["items"]; hasItems {
			mergeSubschema(dstItems, srcItems)
		} else {
			dst["items"] = srcItems
		}
	}

	if srcEnum, ok := src["enum"].([]interface{}); ok {
		if dstEnum, hasEnum := dst["enum"].([]interface{}); hasEnum {
			// An empty enum is invalid in draft 4, so the enum is kept and marked instead.
			if common := intersectEnum(dstEnum, srcEnum); len(common) > 0 {
				dst["enum"] = common
			} else {
				markUnsatisfiable(dst, "enums with no common value", "enum", dstEnum, "conflicting", srcEnum)
			}
		} else {
			dst["enum"] = srcEnum
		}
	}

	if srcAnyOf, ok := src["anyOf"].([]interface{}); ok {
		addAnyOf(dst, srcAnyOf)
	}

	if srcAllOf, ok := src["allOf"].([]interface{}); ok {
		dstAllOf, _ := dst["allOf"].([]interface{})
		dst["allOf"] = append(dstAllOf, srcAllOf...)
	}

	if srcNot, ok := src["not"]; ok {
		dst["not"] = srcNot
	}

	types := typesOf(dst)
	if !allowsType(types, "object") {
		delete(dst, "properties")
		delete(dst, "required")
	}
	if !allowsType(types, "array") {
		delete(dst, "items")
	}
}

// mergeSubschema recursively merges src into dst if both are schema objects.
func mergeSubschema(dst, src interface{}) {
	dstObj, dstIsObj := dst.(map[string]interface{})
	srcObj, srcIsObj := src.(map[string]interface{})
	if dstIsObj && srcIsObj {
		mergeAllOfMember(dstObj, srcObj)
	}
}

// addAnyOf adds anyOf branches. A second anyOf is kept as an allOf entry to preserve
// the (A|B) AND (C|D) constraint instead of loosening to (A|B|C|D).
func addAnyOf(dst map[string]interface{}, branches []interface{}) {
	if _, exists := dst["anyOf"]; !exists {
		dst["anyOf"] = branches
		return
	}
	allOf, _ := dst["allOf"].([]interface{})
	dst["allOf"] = append(allOf, map[string]interface{}{"anyOf": branches})
}

// unionRequired merges required lists, keeping order and removing duplicates.
func unionRequired(first, second []string) []string {
	merged := make([]string, 0, len(first)+len(second))
	for _, list := range [][]string{first, second} {
		for _, name := range list {
			if !slices.Contains(merged, name) {
				merged = append(merged, name)
			}
		}
	}
	return merged
}

// intersectEnum filters to values in both sets by JSON encoding, handling YAML type variance (1 vs 1.0).
func intersectEnum(first, second []interface{}) []interface{} {
	secondJSON := make(map[string]bool, len(second))
	for _, value := range second {
		encoded, _ := json.Marshal(value)
		secondJSON[string(encoded)] = true
	}

	result := make([]interface{}, 0, len(first))
	for _, value := range first {
		if encoded, _ := json.Marshal(value); secondJSON[string(encoded)] {
			result = append(result, value)
		}
	}
	return result
}

// decodeEnum converts YAML enum nodes to plain Go values for marshaling. Timestamps keep
// their text: decoding would turn an unquoted 2024-01-01 into 2024-01-01T00:00:00Z.
func decodeEnum(nodes []*yaml.Node) []interface{} {
	values := make([]interface{}, 0, len(nodes))
	for _, node := range nodes {
		if node.ShortTag() == "!!timestamp" {
			values = append(values, node.Value)
			continue
		}
		var value interface{}
		if err := node.Decode(&value); err != nil {
			continue // unreachable: openapi2kong.Convert already decoded the document
		}
		values = append(values, value)
	}
	return values
}

// buildParameters builds the parameters array for an MCP tool
func buildParameters(toolName string, params []*v3.Parameter) []map[string]interface{} {
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
			p["schema"] = inlineSchema(param.Schema, "tool", toolName, "parameter", param.Name)
		}

		result = append(result, p)
	}

	return result
}

// buildRequestBody builds the request_body object for an MCP tool
func buildRequestBody(toolName string, rb *v3.RequestBody) map[string]interface{} {
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
				mediaContent["schema"] = inlineSchema(mediaTypeObj.Schema, "tool", toolName, "requestBody", mediaType)
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
		toolName = openapitools.ToKebabCase(operation.OperationId)
	}
	if toolName == "" {
		// Fallback: generate from method + path
		toolName = openapitools.ToKebabCase(strings.ToLower(method) + "-" + strings.ReplaceAll(path, "/", "-"))
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
		tool["parameters"] = buildParameters(toolName, allParams)
	}

	// Add request body
	if operation.RequestBody != nil {
		tool["request_body"] = buildRequestBody(toolName, operation.RequestBody)
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
	kongComponents, err := openapitools.GetXKongComponents(doc)
	if err != nil {
		return nil, err
	}
	docRouteDefaults, err := openapitools.GetRouteDefaults(doc.Extensions, kongComponents)
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

	// acl_attribute_type and access_token_claim_field are valid on every listener
	// mode (conversion-listener, listener, passthrough-listener); in conversion-only
	// mode a separate listener plugin is responsible for token validation and
	// these fields must be omitted there.
	if aclConfig != nil && opts.Mode != ModeConversionOnly {
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
