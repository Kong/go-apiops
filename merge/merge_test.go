package merge

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kong/go-apiops/filebasics"
	"github.com/stretchr/testify/assert"
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
