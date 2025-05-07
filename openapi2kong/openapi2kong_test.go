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

	for _, file := range files {
		fileNameIn := file.Name()
		dataIn, _ := os.ReadFile(filepath.Join(dir, fileNameIn))
		_, err := Convert(dataIn, O2kOptions{
			Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
			OIDC: true,
		})
		if err == nil {
			t.Error(fmt.Sprintf("'%s' expected error: %%w", dir+fileNameIn), err)
		} else {
			assert.Contains(t, err.Error(), "must have `.paths` in the root of the document")
		}
	}
}

func Test_Openapi2kong(t *testing.T) {
	files := findFilesBySuffix(t, fixturePath, ".yaml")

	for _, file := range files {
		fileNameIn := file.Name()
		fileNameExpected := strings.TrimSuffix(fileNameIn, ".yaml") + ".expected.json"
		fileNameOut := strings.TrimSuffix(fileNameIn, ".yaml") + ".generated.json"
		dataIn, _ := os.ReadFile(fixturePath + fileNameIn)
		dataOut, err := Convert(dataIn, O2kOptions{
			Tags: []string{"OAS3_import", "OAS3file_" + fileNameIn},
			OIDC: true,
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
