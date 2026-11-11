package openapi2kong

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

func Test_Openapi2kong_InvalidPaths(t *testing.T) {
	dir := filepath.Join(fixturePath, "invalid")
	files := findFilesBySuffix(t, dir, ".yaml")

	// Define expected error messages for different test files
	expectedErrors := map[string]string{
		"no-paths.yaml":                       "must have `.paths` in the root of the document",
		"multiple-security-requirements.yaml": "only a single security-requirement is supported",
		"multiple-security-schemes.yaml":      "within a security-requirement only a single security-scheme is supported",
		"unsupported-security-type.yaml":      "only security-schemes of type 'openIdConnect' are supported",
		"missing-security-scheme.yaml":        "no security-schemes with name 'nonExistentScheme' found in components",
		"route-path-invalid-prefix.yaml": "path '/path1': invalid 'x-kong-route-path' value 'path1/?$'; " +
			"a route path must start with '/' (plain path) or '~/' (regex path)",
		"route-path-invalid-regex-prefix.yaml": "operation 'GET /path1': invalid 'x-kong-route-path' value '~^/path1/?$'",
		"route-path-not-a-string.yaml": "path '/path1': expected 'x-kong-route-path' to be a string " +
			"or a non-empty array of strings",
		"route-path-empty-array.yaml":      "got an empty array",
		"route-path-array-non-string.yaml": "got a non-string array entry",
	}

	for _, file := range files {
		fileNameIn := file.Name()
		t.Run(fileNameIn, func(t *testing.T) {
			dataIn, _ := os.ReadFile(filepath.Join(dir, fileNameIn))
			_, err := Convert(dataIn, O2kOptions{
				Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
				OIDC: true,
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

func Test_Openapi2kong(t *testing.T) {
	files := findFilesBySuffix(t, fixturePath, ".yaml")

	for _, file := range files {
		fileNameIn := file.Name()
		fileNameBase := strings.TrimSuffix(fileNameIn, ".yaml")
		t.Run(fileNameBase, func(t *testing.T) {
			fileNameExpected := fileNameBase + ".expected.json"
			fileNameOut := fileNameBase + ".generated.json"
			dataIn, _ := os.ReadFile(fixturePath + fileNameIn)

			// Test option configuration
			allHeadersRequired := false
			reuseServices := false

			var config map[string]any
			yaml.Unmarshal(dataIn, &config)
			if testConfig, ok := config["x-test-config"].(map[string]any); ok {
				if val, ok := testConfig["treatAllHeadersAsRequired"]; ok {
					allHeadersRequired = val.(bool)
				}
				if val, ok := testConfig["reuseServices"]; ok {
					reuseServices = val.(bool)
				}
			}

			dataOut, err := Convert(dataIn, O2kOptions{
				Tags:                      []string{"OAS3_import", "OAS3file_" + fileNameIn},
				OIDC:                      true,
				TreatAllHeadersAsRequired: allHeadersRequired,
				ReuseServices:             reuseServices,
			})
			if err != nil {
				t.Error(fmt.Sprintf("'%s' didn't expect error: %%w", fixturePath+fileNameIn), err)
			} else {
				JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
				os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
				JSONExpected, _ := os.ReadFile(fixturePath + fileNameExpected)
				assert.JSONEq(t, string(JSONExpected), string(JSONOut),
					"'%s': the JSON blobs should be equal", fixturePath+fileNameIn)
			}
		})
	}
}

func Test_Openapi2kong_InsoCompat(t *testing.T) {
	suffix := ".expected_inso.json"
	files := findFilesBySuffix(t, fixturePath, suffix)

	for _, file := range files {
		fileName := strings.TrimSuffix(file.Name(), suffix)

		fileNameIn := fileName + ".yaml"
		fileNameExpected := fileName + ".expected_inso.json"
		fileNameOut := fileName + ".generated_inso.json"

		dataIn, _ := os.ReadFile(fixturePath + fileNameIn)
		dataOut, err := Convert(dataIn, O2kOptions{
			Tags:       []string{"OAS3_import", "OAS3file_" + fileNameIn},
			InsoCompat: true,
			SkipID:     true,
		})

		if err != nil {
			t.Error(fmt.Sprintf("'%s' didn't expect error: %%w", fixturePath+fileNameIn), err)
		} else {
			JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
			os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
			JSONExpected, _ := os.ReadFile(fixturePath + fileNameExpected)
			assert.JSONEq(t, string(JSONExpected), string(JSONOut),
				"'%s': the JSON blobs should be equal", fixturePath+fileNameIn)
		}
	}
}

func Test_Openapi2kong_IgnoreCircularRefs(t *testing.T) {
	suffix := ".expected_no_circular.json"
	files := findFilesBySuffix(t, fixturePath, suffix)

	for _, file := range files {
		fileName := strings.TrimSuffix(file.Name(), suffix)

		fileNameIn := fileName + ".circular-yaml"
		fileNameExpected := fileName + ".expected_no_circular.json"
		fileNameOut := fileName + ".generated_no_circular.json"
		// log.Printf("input file: '%v', expected file: '%v'", fileNameIn, fileNameExpected)

		dataIn, _ := os.ReadFile(fixturePath + fileNameIn)
		dataOut, err := Convert(dataIn, O2kOptions{
			Tags:               []string{"OAS3_import", "OAS3file_" + fileNameIn},
			IgnoreCircularRefs: true,
		})

		if err != nil {
			t.Error(fmt.Sprintf("'%s' didn't expect error: %%w", fixturePath+fileNameIn), err)
		} else {
			JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
			os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
			JSONExpected, _ := os.ReadFile(fixturePath + fileNameExpected)
			assert.JSONEq(t, string(JSONExpected), string(JSONOut),
				"'%s': the JSON blobs should be equal", fixturePath+fileNameIn)
		}
	}
}

func Test_Openapi2kong_pathParamLength(t *testing.T) {
	// 129 characters, exceeding the 128-character limit
	longParamName := "something-very-long-that-is-way-beyond-the-previous-32-character-" +
		"limit-and-now-valid-with-the-updated-128-character-limit-123456"
	testDataString := fmt.Sprintf(`
openapi: 3.0.3
info:
  title: Path parameter test
  version: v1
servers:
  - url: "https://example.com"

paths:
  /demo/{%s}/:
    get:
      operationId: opsid
      parameters:
        - in: path
          name: %s
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`, longParamName, longParamName)
	_, err := Convert([]byte(testDataString), O2kOptions{})
	if err == nil {
		t.Error("Expected error, but got none")
	} else {
		assert.Contains(t, err.Error(), "path-parameter name exceeds 128 characters")
	}
}

func Test_Openapi2kong_SkipRouteByHeader(t *testing.T) {
	suffix := ".expected_skip_header.json"
	files := findFilesBySuffix(t, fixturePath, suffix)

	for _, file := range files {
		fileName := strings.TrimSuffix(file.Name(), suffix)

		fileNameIn := fileName + ".yaml"
		fileNameExpected := fileName + ".expected_skip_header.json"
		fileNameOut := fileName + ".generated_skip_header.json"

		dataIn, _ := os.ReadFile(fixturePath + fileNameIn)
		dataOut, err := Convert(dataIn, O2kOptions{
			Tags:              []string{"OAS3_import", "OAS3file_" + fileNameIn},
			SkipRouteByHeader: true,
		})

		if err != nil {
			t.Error(fmt.Sprintf("'%s' didn't expect error: %%w", fixturePath+fileNameIn), err)
		} else {
			JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
			os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
			JSONExpected, _ := os.ReadFile(fixturePath + fileNameExpected)
			assert.JSONEq(t, string(JSONExpected), string(JSONOut),
				"'%s': the JSON blobs should be equal", fixturePath+fileNameIn)
		}
	}
}

// Test_Openapi2kong_IgnoreSecurityErrors verifies that --ignore-security-errors
// does not suppress OIDC plugin generation for valid single openIdConnect schemes.
// Regression test for https://github.com/Kong/deck/issues/1829
func Test_Openapi2kong_IgnoreSecurityErrors(t *testing.T) {
	suffix := ".expected.json"
	testFiles := []string{
		"38-ignore-security-errors-oidc",
	}

	for _, fileNameBase := range testFiles {
		t.Run(fileNameBase, func(t *testing.T) {
			fileNameIn := fileNameBase + ".yaml"
			fileNameExpected := fileNameBase + suffix
			fileNameOut := fileNameBase + ".generated.json"

			dataIn, _ := os.ReadFile(fixturePath + fileNameIn)
			dataOut, err := Convert(dataIn, O2kOptions{
				Tags:                 []string{"OAS3_import", "OAS3file_" + fileNameIn},
				OIDC:                 true,
				IgnoreSecurityErrors: true,
			})

			if err != nil {
				t.Errorf("'%s' didn't expect error: %v", fixturePath+fileNameIn, err)
			} else {
				JSONOut, _ := json.MarshalIndent(dataOut, "", "  ")
				os.WriteFile(fixturePath+fileNameOut, JSONOut, 0o600)
				JSONExpected, _ := os.ReadFile(fixturePath + fileNameExpected)
				assert.JSONEq(t, string(JSONExpected), string(JSONOut),
					"'%s': the JSON blobs should be equal", fixturePath+fileNameIn)
			}
		})
	}
}

// Test_Openapi2kong_IgnoreSecurityErrors_SkipsInvalid verifies that --ignore-security-errors
// suppresses errors for unsupported security configurations (multiple requirements, multiple schemes,
// non-OIDC types) but still processes valid ones in the same document.
func Test_Openapi2kong_IgnoreSecurityErrors_SkipsInvalid(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		wantOIDC bool
	}{
		{
			name: "multiple requirements at doc level are ignored",
			spec: `
openapi: "3.0.0"
info:
  title: test
  version: v1
servers:
  - url: https://example.com
security:
  - oidc1: []
  - oidc2: []
paths:
  /test:
    get:
      operationId: getTest
      responses:
        "200":
          description: OK
components:
  securitySchemes:
    oidc1:
      type: openIdConnect
      openIdConnectUrl: https://example.com/.well-known/openid-configuration
    oidc2:
      type: openIdConnect
      openIdConnectUrl: https://example.com/.well-known/openid-configuration
`,
			wantOIDC: false,
		},
		{
			name: "non-OIDC type at operation level is ignored, no plugin generated",
			spec: `
openapi: "3.0.0"
info:
  title: test
  version: v1
servers:
  - url: https://example.com
paths:
  /test:
    get:
      operationId: getTest
      security:
        - apiKey: []
      responses:
        "200":
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
`,
			wantOIDC: false,
		},
		{
			name: "valid single OIDC at operation level generates plugin",
			spec: `
openapi: "3.0.0"
info:
  title: test
  version: v1
servers:
  - url: https://example.com
paths:
  /test:
    get:
      operationId: getTest
      security:
        - oidc: []
      responses:
        "200":
          description: OK
components:
  securitySchemes:
    oidc:
      type: openIdConnect
      openIdConnectUrl: https://example.com/.well-known/openid-configuration
`,
			wantOIDC: true,
		},
		{
			name: "mixed supported OIDC and unsupported apiKey in one spec",
			spec: `
openapi: "3.0.4"
info:
  title: mixed-security
  version: "1.0.0"
servers:
  - url: https://example.com
components:
  securitySchemes:
    OpenIDConnect:
      type: openIdConnect
      openIdConnectUrl: https://auth.example.com/.well-known/openid-configuration
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
paths:
  /secure-endpoint:
    get:
      operationId: secureEndpoint
      security:
        - OpenIDConnect: []
      responses:
        "204":
          description: No content
  /legacy-endpoint:
    get:
      operationId: legacyEndpoint
      security:
        - ApiKeyAuth: []
      responses:
        "204":
          description: No content
`,
			wantOIDC: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Convert([]byte(tc.spec), O2kOptions{
				OIDC:                 true,
				IgnoreSecurityErrors: true,
			})
			assert.NoError(t, err)

			jsonData, _ := json.Marshal(result)
			hasOIDC := strings.Contains(string(jsonData), `"name":"openid-connect"`)
			if tc.wantOIDC {
				assert.True(t, hasOIDC, "expected openid-connect plugin to be generated")
			} else {
				assert.False(t, hasOIDC, "expected no openid-connect plugin to be generated")
			}
		})
	}
}

func Test_getRoutePaths(t *testing.T) {
	tests := []struct {
		name        string
		yamlDoc     string // a yaml mapping holding the extensions
		expected    []string
		expectedErr string
	}{
		{
			name:     "not present",
			yamlDoc:  `x-kong-name: something`,
			expected: nil,
		},
		{
			name:     "single plain path",
			yamlDoc:  `x-kong-route-path: /catalog/products/v1`,
			expected: []string{"/catalog/products/v1"},
		},
		{
			name:     "single regex path",
			yamlDoc:  `x-kong-route-path: ~/catalog/products/v1/?$`,
			expected: []string{"~/catalog/products/v1/?$"},
		},
		{
			name:     "quoted regex path with special characters",
			yamlDoc:  `x-kong-route-path: "~/items/(?<id>[0-9]+)/(a|b)*$"`,
			expected: []string{"~/items/(?<id>[0-9]+)/(a|b)*$"},
		},
		{
			name: "array of mixed paths",
			yamlDoc: `x-kong-route-path:
  - /plain
  - ~/regex/?$`,
			expected: []string{"/plain", "~/regex/?$"},
		},
		{
			name:     "flow style array",
			yamlDoc:  `x-kong-route-path: ["/a", "~/b$"]`,
			expected: []string{"/a", "~/b$"},
		},
		{
			name:        "missing leading slash",
			yamlDoc:     `x-kong-route-path: catalog/products/v1`,
			expectedErr: "a route path must start with '/' (plain path) or '~/' (regex path)",
		},
		{
			name:        "regex without slash after tilde",
			yamlDoc:     `x-kong-route-path: ~^/catalog$`,
			expectedErr: "invalid 'x-kong-route-path' value '~^/catalog$'",
		},
		{
			name:        "empty string",
			yamlDoc:     `x-kong-route-path: ""`,
			expectedErr: "invalid 'x-kong-route-path' value ''",
		},
		{
			name:        "number",
			yamlDoc:     `x-kong-route-path: 123`,
			expectedErr: "expected 'x-kong-route-path' to be a string or a non-empty array of strings, got '123'",
		},
		{
			name:        "boolean",
			yamlDoc:     `x-kong-route-path: true`,
			expectedErr: "expected 'x-kong-route-path' to be a string or a non-empty array of strings",
		},
		{
			name:        "null",
			yamlDoc:     `x-kong-route-path: null`,
			expectedErr: "expected 'x-kong-route-path' to be a string or a non-empty array of strings",
		},
		{
			name:        "object",
			yamlDoc:     "x-kong-route-path:\n  path: /a",
			expectedErr: "expected 'x-kong-route-path' to be a string or a non-empty array of strings",
		},
		{
			name:        "empty array",
			yamlDoc:     `x-kong-route-path: []`,
			expectedErr: "got an empty array",
		},
		{
			name:        "array with non-string entry",
			yamlDoc:     `x-kong-route-path: ["/a", 1]`,
			expectedErr: "got a non-string array entry",
		},
		{
			name:        "array with nested array",
			yamlDoc:     `x-kong-route-path: ["/a", ["/b"]]`,
			expectedErr: "got a non-string array entry",
		},
		{
			name:        "array with an invalid path",
			yamlDoc:     `x-kong-route-path: ["/a", "b"]`,
			expectedErr: "invalid 'x-kong-route-path' value 'b'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var root yaml.Node
			if err := yaml.Unmarshal([]byte(tc.yamlDoc), &root); err != nil {
				t.Fatalf("invalid test yaml: %v", err)
			}
			mapping := root.Content[0]
			extensions := orderedmap.New[string, *yaml.Node]()
			for i := 0; i < len(mapping.Content); i += 2 {
				extensions.Set(mapping.Content[i].Value, mapping.Content[i+1])
			}

			paths, err := getRoutePaths(extensions)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				assert.Nil(t, paths)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, paths)
		})
	}

	t.Run("nil extensions", func(t *testing.T) {
		paths, err := getRoutePaths(nil)
		assert.NoError(t, err)
		assert.Nil(t, paths)
	})
}

// Test_Openapi2kong_RoutePath verifies the generated route paths for x-kong-route-path,
// independent of the golden files, and checks that the other route properties
// (names, methods, regex_priority) are generated as usual.
func Test_Openapi2kong_RoutePath(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: catalog
  version: v1
servers:
  - url: https://example.com
paths:
  /catalog/products/v1:
    x-kong-route-path: ~/catalog/products/v1/?$
    get:
      operationId: listProducts
      responses:
        "200":
          description: OK
    post:
      operationId: createProduct
      x-kong-route-path:
        - /catalog/products/v1/messages
        - ~/catalog/products/v1/msg/?$
      responses:
        "200":
          description: OK
  /items/{id}:
    x-kong-route-path: ~/items/(?<id>\d+)$
    get:
      responses:
        "200":
          description: OK
  /untouched/{id}:
    get:
      responses:
        "200":
          description: OK
`
	testCases := []struct {
		opts              O2kOptions
		listProductsName  string
		createProductName string
	}{
		{O2kOptions{}, "catalog_listproducts", "catalog_createproduct"},
		{O2kOptions{InsoCompat: true, SkipID: true}, "catalog-listProducts", "catalog-createProduct"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("inso=%v", tc.opts.InsoCompat), func(t *testing.T) {
			result, err := Convert([]byte(spec), tc.opts)
			if !assert.NoError(t, err) {
				return
			}

			// collect routes by their method + first path for easy lookup
			type routeInfo struct {
				name          string
				paths         []string
				regexPriority any
			}
			routes := map[string]routeInfo{}
			for _, svc := range result["services"].([]interface{}) {
				for _, r := range svc.(map[string]interface{})["routes"].([]interface{}) {
					route := r.(map[string]interface{})
					method := route["methods"].([]string)[0]
					paths := route["paths"].([]string)
					routes[method+" "+paths[0]] = routeInfo{
						name:          route["name"].(string),
						paths:         paths,
						regexPriority: route["regex_priority"],
					}
				}
			}

			assert.Len(t, routes, 4)

			listProducts := routes["GET ~/catalog/products/v1/?$"]
			assert.Equal(t, []string{"~/catalog/products/v1/?$"}, listProducts.paths)
			assert.Equal(t, tc.listProductsName, listProducts.name)

			createProduct := routes["POST /catalog/products/v1/messages"]
			assert.Equal(t, []string{"/catalog/products/v1/messages", "~/catalog/products/v1/msg/?$"},
				createProduct.paths, "operation level must override path level")
			assert.Equal(t, tc.createProductName, createProduct.name)

			item := routes[`GET ~/items/(?<id>\d+)$`]
			assert.Equal(t, []string{`~/items/(?<id>\d+)$`}, item.paths)
			assert.Equal(t, regexPriorityWithPathParams, item.regexPriority,
				"regex_priority is still derived from the OAS path")

			untouched := routes["GET ~/untouched/(?<id>[^#?/]+)$"]
			assert.Equal(t, []string{"~/untouched/(?<id>[^#?/]+)$"}, untouched.paths,
				"paths without the extension must be generated as before")
		})
	}
}

// Test_Openapi2kong_RoutePathNotShared verifies that routes sharing the same
// x-kong-route-path value get independent copies of the paths array.
func Test_Openapi2kong_RoutePathNotShared(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: shared
  version: v1
servers:
  - url: https://example.com
paths:
  /a:
    x-kong-route-path: ["/a", "/b"]
    get:
      responses:
        "200":
          description: OK
    post:
      responses:
        "200":
          description: OK
`
	result, err := Convert([]byte(spec), O2kOptions{})
	if !assert.NoError(t, err) {
		return
	}
	routes := result["services"].([]interface{})[0].(map[string]interface{})["routes"].([]interface{})
	assert.Len(t, routes, 2)
	paths0 := routes[0].(map[string]interface{})["paths"].([]string)
	paths1 := routes[1].(map[string]interface{})["paths"].([]string)
	paths0[0] = "/modified"
	assert.Equal(t, []string{"/a", "/b"}, paths1)
}
