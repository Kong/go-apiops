package openapi2mcp

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const fixturePath = "./oas3_testfiles/"

// findFilesBySuffix returns a list of files in the fixturePath
// that end with the given suffix.
func findFilesBySuffix(t *testing.T, dir string, suffix string) []fs.DirEntry {
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Error("failed reading test data: %w", err)
	}

	// loop over all files, and remove anything that doesn't end with the suffix
	for i := 0; i < len(files); i++ {
		if !strings.HasSuffix(files[i].Name(), suffix) {
			files = slices.Delete(files, i, i+1)
			i--
		}
	}

	return files
}

func Test_Openapi2mcp_InvalidPaths(t *testing.T) {
	dir := filepath.Join(fixturePath, "invalid")
	files := findFilesBySuffix(t, dir, ".yaml")

	// Define expected error messages for different test files
	expectedErrors := map[string]string{
		"no-paths.yaml": "must have `.paths` in the root of the document",
	}

	for _, file := range files {
		fileNameIn := file.Name()
		t.Run(fileNameIn, func(t *testing.T) {
			dataIn, _ := os.ReadFile(filepath.Join(dir, fileNameIn))
			_, err := Convert(dataIn, O2MOptions{
				Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
			})

			if err == nil {
				t.Errorf("'%s' expected error but got none", fileNameIn)
			} else {
				expectedError, exists := expectedErrors[fileNameIn]
				if !exists {
					t.Errorf("No expected error defined for test file: %s", fileNameIn)
				} else {
					assert.Contains(t, err.Error(), expectedError,
						"Error message for '%s' should contain '%s', but got: %s",
						fileNameIn, expectedError, err.Error())
				}
			}
		})
	}
}

// assertFixtureConversions converts each fixture with the options returned by
// optionsFor, writes the .generated.json output, and compares it to .expected.json.
func assertFixtureConversions(t *testing.T, files []string, optionsFor func(fileNameIn string) O2MOptions) {
	t.Helper()

	for _, fileNameIn := range files {
		t.Run(fileNameIn, func(t *testing.T) {
			fileNameExpected := strings.TrimSuffix(fileNameIn, ".yaml") + ".expected.json"
			fileNameOut := strings.TrimSuffix(fileNameIn, ".yaml") + ".generated.json"
			dataIn, err := os.ReadFile(fixturePath + fileNameIn)
			if err != nil {
				t.Fatalf("Failed to read input file: %v", err)
			}

			dataOut, err := Convert(dataIn, optionsFor(fileNameIn))
			if err != nil {
				t.Errorf("'%s' didn't expect error: %v", fixturePath+fileNameIn, err)
				return
			}

			JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
			os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
			JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
			if err != nil {
				t.Fatalf("Failed to read expected file: %v", err)
			}

			assert.JSONEq(t, string(JSONExpected), string(JSONOut),
				"'%s': the JSON blobs should be equal", fixturePath+fileNameIn)
		})
	}
}

func Test_Openapi2mcp_Basic(t *testing.T) {
	// Test basic conversion with default options
	files := []string{
		"01-basic-conversion.yaml",
		"02-mcp-extensions.yaml",
		"07-multiple-servers.yaml",
	}

	assertFixtureConversions(t, files, func(fileNameIn string) O2MOptions {
		return O2MOptions{Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn}}
	})
}

// assertNoSchemaPointers enforces that generated tool schemas are
// self-contained: the ai-mcp-proxy plugin has no OpenAPI document behind it, so
// any $ref, $defs or #/components pointer in the output points at nothing.
func assertNoSchemaPointers(t *testing.T, dataOut map[string]interface{}) {
	t.Helper()

	encoded, err := json.Marshal(dataOut)
	assert.NoError(t, err)

	for _, token := range []string{`"$ref"`, `"$defs"`, "#/components/"} {
		assert.NotContains(t, string(encoded), token,
			"generated config must not contain schema pointers (%s)", token)
	}
}

func Test_Openapi2mcp_SchemaComposition(t *testing.T) {
	// Fixtures covering allOf/anyOf/oneOf and circular references
	files := []string{
		"10-schema-allof.yaml",
		"11-schema-oneof-anyof.yaml",
		"12-schema-circular.yaml",
		"13-schema-params.yaml",
	}

	assertFixtureConversions(t, files, func(string) O2MOptions {
		return O2MOptions{SkipID: true}
	})
}

// Test_Openapi2mcp_NoSchemaPointers converts every fixture and checks that no
// generated schema leaks a $ref, $defs or #/components pointer, regardless of
// the schema shapes the fixture exercises.
func Test_Openapi2mcp_NoSchemaPointers(t *testing.T) {
	files := findFilesBySuffix(t, fixturePath, ".yaml")

	for _, file := range files {
		fileNameIn := file.Name()
		t.Run(fileNameIn, func(t *testing.T) {
			dataIn, err := os.ReadFile(fixturePath + fileNameIn)
			assert.NoError(t, err)

			dataOut, err := Convert(dataIn, O2MOptions{SkipID: true})
			assert.NoError(t, err, "'%s' should convert without error", fileNameIn)

			assertNoSchemaPointers(t, dataOut)
		})
	}
}

func Test_Openapi2mcp_ConversionMode(t *testing.T) {
	// Test with mode=conversion
	fileNameIn := "03-mode-conversion.yaml"
	fileNameExpected := "03-mode-conversion.expected.json"
	fileNameOut := "03-mode-conversion.generated.json"

	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
		Mode: ModeConversion,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
	os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
	JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	assert.JSONEq(t, string(JSONExpected), string(JSONOut),
		"the JSON blobs should be equal for mode=conversion")
}

func Test_Openapi2mcp_DirectRoute(t *testing.T) {
	// Test with IncludeDirectRoute=true
	fileNameIn := "04-direct-route.yaml"
	fileNameExpected := "04-direct-route.expected.json"
	fileNameOut := "04-direct-route.generated.json"

	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags:               []string{"OAS3_import", "OAS3file_" + fileNameIn},
		IncludeDirectRoute: true,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
	os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
	JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	assert.JSONEq(t, string(JSONExpected), string(JSONOut),
		"the JSON blobs should be equal for IncludeDirectRoute=true")
}

func Test_Openapi2mcp_KongExtensions(t *testing.T) {
	// Test with Kong extensions
	fileNameIn := "05-kong-extensions.yaml"
	fileNameExpected := "05-kong-extensions.expected.json"
	fileNameOut := "05-kong-extensions.generated.json"

	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
	os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
	JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	assert.JSONEq(t, string(JSONExpected), string(JSONOut),
		"the JSON blobs should be equal for Kong extensions")
}

func Test_Openapi2mcp_CustomPathPrefix(t *testing.T) {
	// Test with custom path prefix
	fileNameIn := "06-custom-path-prefix.yaml"
	fileNameExpected := "06-custom-path-prefix.expected.json"
	fileNameOut := "06-custom-path-prefix.generated.json"

	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags:       []string{"OAS3_import", "OAS3file_" + fileNameIn},
		PathPrefix: "/custom/mcp/path",
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
	os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
	JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	assert.JSONEq(t, string(JSONExpected), string(JSONOut),
		"the JSON blobs should be equal for custom path prefix")
}

func Test_Openapi2mcp_NoID(t *testing.T) {
	// Test with SkipID=true
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      operationId: list-items
      summary: List items
`)

	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	// Verify no id fields are present
	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	assert.Nil(t, service["id"], "service should not have id when SkipID=true")

	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	assert.Nil(t, route["id"], "route should not have id when SkipID=true")

	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	assert.Nil(t, plugin["id"], "plugin should not have id when SkipID=true")
}

// Test_Openapi2mcp_MultiServerNoDirectRoutes verifies that path-level
// `servers` overrides, which make openapi2kong generate additional services,
// do not leak direct routes into the output when IncludeDirectRoute is false.
// Only the single MCP route on the main service must remain.
func Test_Openapi2mcp_MultiServerNoDirectRoutes(t *testing.T) {
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      operationId: list-items
      summary: List items
  /regions:
    servers:
      - url: https://regions.example.com
    get:
      operationId: list-regions
      summary: List regions
`)

	countRoutes := func(dataOut map[string]interface{}) (total int, servicesWithRoutes int) {
		for _, s := range dataOut["services"].([]interface{}) {
			svc := s.(map[string]interface{})
			rts, ok := svc["routes"].([]interface{})
			if !ok {
				continue
			}
			total += len(rts)
			if len(rts) > 0 {
				servicesWithRoutes++
			}
		}
		return total, servicesWithRoutes
	}

	// Default: only the MCP route on the main service, nothing on the
	// per-server service created for /regions.
	dataOut, err := Convert(dataIn, O2MOptions{SkipID: true})
	assert.NoError(t, err, "should convert without error")

	total, servicesWithRoutes := countRoutes(dataOut)
	assert.Equal(t, 1, total,
		"only the MCP route should exist, direct routes must be stripped from all services")
	assert.Equal(t, 1, servicesWithRoutes,
		"only the main service should carry the MCP route")

	// The single remaining route must be the MCP route.
	svc := dataOut["services"].([]interface{})[0].(map[string]interface{})
	route := svc["routes"].([]interface{})[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	assert.Equal(t, "ai-mcp-proxy",
		plugins[0].(map[string]interface{})["name"],
		"the remaining route must be the MCP route")
	tools := plugins[0].(map[string]interface{})["config"].(map[string]interface{})["tools"]
	assert.Len(t, tools, 2, "both operations must still become MCP tools")

	// With IncludeDirectRoute=true the direct routes are kept, including on
	// the extra per-server service.
	dataOut, err = Convert(dataIn, O2MOptions{SkipID: true, IncludeDirectRoute: true})
	assert.NoError(t, err, "should convert without error with direct routes")

	total, servicesWithRoutes = countRoutes(dataOut)
	assert.Equal(t, 3, total,
		"MCP route plus both direct routes should exist")
	assert.Equal(t, 2, servicesWithRoutes,
		"the main service and the per-server service should both carry routes")
}

func Test_Openapi2mcp_SimplifySchema(t *testing.T) {
	// Test that schemas are simplified properly
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      operationId: list-items
      summary: List items
      parameters:
        - name: date
          in: query
          schema:
            type: string
            format: date
            pattern: "^\\d{4}-\\d{2}-\\d{2}$"
            minLength: 10
            maxLength: 10
`)

	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	// Navigate to the parameter schema
	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})
	tools := config["tools"].([]interface{})
	tool := tools[0].(map[string]interface{})
	params := tool["parameters"].([]map[string]interface{})
	param := params[0]
	schema := param["schema"].(map[string]interface{})

	// Verify schema is simplified - only type should be present
	assert.Equal(t, "string", schema["type"], "schema should have type")
	assert.Nil(t, schema["format"], "schema should not have format (simplified)")
	assert.Nil(t, schema["pattern"], "schema should not have pattern (simplified)")
	assert.Nil(t, schema["minLength"], "schema should not have minLength (simplified)")
	assert.Nil(t, schema["maxLength"], "schema should not have maxLength (simplified)")
}

// schemaOfType returns a simplified schema carrying only a type.
func schemaOfType(name string) map[string]interface{} {
	return withType(name, map[string]interface{}{})
}

// anyOfSchema returns a simplified schema carrying only anyOf branches.
func anyOfSchema(branches ...interface{}) map[string]interface{} {
	return map[string]interface{}{"anyOf": branches}
}

// enumOf returns a simplified schema carrying only an enum.
func enumOf(values ...interface{}) map[string]interface{} {
	return map[string]interface{}{"enum": values}
}

// withType sets the type of a simplified schema and returns it.
func withType(name string, schema map[string]interface{}) map[string]interface{} {
	schema["type"] = name
	return schema
}

func Test_mergeAllOfMember(t *testing.T) {
	objectOf := func(properties map[string]interface{}) map[string]interface{} {
		return withType("object", map[string]interface{}{"properties": properties})
	}
	stringType := schemaOfType("string")
	integerType := schemaOfType("integer")

	tests := []struct {
		name     string
		dst      map[string]interface{}
		src      map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "conflicting types are marked unsatisfiable",
			dst:      schemaOfType("object"),
			src:      schemaOfType("array"),
			expected: withType("object", map[string]interface{}{"not": map[string]interface{}{}}),
		},
		{
			name:     "number and integer intersect to integer",
			dst:      schemaOfType("number"),
			src:      schemaOfType("integer"),
			expected: schemaOfType("integer"),
		},
		{
			name:     "type lists are intersected",
			dst:      map[string]interface{}{"type": []string{"string", "integer"}},
			src:      map[string]interface{}{"type": []string{"number", "boolean"}},
			expected: schemaOfType("integer"),
		},
		{
			name:     "type is taken when the destination has none",
			dst:      map[string]interface{}{},
			src:      schemaOfType("object"),
			expected: schemaOfType("object"),
		},
		{
			name:     "required union deduplicates and keeps first-seen order",
			dst:      map[string]interface{}{"required": []string{"a", "b"}},
			src:      map[string]interface{}{"required": []string{"b", "c"}},
			expected: map[string]interface{}{"required": []string{"a", "b", "c"}},
		},
		{
			name: "properties are unioned",
			dst:  objectOf(map[string]interface{}{"a": stringType}),
			src:  objectOf(map[string]interface{}{"b": integerType}),
			expected: objectOf(map[string]interface{}{
				"a": stringType,
				"b": integerType,
			}),
		},
		{
			name: "overlapping object properties merge recursively",
			dst: objectOf(map[string]interface{}{
				"config": objectOf(map[string]interface{}{"a": stringType}),
			}),
			src: objectOf(map[string]interface{}{
				"config": objectOf(map[string]interface{}{"b": stringType}),
			}),
			expected: objectOf(map[string]interface{}{
				"config": objectOf(map[string]interface{}{
					"a": stringType,
					"b": stringType,
				}),
			}),
		},
		{
			name: "conflicting property types mark the property unsatisfiable",
			dst:  objectOf(map[string]interface{}{"value": schemaOfType("string")}),
			src:  objectOf(map[string]interface{}{"value": schemaOfType("integer")}),
			expected: objectOf(map[string]interface{}{
				"value": withType("string", map[string]interface{}{"not": map[string]interface{}{}}),
			}),
		},
		{
			name:     "enum is taken when the destination has none",
			dst:      schemaOfType("string"),
			src:      withType("string", enumOf("a", "b")),
			expected: withType("string", enumOf("a", "b")),
		},
		{
			name:     "enums are intersected",
			dst:      enumOf("a", "b", "c"),
			src:      enumOf("c", "b", "d"),
			expected: enumOf("b", "c"),
		},
		{
			name: "disjoint enums are marked unsatisfiable, not emptied",
			dst:  enumOf("a", "b"),
			src:  enumOf("c"),
			expected: map[string]interface{}{
				"enum": []interface{}{"a", "b"},
				"not":  map[string]interface{}{},
			},
		},
		{
			name: "object keywords are dropped when the type rules out objects",
			dst:  schemaOfType("string"),
			src: withType("string", map[string]interface{}{
				"properties": map[string]interface{}{"p": stringType},
				"required":   []string{"p"},
			}),
			expected: schemaOfType("string"),
		},
		{
			name:     "items are dropped when the type rules out arrays",
			dst:      schemaOfType("string"),
			src:      map[string]interface{}{"items": stringType},
			expected: schemaOfType("string"),
		},
		{
			name:     "items are taken when the destination has none",
			dst:      schemaOfType("array"),
			src:      map[string]interface{}{"items": stringType},
			expected: withType("array", map[string]interface{}{"items": stringType}),
		},
		{
			name: "items on both sides merge recursively",
			dst: withType("array", map[string]interface{}{
				"items": objectOf(map[string]interface{}{"a": schemaOfType("string")}),
			}),
			src: map[string]interface{}{
				"items": objectOf(map[string]interface{}{"b": schemaOfType("string")}),
			},
			expected: withType("array", map[string]interface{}{
				"items": objectOf(map[string]interface{}{
					"a": stringType,
					"b": stringType,
				}),
			}),
		},
		{
			name:     "anyOf is taken when the destination has none",
			dst:      map[string]interface{}{},
			src:      anyOfSchema(stringType),
			expected: anyOfSchema(stringType),
		},
		{
			name: "a second anyOf is kept as an allOf entry, not concatenated",
			dst:  anyOfSchema(stringType),
			src:  anyOfSchema(integerType),
			expected: map[string]interface{}{
				"anyOf": []interface{}{stringType},
				"allOf": []interface{}{anyOfSchema(integerType)},
			},
		},
		{
			name:     "properties imply an object type",
			dst:      map[string]interface{}{},
			src:      objectOf(map[string]interface{}{"a": stringType}),
			expected: objectOf(map[string]interface{}{"a": stringType}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mergeAllOfMember(tc.dst, tc.src)
			assert.Equal(t, tc.expected, tc.dst)
		})
	}
}

func Test_Openapi2mcp_SecurityACL(t *testing.T) {
	// Test ACL generation from oauth2 security with x-kong-mcp-acl
	fileNameIn := "08-security-acl.yaml"
	fileNameExpected := "08-security-acl.expected.json"
	fileNameOut := "08-security-acl.generated.json"

	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
	os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
	JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	assert.JSONEq(t, string(JSONExpected), string(JSONOut),
		"the JSON blobs should be equal for security ACL")

	// Also verify the ACL structure programmatically
	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	// Verify plugin-level ACL config
	assert.Equal(t, "oauth_access_token", config["acl_attribute_type"],
		"acl_attribute_type should be set")
	assert.Equal(t, "scp", config["access_token_claim_field"],
		"access_token_claim_field should be set")
	assert.NotNil(t, config["default_acl"], "default_acl should be set")

	// Verify per-tool ACL
	tools := config["tools"].([]interface{})
	assert.Len(t, tools, 3, "should have 3 tools (excluded operation filtered)")

	// First tool: get-cool-flights with flights:read
	tool0 := tools[0].(map[string]interface{})
	assert.Equal(t, "get-cool-flights", tool0["name"])
	acl0 := tool0["acl"].(map[string]interface{})
	assert.Equal(t, []string{"flights:read"}, acl0["allow"])

	// Second tool: create-flight with flights:write
	tool1 := tools[1].(map[string]interface{})
	assert.Equal(t, "create-flight", tool1["name"])
	acl1 := tool1["acl"].(map[string]interface{})
	assert.Equal(t, []string{"flights:write"}, acl1["allow"])

	// Third tool: get-flight-by-number with flights:read
	tool2 := tools[2].(map[string]interface{})
	assert.Equal(t, "get-flight-by-number", tool2["name"])
	acl2 := tool2["acl"].(map[string]interface{})
	assert.Equal(t, []string{"flights:read"}, acl2["allow"])
}

func Test_Openapi2mcp_SecurityACL_NoExtension(t *testing.T) {
	// Test that oauth2 security without x-kong-mcp-acl doesn't generate ACL (auto-detect)
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      security:
        - my_oauth:
            - items:read
      operationId: list-items
      summary: List items
components:
  securitySchemes:
    my_oauth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/auth
          tokenUrl: https://example.com/token
          scopes:
            items:read: Read items
`)

	// Without x-kong-mcp-acl on the scheme, ACL generation is not activated (auto-detect).
	// No error should occur, and no ACL fields should be generated.
	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	assert.NoError(t, err, "should not error when oauth2 scheme lacks x-kong-mcp-acl (auto-detect)")

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	// Should have no ACL fields
	assert.Nil(t, config["acl_attribute_type"], "should not have acl_attribute_type")
	assert.Nil(t, config["access_token_claim_field"], "should not have access_token_claim_field")

	// Tool should have no ACL
	tools := config["tools"].([]interface{})
	tool := tools[0].(map[string]interface{})
	assert.Nil(t, tool["acl"], "tool should not have acl")
}

func Test_Openapi2mcp_SecurityACL_UnsupportedScheme(t *testing.T) {
	// Test that non-oauth2 security scheme is silently skipped (no x-kong-mcp-acl to activate ACL)
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      security:
        - api_key: []
      operationId: list-items
      summary: List items
components:
  securitySchemes:
    api_key:
      type: apiKey
      in: header
      name: X-API-Key
`)

	// Without any scheme having x-kong-mcp-acl, ACL is not activated, so no error
	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	assert.NoError(t, err, "should not error for non-oauth2 scheme without ACL activation")

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	tools := config["tools"].([]interface{})
	tool := tools[0].(map[string]interface{})
	assert.Nil(t, tool["acl"], "tool should not have acl for non-oauth2 scheme")
}

func Test_Openapi2mcp_SecurityACL_MixedSchemes(t *testing.T) {
	// Test that when an oauth2 scheme has x-kong-mcp-acl but an operation references
	// a different non-oauth2 scheme, it errors (unless ignore-security-errors is set)
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      security:
        - api_key: []
      operationId: list-items
      summary: List items
  /users:
    get:
      security:
        - my_oauth:
            - users:read
      operationId: list-users
      summary: List users
components:
  securitySchemes:
    api_key:
      type: apiKey
      in: header
      name: X-API-Key
    my_oauth:
      type: oauth2
      x-kong-mcp-acl:
        acl_attribute_type: oauth_access_token
        access_token_claim_field: scp
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/auth
          tokenUrl: https://example.com/token
          scopes:
            users:read: Read users
`)

	// The api_key operation should error because ACL is activated (my_oauth has x-kong-mcp-acl)
	// but api_key is not oauth2
	_, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	assert.Error(t, err, "should error when operation uses non-oauth2 scheme while ACL is active")
	assert.Contains(t, err.Error(), "oauth2")

	// With ignore-security-errors, api_key operation should have no ACL, oauth operation should
	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID:               true,
		IgnoreSecurityErrors: true,
	})
	assert.NoError(t, err, "should not error with ignore-security-errors")

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	tools := config["tools"].([]interface{})
	assert.Len(t, tools, 2)

	// First tool (list-items with api_key) should have no ACL
	tool0 := tools[0].(map[string]interface{})
	assert.Equal(t, "list-items", tool0["name"])
	assert.Nil(t, tool0["acl"], "api_key tool should not have acl")

	// Second tool (list-users with my_oauth) should have ACL
	tool1 := tools[1].(map[string]interface{})
	assert.Equal(t, "list-users", tool1["name"])
	acl1 := tool1["acl"].(map[string]interface{})
	assert.Equal(t, []string{"users:read"}, acl1["allow"])
}

func Test_Openapi2mcp_SecurityACL_NoSecurity(t *testing.T) {
	// Test that specs without any security don't generate ACL
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      operationId: list-items
      summary: List items
`)

	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	assert.NoError(t, err, "should not error without security")

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	assert.Nil(t, config["acl_attribute_type"], "should not have acl_attribute_type")
	assert.Nil(t, config["access_token_claim_field"], "should not have access_token_claim_field")
	assert.Nil(t, config["default_acl"], "should not have default_acl")

	tools := config["tools"].([]interface{})
	tool := tools[0].(map[string]interface{})
	assert.Nil(t, tool["acl"], "tool should not have acl")
}

// When the deprecated mode=conversion alias (normalized to conversion-only) is
// used with x-kong-mcp-acl present in the spec, the output:
//   - MUST NOT contain acl_attribute_type at the plugin level
//     (Kong Gateway rejects this field in conversion-only mode)
//   - MUST NOT contain access_token_claim_field at the plugin level
//     (same reason — these are valid on every listener mode, but not conversion-only)
//   - MUST still contain acl.allow on every tool
//     (the upstream listener needs these scopes to enforce per-tool ACL)
func Test_Openapi2mcp_SecurityACL_ConversionMode(t *testing.T) {
	fileNameIn := "09-security-acl-conversion-mode.yaml"
	fileNameExpected := "09-security-acl-conversion-mode.expected.json"
	fileNameOut := "09-security-acl-conversion-mode.generated.json"

	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
		Mode: ModeConversion,
	})
	assert.NoError(t, err, "should not error for mode=conversion with ACL")

	JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
	os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
	JSONExpected, err := os.ReadFile(fixturePath + fileNameExpected)
	if err != nil {
		t.Fatalf("Failed to read expected file: %v", err)
	}

	assert.JSONEq(t, string(JSONExpected), string(JSONOut),
		"the JSON blobs should be equal for mode=conversion with ACL")

	// --- Programmatic assertions for the fix ---
	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	// Core fix: these two fields MUST be absent in conversion mode.
	// If either is present, Kong Gateway will reject the config with HTTP 400.
	assert.Nil(t, config["acl_attribute_type"],
		"acl_attribute_type must NOT be present in conversion mode — Gateway rejects it")
	assert.Nil(t, config["access_token_claim_field"],
		"access_token_claim_field must NOT be present in conversion mode — Gateway rejects it")

	// Mode must be normalized from the deprecated alias to the real mode name
	assert.Equal(t, ModeConversionOnly, config["mode"])

	// Per-tool acl.allow MUST still be present — the upstream listener uses these scopes
	tools := config["tools"].([]interface{})
	assert.Len(t, tools, 3, "should have 3 tools")

	tool0 := tools[0].(map[string]interface{})
	assert.Equal(t, "get-cool-flights", tool0["name"])
	acl0 := tool0["acl"].(map[string]interface{})
	assert.Equal(t, []string{"flights:read"}, acl0["allow"],
		"tool must still have acl.allow even in conversion mode")

	tool1 := tools[1].(map[string]interface{})
	assert.Equal(t, "create-flight", tool1["name"])
	acl1 := tool1["acl"].(map[string]interface{})
	assert.Equal(t, []string{"flights:write"}, acl1["allow"],
		"tool must still have acl.allow even in conversion mode")

	tool2 := tools[2].(map[string]interface{})
	assert.Equal(t, "get-flight-by-number", tool2["name"])
	acl2 := tool2["acl"].(map[string]interface{})
	assert.Equal(t, []string{"flights:read"}, acl2["allow"],
		"tool must still have acl.allow even in conversion mode")
}

// Test_Openapi2mcp_SecurityACL_ConversionListenerMode confirms that
// conversion-listener mode (the default) still emits acl_attribute_type and
// access_token_claim_field — i.e. the fix is backward-compatible.
func Test_Openapi2mcp_SecurityACL_ConversionListenerMode(t *testing.T) {
	// Re-use the existing 08 fixture which was designed for conversion-listener
	fileNameIn := "08-security-acl.yaml"
	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	// Explicitly pass ModeConversionListener (same as the default)
	dataOut, err := Convert(dataIn, O2MOptions{
		Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
		Mode: ModeConversionListener,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	// These MUST still be present in conversion-listener mode (backward compat)
	assert.Equal(t, "oauth_access_token", config["acl_attribute_type"],
		"acl_attribute_type must still be emitted for conversion-listener mode")
	assert.Equal(t, "scp", config["access_token_claim_field"],
		"access_token_claim_field must still be emitted for conversion-listener mode")

	// Per-tool acl.allow must also still be present
	tools := config["tools"].([]interface{})
	for i, t2 := range tools {
		tool := t2.(map[string]interface{})
		assert.NotNil(t, tool["acl"], "tool[%d] must have acl.allow in conversion-listener mode", i)
	}
}

// Test_Openapi2mcp_SecurityACL_ListenerMode confirms that listener mode emits
// acl_attribute_type and access_token_claim_field, matching the ai-mcp-proxy
// plugin schema (these fields are valid on every listener mode, not just
// conversion-listener).
func Test_Openapi2mcp_SecurityACL_ListenerMode(t *testing.T) {
	fileNameIn := "08-security-acl.yaml"
	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
		Mode: ModeListener,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	assert.Equal(t, "oauth_access_token", config["acl_attribute_type"],
		"acl_attribute_type must be emitted for listener mode")
	assert.Equal(t, "scp", config["access_token_claim_field"],
		"access_token_claim_field must be emitted for listener mode")
}

// Test_Openapi2mcp_SecurityACL_PassthroughListenerMode confirms that
// passthrough-listener mode emits acl_attribute_type and
// access_token_claim_field, matching the ai-mcp-proxy plugin schema.
func Test_Openapi2mcp_SecurityACL_PassthroughListenerMode(t *testing.T) {
	fileNameIn := "08-security-acl.yaml"
	dataIn, err := os.ReadFile(fixturePath + fileNameIn)
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	dataOut, err := Convert(dataIn, O2MOptions{
		Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
		Mode: ModePassthroughListener,
	})
	if err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	assert.Equal(t, "oauth_access_token", config["acl_attribute_type"],
		"acl_attribute_type must be emitted for passthrough-listener mode")
	assert.Equal(t, "scp", config["access_token_claim_field"],
		"access_token_claim_field must be emitted for passthrough-listener mode")
}

func Test_Openapi2mcp_SecurityACL_DocLevelInheritance(t *testing.T) {
	// Test that operations without security inherit from document-level security
	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
servers:
  - url: https://api.example.com
security:
  - my_oauth:
      - items:read
paths:
  /items:
    get:
      operationId: list-items
      summary: List items
  /items/{id}:
    get:
      operationId: get-item
      summary: Get item
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
components:
  securitySchemes:
    my_oauth:
      type: oauth2
      x-kong-mcp-acl:
        acl_attribute_type: oauth_access_token
        access_token_claim_field: scp
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/auth
          tokenUrl: https://example.com/token
          scopes:
            items:read: Read items
`)

	dataOut, err := Convert(dataIn, O2MOptions{
		SkipID: true,
	})
	assert.NoError(t, err, "should not error")

	services := dataOut["services"].([]interface{})
	service := services[0].(map[string]interface{})
	routes := service["routes"].([]interface{})
	route := routes[0].(map[string]interface{})
	plugins := route["plugins"].([]interface{})
	plugin := plugins[0].(map[string]interface{})
	config := plugin["config"].(map[string]interface{})

	// Verify plugin-level ACL config
	assert.Equal(t, "oauth_access_token", config["acl_attribute_type"])
	assert.Equal(t, "scp", config["access_token_claim_field"])

	// Both tools should inherit ACL from document level
	tools := config["tools"].([]interface{})
	assert.Len(t, tools, 2)

	tool0 := tools[0].(map[string]interface{})
	acl0 := tool0["acl"].(map[string]interface{})
	assert.Equal(t, []string{"items:read"}, acl0["allow"],
		"first tool should inherit doc-level security scopes")

	tool1 := tools[1].(map[string]interface{})
	acl1 := tool1["acl"].(map[string]interface{})
	assert.Equal(t, []string{"items:read"}, acl1["allow"],
		"second tool should inherit doc-level security scopes")
}

// convertBodySchema converts a spec whose single operation takes a
// `#/components/schemas/Root` request body, and returns that body's generated
// schema decoded from JSON.
func convertBodySchema(t *testing.T, schemas string) (map[string]interface{}, int) {
	t.Helper()

	dataIn := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /items:
    post:
      operationId: create-item
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Root'
components:
  schemas:
` + schemas)

	dataOut, err := Convert(dataIn, O2MOptions{SkipID: true})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assertNoSchemaPointers(t, dataOut)

	encoded, err := json.Marshal(dataOut)
	assert.NoError(t, err)

	var decoded struct {
		Services []struct {
			Routes []struct {
				Plugins []struct {
					Config struct {
						Tools []struct {
							RequestBody struct {
								Content map[string]struct {
									Schema map[string]interface{} `json:"schema"`
								} `json:"content"`
							} `json:"request_body"`
						} `json:"tools"`
					} `json:"config"`
				} `json:"plugins"`
			} `json:"routes"`
		} `json:"services"`
	}
	assert.NoError(t, json.Unmarshal(encoded, &decoded))

	tool := decoded.Services[0].Routes[0].Plugins[0].Config.Tools[0]
	return tool.RequestBody.Content["application/json"].Schema, len(encoded)
}

func Test_Openapi2mcp_SchemaCompositionEdgeCases(t *testing.T) {
	t.Run("allOf wrapper keeps the referenced enum", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Status:
      type: string
      enum: [active, inactive]
    Root:
      type: object
      properties:
        status:
          allOf:
            - $ref: '#/components/schemas/Status'
`)
		assert.Equal(t, withType("string", map[string]interface{}{
			"enum": []interface{}{"active", "inactive"},
		}), schema["properties"].(map[string]interface{})["status"])
	})

	t.Run("own oneOf does not replace oneOf from allOf", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - oneOf:
            - type: string
            - type: integer
      oneOf:
        - type: boolean
        - type: number
`)
		assert.Equal(t, map[string]interface{}{
			"anyOf": []interface{}{schemaOfType("string"), schemaOfType("integer")},
			"allOf": []interface{}{anyOfSchema(schemaOfType("boolean"), schemaOfType("number"))},
		}, schema)
	})

	t.Run("oneOf lists from two allOf members are not concatenated", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - oneOf:
            - type: string
            - type: integer
        - oneOf:
            - type: boolean
            - type: number
`)
		assert.Len(t, schema["anyOf"], 2)
		assert.Len(t, schema["allOf"], 1)
	})

	t.Run("allOf members' items are merged", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Base:
      type: object
      properties:
        id:
          type: string
    Root:
      allOf:
        - type: array
          items:
            $ref: '#/components/schemas/Base'
        - items:
            properties:
              extra:
                type: integer
`)
		items := schema["items"].(map[string]interface{})
		assert.Equal(t, map[string]interface{}{
			"id":    schemaOfType("string"),
			"extra": schemaOfType("integer"),
		}, items["properties"])
	})

	t.Run("recursive schema defined through allOf keeps its type", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Base:
      type: object
      properties:
        id:
          type: string
    Root:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            children:
              type: array
              items:
                $ref: '#/components/schemas/Root'
`)
		children := schema["properties"].(map[string]interface{})["children"].(map[string]interface{})
		assert.Equal(t, schemaOfType("object"), children["items"])
	})

	t.Run("densely linked schemas are bounded", func(t *testing.T) {
		const count = 10
		var schemas strings.Builder
		for i := 0; i < count; i++ {
			if i == 0 {
				schemas.WriteString("    Root:\n")
			} else {
				schemas.WriteString("    S" + strconv.Itoa(i) + ":\n")
			}
			schemas.WriteString("      type: object\n      properties:\n")
			for j := 1; j < count; j++ {
				if j != i {
					schemas.WriteString("        p" + strconv.Itoa(j) +
						":\n          $ref: '#/components/schemas/S" + strconv.Itoa(j) + "'\n")
				}
			}
		}

		schema, size := convertBodySchema(t, schemas.String())
		assert.Equal(t, "object", schema["type"])
		assert.Less(t, size, 10*1024*1024, "generated config should stay bounded")
		// Past the budget, only the schemas still being expanded add type-only leaves.
		assert.LessOrEqual(t, countSchemaNodes(schema), maxSchemaNodes+count*count,
			"truncated leaves should count toward the budget")
	})

	t.Run("oneOf with overlapping branches is emitted as anyOf", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Pet:
      type: object
      properties:
        petType:
          type: string
      required: [petType]
    Cat:
      allOf:
        - $ref: '#/components/schemas/Pet'
        - properties:
            meow:
              type: string
    Dog:
      allOf:
        - $ref: '#/components/schemas/Pet'
        - properties:
            bark:
              type: string
    Root:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
      discriminator:
        propertyName: petType
`)
		assert.NotContains(t, schema, "oneOf")
		assert.Len(t, schema["anyOf"], 2)
	})

	t.Run("oneOf whose branches differ only in dropped keywords is emitted as anyOf", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      oneOf:
        - type: string
          format: date
        - type: string
          format: date-time
`)
		assert.Equal(t, anyOfSchema(schemaOfType("string"), schemaOfType("string")), schema)
	})

	t.Run("allOf enums intersect numerically equal values", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - type: number
          enum: [1, 2, 3]
        - enum: [1.0, 2.0]
`)
		assert.Equal(t, []interface{}{1.0, 2.0}, schema["enum"])
	})

	t.Run("type list keeps the object shape", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      type: object
      properties:
        config:
          type: ["null", "object"]
          properties:
            a:
              type: string
          required: [a]
`)
		config := schema["properties"].(map[string]interface{})["config"]
		assert.Equal(t, map[string]interface{}{
			"type":       []interface{}{"null", "object"},
			"properties": map[string]interface{}{"a": schemaOfType("string")},
			"required":   []interface{}{"a"},
		}, config)
	})

	t.Run("type list keeps the array shape", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      type: [array, "null"]
      items:
        type: string
`)
		assert.Equal(t, map[string]interface{}{
			"type":  []interface{}{"array", "null"},
			"items": schemaOfType("string"),
		}, schema)
	})

	t.Run("typeless recursive array keeps its type at the cycle", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Arr:
      items:
        $ref: '#/components/schemas/Arr'
    Root:
      type: object
      properties:
        rec:
          $ref: '#/components/schemas/Arr'
`)
		rec := schema["properties"].(map[string]interface{})["rec"]
		assert.Equal(t, withType("array", map[string]interface{}{"items": schemaOfType("array")}), rec)
	})

	t.Run("type implied by allOf member properties, also at the cycle point", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      required: [a]
      allOf:
        - properties:
            next:
              $ref: '#/components/schemas/Root'
`)
		next := schema["properties"].(map[string]interface{})["next"].(map[string]interface{})
		assert.Equal(t, "object", schema["type"], "properties from allOf imply an object")
		assert.Equal(t, schemaOfType("object"), next, "the truncated cycle point infers the same type")
	})

	t.Run("mutually including typeless allOf members do not hang", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    A:
      allOf:
        - $ref: '#/components/schemas/B'
        - $ref: '#/components/schemas/C'
    B:
      allOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/C'
    C:
      allOf:
        - $ref: '#/components/schemas/A'
        - $ref: '#/components/schemas/B'
    Root:
      type: object
      properties:
        a:
          $ref: '#/components/schemas/A'
`)
		assert.Equal(t, map[string]interface{}{}, schema["properties"].(map[string]interface{})["a"])
	})

	t.Run("type list keeps all its types, null included", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      type: [string, integer, "null"]
      enum: [1, "a"]
`)
		assert.Equal(t, map[string]interface{}{
			"type": []interface{}{"string", "integer", "null"},
			"enum": []interface{}{1.0, "a"},
		}, schema)
	})

	t.Run("allOf object members do not add properties to a non-object schema", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      type: string
      allOf:
        - type: [object, string]
          properties:
            p:
              type: string
          required: [p]
`)
		assert.Equal(t, schemaOfType("string"), schema)
	})

	t.Run("allOf members with no common type accept nothing", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - type: string
        - type: integer
`)
		assert.Equal(t, withType("string", map[string]interface{}{"not": map[string]interface{}{}}), schema)
	})

	t.Run("allOf members with disjoint enums accept nothing", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - type: string
          enum: [a, b]
        - enum: [c]
`)
		assert.Equal(t, withType("string", map[string]interface{}{
			"enum": []interface{}{"a", "b"},
			"not":  map[string]interface{}{},
		}), schema)
	})

	t.Run("implied object type does not conflict with a declared type", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - properties:
            a:
              type: string
        - type: string
`)
		assert.Equal(t, schemaOfType("string"), schema)
	})

	t.Run("null in both allOf members is not a conflict", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      allOf:
        - type: "null"
        - type: [string, "null"]
`)
		assert.Equal(t, schemaOfType("null"), schema)
	})

	t.Run("unquoted date enum values keep their text", func(t *testing.T) {
		schema, _ := convertBodySchema(t, `
    Root:
      type: string
      enum: [2024-01-01, "2024-01-01T00:00:00Z"]
`)
		assert.Equal(t, []interface{}{"2024-01-01", "2024-01-01T00:00:00Z"}, schema["enum"])
	})
}

// countSchemaNodes counts the schemas in a simplified schema, itself included.
func countSchemaNodes(schema map[string]interface{}) int {
	count := 1
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for _, property := range properties {
			count += countSchemaNodes(property.(map[string]interface{}))
		}
	}
	if items, ok := schema["items"].(map[string]interface{}); ok {
		count += countSchemaNodes(items)
	}
	for _, keyword := range []string{"allOf", "anyOf"} {
		if branches, ok := schema[keyword].([]interface{}); ok {
			for _, branch := range branches {
				count += countSchemaNodes(branch.(map[string]interface{}))
			}
		}
	}
	return count
}
