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
	testDataString := `
openapi: 3.0.3
info:
  title: Path parameter test
  version: v1
servers:
  - url: "https://example.com"

paths:
  /demo/{something-very-long-that-is-way-beyond-the-32-limit}/:
    get:
      operationId: opsid
      parameters:
        - in: path
          name: something-very-long-that-is-way-beyond-the-32-limit
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`
	_, err := Convert([]byte(testDataString), O2kOptions{})
	if err == nil {
		t.Error("Expected error, but got none")
	} else {
		assert.Contains(t, err.Error(), "path-parameter name exceeds 32 characters")
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
