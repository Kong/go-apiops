package merge

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kong/go-apiops/filebasics"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func Test_Merge_Files_Good(t *testing.T) {
	testSet := []struct {
		files            []string
		expectedFilename string
	}{
		{[]string{
			"./merge_testfiles/file1.yml",
			"./merge_testfiles/file2.yml",
			"./merge_testfiles/file3.yml",
		}, "test1_expected.json"},
		{[]string{
			"./merge_testfiles/file3.yml",
			"./merge_testfiles/file2.yml",
			"./merge_testfiles/file1.yml",
		}, "test2_expected.json"},
	}

	for _, tdata := range testSet {
		res, err := Files(tdata.files)
		if err != nil {
			t.Error(fmt.Sprintf("'%s' didn't expect error: %%w", tdata.files), err)
		} else {
			expected := filebasics.MustReadFile("./merge_testfiles/" + tdata.expectedFilename)
			result := filebasics.MustSerialize(res, false)

			filebasics.MustWriteSerializedFile("./merge_testfiles/"+
				strings.Replace(tdata.expectedFilename, "_expected.", "_generated.", -1), res, false)
			assert.JSONEq(t, string(*expected), string(*result),
				"'%s': the JSON blobs should be equal", tdata.files)
		}
	}
}

func Test_Merge_Files_Bad(t *testing.T) {
	testSet := []struct {
		files       []string
		errorString string
	}{
		{[]string{
			"./merge_testfiles/file1.yml",
			"./merge_testfiles/badversion.yml",
		}, "failed to merge ./merge_testfiles/badversion.yml: files are incompatible; " +
			"major versions are incompatible; 3.0 and 1.0"},
		{[]string{
			"./merge_testfiles/file1.yml",
			"./merge_testfiles/transform_false.yml",
		}, "failed to merge ./merge_testfiles/transform_false.yml: files are incompatible; " +
			"files with '_transform: true' (default) and '_transform: false' are not compatible"},
	}

	for _, tdata := range testSet {
		_, err := Files(tdata.files)
		assert.EqualError(t, err, tdata.errorString)
	}
}

func Test_Merge_Alpha_Ordering(t *testing.T) {
	// Define the test data.
	testDataYAML := `plugins:
- name: acl
  config:
    whitelist:
    - example.com
    blacklist:
    - example.org
services:
- name: example-service
  url: http://example.com
  routes:
  - name: example-route
    regex_priority: 0
    hosts:
    - example.com
    methods:
    - GET
    strip_path: true
    preserve_host: true
    path_handling: v1
    protocols:
    - http
    - https`

	// Write the test data to a temporary file.
	tmpfile, err := os.CreateTemp("", "test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.WriteString(testDataYAML)
	if err != nil {
		t.Fatal(err)
	}

	// Call the function under test.
	result, err := Files([]string{tmpfile.Name()})
	if err != nil {
		t.Fatal(err)
	}

	// Marshal the actual and expected results to YAML strings.
	actualYAML, err := yaml.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	expectedResultYAML := `plugins:
- config:
    blacklist:
    - example.org
    whitelist:
    - example.com
  name: acl
services:
- name: example-service
  routes:
  - hosts:
    - example.com
    methods:
    - GET
    name: example-route
    path_handling: v1
    preserve_host: true
    protocols:
    - http
    - https
    regex_priority: 0
    strip_path: true
  url: http://example.com`
	expectedYAML := []byte(expectedResultYAML)

	sExpected := strings.TrimRight(string(expectedYAML), "\n")
	sActual := strings.TrimRight(string(actualYAML), "\n")

	// Compare the actual and expected YAML strings using go-cmp.
	if diff := cmp.Diff(sExpected, sActual); diff != "" {
		t.Fatalf("Files() mismatch (-want +got):\n%s", diff)
	}
}

func Test_Merge_Multiple_Simple_Files(t *testing.T) {
	// Define the test data for the first file.
	file1YAML := `plugins:
- name: acl
  config:
    whitelist:
    - example.com`

	// Write the test data to a temporary file.
	file1, err := os.CreateTemp("", "test1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file1.Name())
	_, err = file1.WriteString(file1YAML)
	if err != nil {
		t.Fatal(err)
	}

	// Define the test data for the second file.
	file2YAML := `plugins:
- name: key-auth
  config:
    anonymous:
    - example.org`
	// Write the test data to a temporary file.
	file2, err := os.CreateTemp("", "test2.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file2.Name())
	_, err = file2.WriteString(file2YAML)
	if err != nil {
		t.Fatal(err)
	}

	// Call the function under test with both files.
	result, err := Files([]string{file1.Name(), file2.Name()})
	if err != nil {
		t.Fatal(err)
	}
	// Marshal the actual and expected results to YAML strings.
	actualYAML, err := yaml.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	// Define the expected result.
	expectedResultYAML := `plugins:
- config:
    whitelist:
    - example.com
  name: acl
- config:
    anonymous:
    - example.org
  name: key-auth`
	expectedYAML := []byte(expectedResultYAML)

	sExpected := strings.TrimRight(string(expectedYAML), "\n")
	sActual := strings.TrimRight(string(actualYAML), "\n")

	// Compare the actual and expected YAML strings using go-cmp.
	if diff := cmp.Diff(sExpected, sActual); diff != "" {
		t.Fatalf("Files() mismatch (-want +got):\n%s", diff)
	}
}
