package merge

import (
	"fmt"
	"sort"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
)

func merge2Files(data1 map[string]interface{}, data2 map[string]interface{}) map[string]interface{} {
	mergedData := make(map[string]interface{})

	for key, value := range data1 {
		mergedData[key] = value
	}

	for key, value := range data2 {
		if existingValue, ok := mergedData[key]; ok {
			// target already has this key
			if a, ok := existingValue.([]interface{}); ok {
				// we currently have an array
				if b, ok := value.([]interface{}); ok {
					// the new value also is an array, so append it
					mergedData[key] = append(a, b...)
				} else {
					// the new value is not an array, overwrite the existing array
					mergedData[key] = value
				}
			} else {
				// existing value is not an array, overwrite it with the new value
				mergedData[key] = value
			}
		} else {
			// key doesn't exist in the target, so just insert
			mergedData[key] = value
		}
	}

	return mergedData
}

// MustFiles is identical to `Files` except that it will panic instead of return
// an error.
func MustFiles(filenames []string) map[string]interface{} {
	result, err := Files(filenames)
	if err != nil {
		panic(err)
	}

	return result
}

// MergeFiles reads and merges files. Will merge all top-level arrays by simply
// concatenating them. Any other keys will be copied. The files will be processed
// in order of the '_format_version' field in the file (an omitted version defaults
// to "0.0"). An error will be returned if files are incompatible.
// There are no checks on duplicates, etc... garbage-in-garbage-out.
func Files(filenames []string) (map[string]interface{}, error) {
	if len(filenames) == 0 {
		panic("no filenames provided")
	}

	// sort files by version to ensure compatibility
	type torder struct {
		filename string
		data     map[string]interface{}
		order    int
	}
	ordered := make([]torder, len(filenames))

	for i, filename := range filenames {
		data, err := filebasics.DeserializeFile(filename)
		if err != nil {
			return nil, err
		}

		// omitted versions are considered to be "0.0"
		major, minor := 0, 0
		if data[deckformat.VersionKey] != nil {
			major, minor, err = deckformat.ParseFormatVersion(data)
			if err != nil {
				return nil, fmt.Errorf("failed to merge %s: %w", filename, err)
			}
		}

		ordered[i] = torder{
			filename: filename,
			order:    major*1000*1000 + minor*1000 + i,
			data:     data,
		}
	}

	sort.Slice(ordered, func(i1, i2 int) bool {
		return ordered[i1].order < ordered[i2].order
	})

	// set up initial map, ensure it is "compatible" with first entry
	result := make(map[string]interface{})
	if ordered[0].data[deckformat.TransformKey] != nil {
		result[deckformat.TransformKey] = ordered[0].data[deckformat.TransformKey]
	}
	if ordered[0].data[deckformat.VersionKey] != nil {
		result[deckformat.VersionKey] = ordered[0].data[deckformat.VersionKey]
	}

	// traverse all files
	for _, fileEntry := range ordered {
		if err := deckformat.CompatibleFile(result, fileEntry.data); err != nil {
			return nil, fmt.Errorf("failed to merge %s: %w", fileEntry.filename, err)
		}

		result = merge2Files(result, fileEntry.data)
	}

	return result, nil
}
