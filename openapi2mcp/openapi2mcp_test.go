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
