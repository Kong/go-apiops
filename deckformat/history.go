package deckformat

import "github.com/kong/go-apiops/jsonbasics"

//
//
//  Section for tracking history of the file
//
//

// HistoryGet returns a the history info array. If there is none, or if filedata is nil,
// it will return an empty one.
func HistoryGet(filedata map[string]interface{}) (historyArray []interface{}) {
	if filedata == nil || filedata[HistoryKey] == nil {
		historyInfo := make([]interface{}, 0)
		return historyInfo
	}

	trackInfo, err := jsonbasics.ToArray(filedata[HistoryKey])
	if err != nil {
		// the entry wasn't an array, so wrap it in one
		trackInfo = []interface{}{filedata[HistoryKey]}
	}

	// Return a copy
	return jsonbasics.DeepCopyArray(trackInfo)
}

// HistorySet sets the history info array. Setting to nil will delete the history.
func HistorySet(filedata map[string]interface{}, historyArray []interface{}) {
	if historyArray == nil {
		HistoryClear(filedata)
		return
	}
	filedata[HistoryKey] = historyArray

	// TODO: remove this after the we get support for metafields in deck
	HistoryClear(filedata)
}

// HistoryAppend appends an entry (if non-nil) to the history info array. If there is
// no array, it will create one.
func HistoryAppend(filedata map[string]interface{}, newEntry interface{}) {
	hist := HistoryGet(filedata)
	hist = append(hist, newEntry)
	HistorySet(filedata, hist)
}

func HistoryClear(filedata map[string]interface{}) {
	delete(filedata, HistoryKey)
}

// HistoryNewEntry returns a new JSONobject with tool version and command keys set.
func HistoryNewEntry(cmd string) map[string]interface{} {
	return map[string]interface{}{
		"tool":    ToolVersionString(),
		"command": cmd,
		// For now: no timestamps in git-ops!
		// "time":    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"), // ISO8601 format
	}
}
