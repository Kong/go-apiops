package yamlbasics

import (
	"github.com/kong/go-apiops/logbasics"
	"gopkg.in/yaml.v3"
)

//
//
// NodeSet implementation, just a list of yaml nodes
//
//

// represents a set of yaml nodes
type NodeSet []*yaml.Node

// Intersection returns the intersection of the two sets of nodes.
// nil entries will be ignored. The result will a copy and have no duplicates.
// The second return value is the remainder of set2 after the intersection was removed (also a copy).
func (mainSet *NodeSet) Intersection(set2 NodeSet) (intersection NodeSet, remainder NodeSet) {
	if len(*mainSet) == 0 || len(set2) == 0 {
		intersection := make(NodeSet, 0)
		remainder := make(NodeSet, len(set2))
		copy(remainder, set2)
		return intersection, remainder
	}

	// deduplicate
	seen1 := make(map[*yaml.Node]bool)
	for _, node := range *mainSet {
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

// IsIntersection returns true if all nodes in the subset also appear in the main set.
// nil entries will be ignored. Returns true if subset is empty.
func (mainSet *NodeSet) IsIntersection(subset NodeSet) bool {
	_, remainder := mainSet.Intersection(subset)
	return len(remainder) == 0
}

// Subtract returns the set of nodes that are in mainSet but not in setToSubtract.
// nil entries will be ignored. The result will have no duplicates.
func (mainSet *NodeSet) Subtract(setToSubtract NodeSet) NodeSet {
	_, remainder := setToSubtract.Intersection(*mainSet)
	return remainder
}

// Union returns the union of the two (or more) sets of nodes.
// nil entries will be ignored. The result will have no duplicates.
func (mainSet *NodeSet) Union(sets ...NodeSet) NodeSet {
	union := make(NodeSet, 0)
	sets = append([]NodeSet{*mainSet}, sets...)

	seen := make(map[*yaml.Node]bool)
	for _, nodeset := range sets {
		for _, node := range nodeset {
			if node != nil && !seen[node] {
				seen[node] = true
				union = append(union, node)
			}
		}
	}

	return union
}
