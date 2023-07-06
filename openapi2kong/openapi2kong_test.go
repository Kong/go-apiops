package openapi2kong

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const fixturePath = "./oas3_testfiles/"

func Test_Openapi2kong(t *testing.T) {
	files, err := os.ReadDir(fixturePath)
	if err != nil {
		t.Error("failed reading test data: %w", err)
	}

	for _, file := range files {
		fileNameIn := file.Name()
		if strings.HasSuffix(fileNameIn, ".yaml") {
			fileNameExpected := strings.TrimSuffix(fileNameIn, ".yaml") + ".expected.json"
			fileNameOut := strings.TrimSuffix(fileNameIn, ".yaml") + ".generated.json"
			dataIn, _ := os.ReadFile(fixturePath + fileNameIn)
			dataOut, err := Convert(dataIn, O2kOptions{
				Tags: &[]string{"OAS3_import", "OAS3file_" + fileNameIn},
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
}
