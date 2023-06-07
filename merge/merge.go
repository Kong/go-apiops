package merge

import (
	"fmt"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/logbasics"
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

// MustFiles is identical to `Files` except that it will panic instead of returning
// an error.
func MustFiles(filenames []string) (result map[string]interface{}, history []interface{}) {
	result, info, err := Files(filenames)
	if err != nil {
		panic(err)
	}

	return result, info
}

// Files reads and merges files. Will merge all top-level arrays by simply
// concatenating them. Any other keys will be copied. The files will be processed
// in order provided. An error will be returned if files are incompatible.
// There are no checks on duplicates, etc... garbage-in-garbage-out.
func Files(filenames []string) (result map[string]interface{}, history []interface{}, err error) {
	if len(filenames) == 0 {
		panic("no filenames provided")
	}

	historyArray := make([]interface{}, len(filenames))
	minorVersion := 0

	// traverse all files
	for i, filename := range filenames {
		logbasics.Info("merging file", "filename", filename)

		// read the file
		data, err := filebasics.DeserializeFile(filename)
		if err != nil {
			return nil, nil, err
		}

		newInfo := make(map[string]interface{})
		newInfo["filename"] = filename
		fileHistory := deckformat.HistoryGet(data)
		if len(fileHistory) > 0 {
			newInfo["info"] = fileHistory
		}
		historyArray[i] = newInfo

		if result == nil {
			// set up initial map, ensure it is "compatible" with first entry
			result = make(map[string]interface{})
			if data[deckformat.TransformKey] != nil {
				logbasics.Debug("setting transform meta-field", "value", data[deckformat.TransformKey])
				result[deckformat.TransformKey] = data[deckformat.TransformKey]
			}
			if data[deckformat.VersionKey] != nil {
				logbasics.Debug("setting version meta-field", "value", data[deckformat.VersionKey])
				result[deckformat.VersionKey] = data[deckformat.VersionKey]
			}
		}

		// check compatibility
		if err := deckformat.CompatibleFile(result, data); err != nil {
			return nil, nil, fmt.Errorf("failed to merge %s: %w", filename, err)
		}

		// record minor version
		_, m, _ := deckformat.ParseFormatVersion(data)
		if m > minorVersion {
			// we only track minor version, because majors must be the same to pass the
			// compatibility check above
			logbasics.Debug("updating resulting version", "sourcefile", filename, "new_minor", m)
			minorVersion = m
		}

		result = merge2Files(result, data)
	}

	// set final resulting format version
	if result[deckformat.VersionKey] != nil {
		ma, _, _ := deckformat.ParseFormatVersion(result)
		if ma == 0 {
			delete(result, deckformat.VersionKey)
		} else {
			result[deckformat.VersionKey] = fmt.Sprint(ma, ".", minorVersion)
		}
	}

	return result, historyArray, nil
}
