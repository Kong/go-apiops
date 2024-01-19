package yamlbasics

import (
	"fmt"

	"github.com/kong/go-apiops/logbasics"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

//
//
// NodeSet implementation, just a list of yaml nodes
//
//

// represents a set of yaml nodes
type NodeSet []*yaml.Node

// IsIntersection returns true if all nodes in the subset also appear in the main set.
// nil entries will be ignored. Returns true if subset is empty.
func IsIntersection(mainSet NodeSet, subset NodeSet) bool {
	if len(subset) == 0 {
		return true
	}
	if len(mainSet) == 0 {
		return false
	}

	// deduplicate
	seen := make(map[*yaml.Node]bool)
	for _, node := range mainSet {
		if node != nil {
			seen[node] = true
		}
	}

	for _, node := range subset {
		if node != nil && !seen[node] {
			return false
		}
	}
	return true
}

// Intersection returns the intersection of the two given sets of nodes.
// nil entries will be ignored. The result will have no duplicates.
// the second return value is the remainder of set2 after the intersection was removed.
func Intersection(set1, set2 NodeSet) (intersection NodeSet, remainder NodeSet) {
	if len(set1) == 0 || len(set2) == 0 {
		return make(NodeSet, 0), make(NodeSet, 0)
	}

	// deduplicate
	seen1 := make(map[*yaml.Node]bool)
	for _, node := range set1 {
		if node != nil {
			seen1[node] = true
		}
	}

	intersection = make(NodeSet, 0)
	remainder = make(NodeSet, 0)
	seen2 := make(map[*yaml.Node]bool)
	for _, node := range set2 {
		if node != nil && !seen2[node] {
			seen2[node] = true
			if seen1[node] {
				intersection = append(intersection, node)
			} else {
				remainder = append(remainder, node)
			}
		}
	}
	logbasics.Debug("intersection", "#found", len(intersection), "#remainder", len(remainder))
	return intersection, remainder
}

// SubtractSet returns the set of nodes that are in mainSet but not in setToSubtract.
// nil entries will be ignored. The result will have no duplicates.
func SubtractSet(mainSet NodeSet, setToSubtract NodeSet) NodeSet {
	// TODO: this can be implemneted by using Intersection, and returning the remainderset.
	if len(mainSet) == 0 || len(setToSubtract) == 0 {
		return make(NodeSet, 0)
	}

	// deduplicate
	seen1 := make(map[*yaml.Node]bool)
	for _, node := range setToSubtract {
		if node != nil {
			seen1[node] = true
		}
	}

	subtracted := make(NodeSet, 0)
	seen2 := make(map[*yaml.Node]bool)
	for _, node := range mainSet {
		if node != nil && !seen1[node] && !seen2[node] {
			seen2[node] = true
			subtracted = append(subtracted, node)
		}
	}
	return subtracted
}

//
//
// SelectorSet implementation, handles mutiple instead of 1 JSONpath selector
//
//

// Represents a set of JSONpath selectors. Call NewSelectorSet to create one.
// The SelectorSet can be empty, in which case it will return only empty results.
type SelectorSet struct {
	selectors   []*yamlpath.Path // the compiled selectors
	source      []string         // matching source strings of the selectors
	initialized bool             // indicator whether is was initialized or not
}

// NewSelectorSet compiles the given selectors into a list of yaml nodes.
// If any of the selectors is invalid, an error will be returned.
// If the selectors are omitted/empty then an empty set is returned.
func NewSelectorSet(selectors []string) (SelectorSet, error) {
	var (
		set SelectorSet
		err error
	)

	set.selectors = make([]*yamlpath.Path, len(selectors))
	set.source = make([]string, len(selectors))
	for i, selector := range selectors {
		set.source[i] = selector
		set.selectors[i], err = yamlpath.NewPath(selector)
		if err != nil {
			return SelectorSet{}, fmt.Errorf("selector '%s' is not a valid JSONpath expression; %w", selector, err)
		}
	}
	set.initialized = true
	return set, nil
}

// IsEmpty returns true if the selector set is empty.
func (set *SelectorSet) IsEmpty() bool {
	return set.selectors == nil || len(set.selectors) == 0
}

// GetSources returns a copy of the selector sources
func (set *SelectorSet) GetSources() []string {
	sources := make([]string, 0)
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
	if set.selectors == nil || len(set.selectors) == 0 {
		return make(NodeSet, 0), nil
	}

	results := make(NodeSet, 0)
	seen := make(map[*yaml.Node]bool)
	for i, selector := range set.selectors {
		matches, err := selector.Find(nodeToSearch)
		if err != nil {
			return nil, fmt.Errorf("failed to execute selector '%s'; %w", set.source[i], err)
		}
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
