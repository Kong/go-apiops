package openapi2mcp

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

func Test_Openapi2mcp_Basic(t *testing.T) {
	// Test basic conversion with default options
	files := []string{
		"01-basic-conversion.yaml",
		"02-mcp-extensions.yaml",
		"07-multiple-servers.yaml",
	}

	for _, fileNameIn := range files {
		t.Run(fileNameIn, func(t *testing.T) {
			fileNameExpected := strings.TrimSuffix(fileNameIn, ".yaml") + ".expected.json"
			fileNameOut := strings.TrimSuffix(fileNameIn, ".yaml") + ".generated.json"
			dataIn, err := os.ReadFile(fixturePath + fileNameIn)
			if err != nil {
				t.Fatalf("Failed to read input file: %v", err)
			}

			dataOut, err := Convert(dataIn, O2MOptions{
				Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
			})
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

func Test_Openapi2mcp_ToolNameNormalization(t *testing.T) {
	// Test tool name kebab-case normalization
	testCases := []struct {
		input    string
		expected string
	}{
		{"getFlights", "get-flights"},
		{"get_flights", "get-flights"},
		{"GetFlights", "get-flights"},
		{"get-flights", "get-flights"},
		{"listAllUsers", "list-all-users"},
		{"CreateNewItem", "create-new-item"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := toKebabCase(tc.input)
			assert.Equal(t, tc.expected, result, fmt.Sprintf("toKebabCase(%s)", tc.input))
		})
	}
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
