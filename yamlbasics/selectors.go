package yamlbasics

import (
	"fmt"

	"github.com/kong/go-apiops/logbasics"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"go.yaml.in/yaml/v4"
)

//
//
// SelectorSet implementation, handles multiple instead of 1 JSONpath selector
//
//

// Represents a set of JSONpath selectors. Call NewSelectorSet to create one.
// The SelectorSet can be empty, in which case it will return only empty results.
type SelectorSet struct {
	selectors   []*jsonpath.JSONPath // the compiled selectors
	source      []string             // matching source strings of the selectors
	initialized bool                 // indicator whether is was initialized or not
}

// NewSelectorSet compiles the given selectors into a list of yaml nodes.
// If any of the selectors is invalid, an error will be returned.
// If the selectors are omitted/empty then an empty set is returned.
func NewSelectorSet(selectors []string) (SelectorSet, error) {
	var (
		set SelectorSet
		err error
	)

	set.selectors = make([]*jsonpath.JSONPath, len(selectors))
	set.source = make([]string, len(selectors))
	for i, selector := range selectors {
		set.source[i] = selector
		set.selectors[i], err = jsonpath.NewPath(selector)
		if err != nil {
			return SelectorSet{}, fmt.Errorf("selector '%s' is not a valid JSONpath expression; %w", selector, err)
		}
	}
	set.initialized = true
	return set, nil
}

// IsEmpty returns true if the selector set is empty.
func (set *SelectorSet) IsEmpty() bool {
	return len(set.selectors) == 0
}

// GetSources returns a copy of the selector sources
func (set *SelectorSet) GetSources() []string {
	sources := make([]string, len(set.source))
	copy(sources, set.source)
	return sources
}

// Find executes the given selectors on the given yaml node.
// The result will never be nil, will not have duplicates, but can be an empty array.
// An error is only returned if any of the selectors errors when searching.
// nodeToSearch cannot be nil, in which case it will panic.
func (set *SelectorSet) Find(nodeToSearch *yaml.Node) (NodeSet, error) {
	if !set.initialized {
		panic("selector set uninitialized, call NewSelectorSet to create and initialize one")
	}
	if nodeToSearch == nil {
		panic("expected nodeToSearch to be non-nil")
	}
	if len(set.selectors) == 0 {
		return make(NodeSet, 0), nil
	}

	results := make(NodeSet, 0)
	seen := make(map[*yaml.Node]bool)
	for i, selector := range set.selectors {
		matches := selector.Query(nodeToSearch)
		logbasics.Debug("selector results", "selector", set.source[i], "#found", len(matches))

		for _, match := range matches {
			if match != nil && !seen[match] {
				results = append(results, match)
				seen[match] = true
			}
		}
	}

	return results, nil
}
