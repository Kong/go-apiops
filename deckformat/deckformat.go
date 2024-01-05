package deckformat

import (
	"fmt"
)

const (
	VersionKey   = "_format_version"
	TransformKey = "_transform"
	HistoryKey   = "_ignore" // the top-level key in deck files for storing history info
)

func init() {
	initPointerCollections()
}

//
//
//  Keeping track of the tool/binary version info (set once at startup)
//
//

var toolInfo = struct {
	name    string
	version string
	commit  string
}{}

// ToolVersionSet can be called once to set the tool info that is reported in the history.
// The 'version' and 'commit' strings are optional. Omitting them (lower cardinality) makes
// for a better GitOps experience, but provides less detail.
func ToolVersionSet(name string, version string, commit string) {
	if toolInfo.name != "" || name == "" {
		panic("the tool information was already set, or cannot be set to an empty string")
	}
	toolInfo.name = name
	toolInfo.version = version
	toolInfo.commit = commit
}

// ToolVersionGet returns the individual components of the info
func ToolVersionGet() (name string, version string, commit string) {
	if toolInfo.name == "" {
		panic("the tool information wasn't set, call ToolVersionSet first")
	}
	return toolInfo.name, toolInfo.version, toolInfo.commit
}

// ToolVersionString returns the info in a single formatted string. eg. "decK 1.2 (123abc)"
func ToolVersionString() string {
	n, v, c := ToolVersionGet()
	if c != "" {
		return fmt.Sprintf("%s %s (%s)", n, v, c)
	}
	if v != "" {
		return fmt.Sprintf("%s %s", n, v)
	}
	return n
}
