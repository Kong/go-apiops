package merge_test

import (
	"fmt"
	"strings"

	. "github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/merge"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge", func() {
	validateMerge := func(filenames []string, expected string, expectError bool, expectedHistory []interface{}) {
		GinkgoHelper()
		res, hist, err := merge.Files(filenames)
		if err != nil {
			if expectError {
				// 'expected' is an error string to match
				Expect(err).To(MatchError(expected))
			} else {
				// 'expected' is filename of expected json output, so an error wasn't expected
				Fail(fmt.Sprintf("'%s' didn't expect error: %s", filenames, err))
			}
		} else {
			// 'expected' is filename of expected json output
			if expectError {
				// 'expected' is an error string to match, but the merge succeeded...
				Fail(fmt.Sprintf("'%s' expected error: %s", filenames, expected))
			} else {
				// 'expected' is filename of expected json output, so an error wasn't expected
				expectedResult := MustReadFile("./merge_testfiles/" + expected)
				result := MustSerialize(res, OutputFormatJSON)

				MustWriteSerializedFile("./merge_testfiles/"+
					strings.Replace(expected, "_expected.", "_generated.", -1), res, OutputFormatJSON)

				Expect(*result).To(MatchJSON(*expectedResult))
			}
		}

		Expect(hist).To(BeEquivalentTo(expectedHistory))
	}

	Describe("merges files", func() {
		It("attempt 1: original order", func() {
			// This tests the order of the resulting file, but also the version of the
			// final file
			fileList := []string{
				"./merge_testfiles/file1.yml",
				"./merge_testfiles/file2.yml",
				"./merge_testfiles/file3.yml",
			}
			expected := "test1_expected.json"
			expectErr := false
			expectedHist := []interface{}{
				map[string]interface{}{
					"filename": "./merge_testfiles/file1.yml",
				},
				map[string]interface{}{
					"filename": "./merge_testfiles/file2.yml",
				},
				map[string]interface{}{
					"filename": "./merge_testfiles/file3.yml",
				},
			}

			validateMerge(fileList, expected, expectErr, expectedHist)
		})

		It("attempt 2: same files, different order", func() {
			// This tests the order of the resulting file (different from attempt 1),
			// but also the version of the final file (same as attempt 1)
			fileList := []string{
				"./merge_testfiles/file3.yml",
				"./merge_testfiles/file2.yml",
				"./merge_testfiles/file1.yml",
			}
			expected := "test2_expected.json"
			expectErr := false
			expectedHist := []interface{}{
				map[string]interface{}{
					"filename": "./merge_testfiles/file3.yml",
				},
				map[string]interface{}{
					"filename": "./merge_testfiles/file2.yml",
				},
				map[string]interface{}{
					"filename": "./merge_testfiles/file1.yml",
				},
			}

			validateMerge(fileList, expected, expectErr, expectedHist)
		})

		It("with incompatible versions errors", func() {
			fileList := []string{
				"./merge_testfiles/file1.yml",
				"./merge_testfiles/badversion.yml",
			}
			expected := "failed to merge ./merge_testfiles/badversion.yml: files are incompatible; " +
				"major versions are incompatible; 3.0 and 1.0"
			expectErr := true

			validateMerge(fileList, expected, expectErr, nil)
		})

		It("with incompatible '_transform' errors", func() {
			fileList := []string{
				"./merge_testfiles/file1.yml",
				"./merge_testfiles/transform_false.yml",
			}
			expected := "failed to merge ./merge_testfiles/transform_false.yml: files are incompatible; " +
				"files with '_transform: true' (default) and '_transform: false' are not compatible"
			expectErr := true

			validateMerge(fileList, expected, expectErr, nil)
		})
	})

	Describe("MustMerge", func() {
		It("succeeds on proper files", func() {
			// This tests the order of the resulting file, but also the version of the
			// final file
			fileList := []string{
				"./merge_testfiles/file1.yml",
				"./merge_testfiles/file2.yml",
				"./merge_testfiles/file3.yml",
			}
			expectedFile := "./merge_testfiles/test1_expected.json"

			res, _ := merge.MustFiles(fileList)
			result := MustSerialize(res, OutputFormatJSON)
			expected := MustReadFile(expectedFile)

			Expect(*result).To(MatchJSON(*expected))
		})

		It("throws error on bad files", func() {
			t := func() {
				merge.MustFiles([]string{"bad_file1.yml", "bad_file2.yml"})
			}
			Expect(t).Should(Panic())
		})
	})
})
