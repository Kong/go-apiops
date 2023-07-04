package tags

import (
	"fmt"
	"sort"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/yamlbasics"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

const tagArrayName = "tags"

// defaultSelectors is the list of JSONpointers to entities that can hold tags
var defaultSelectors []string

func init() {
	defaultSelectors = make([]string, 0)
	for _, selectors := range deckformat.EntityPointers {
		defaultSelectors = append(defaultSelectors, selectors...)
	}
}

type Tagger struct {
	// list of JSONpointers to entities that can hold tags, so the selector
	// returns entities that can hold tags, not the tag arrays themselves
	selectors []*yamlpath.Path
	// list of Nodes (selected by the selectors) representing entities that can
	// hold tags, not the tag arrays themselves
	tagOwners []*yaml.Node
	// The document to operate on
	data *yaml.Node
}

// SetData sets the Yaml document to operate on. Cannot be set to nil (panic).
func (ts *Tagger) SetData(data map[string]interface{}) {
	if data == nil {
		panic("data cannot be nil")
	}
	ts.data = jsonbasics.ConvertToYamlNode(data)
	ts.tagOwners = nil // clear previous JSONpointer search results
}

// GetData returns the (modified) document.
func (ts *Tagger) GetData() map[string]interface{} {
	d := jsonbasics.ConvertToJSONInterface(ts.data)
	return (*d).(map[string]interface{})
}

// SetSelectors sets the selectors to use. If empty (or nil), the default selectors
// are set.
func (ts *Tagger) SetSelectors(selectors []string) error {
	if len(selectors) == 0 {
		logbasics.Debug("no selectors provided, using defaults")
		selectors = defaultSelectors
	}

	compiledSelectors := make([]*yamlpath.Path, len(selectors))
	for i, selector := range selectors {
		logbasics.Debug("compiling JSONpath", "path", selector)
		compiledpath, err := yamlpath.NewPath(selector)
		if err != nil {
			return fmt.Errorf("selector '%s' is not a valid JSONpath expression; %w", selector, err)
		}
		compiledSelectors[i] = compiledpath
	}
	// we're good, they are all valid
	ts.selectors = compiledSelectors
	ts.tagOwners = nil // clear previous JSONpointer search results
	logbasics.Debug("successfully compiled JSONpaths")
	return nil
}

// Search searches the document using the selectors, and stores the results
// internally. Only results that are JSONobjects are stored.
// The search is performed only once, and the results are cached.
// Will panic if data (the document to search) has not been set yet.
func (ts *Tagger) search() error {
	if ts.tagOwners != nil { // already searched
		return nil
	}

	if ts.data == nil {
		panic("data hasn't been set, see SetData()")
	}

	if ts.selectors == nil {
		err := ts.SetSelectors(nil) // set to 'nil' to set the default selectors
		if err != nil {
			panic("this should never happen, since we're setting the default selectors")
		}
	}

	// build list of targets by executing the selectors one by one
	targets := make([]*yaml.Node, 0)
	refs := make(map[*yaml.Node]bool, 0) // keep track of already found nodes
	for idx, selector := range ts.selectors {
		nodes, err := selector.Find(ts.data)
		if err != nil {
			return err
		}

		// 'nodes' is an array of nodes matching the selector
		objCount := 0
		for _, node := range nodes {
			// since we're updating object fields, we'll skip anything that is
			// not a JSONobject
			if node.Kind == yaml.MappingNode && !refs[node] {
				refs[node] = true
				targets = append(targets, node)
				objCount++
			}
		}
		logbasics.Debug("selector results", "selector", idx, "results", len(nodes), "objects", objCount)
	}
	ts.tagOwners = targets
	return nil
}

// RemoveTags removes the listed tags from any entity selected. Empty tag arrays are
// removed if 'removeEmptyTagArrays' is true. The order of the remaining tags is
// preserved.
func (ts *Tagger) RemoveTags(tags []string, removeEmptyTagArrays bool) error {
	if len(tags) == 0 {
		if !removeEmptyTagArrays {
			logbasics.Debug("no tags to remove", "tags", tags)
			return nil
		}
		logbasics.Debug("no tags to remove, removing empty tag-arrays")
		tags = []string{}
	}

	reverseTags := make(map[string]bool)
	for _, tag := range tags {
		reverseTags[tag] = true
	}

	if err := ts.search(); err != nil {
		return err
	}

	for _, node := range ts.tagOwners {
		// 'node' is a JSONobject that can hold tags, type is yaml.MappingNode
		tagArray := yamlbasics.GetFieldValue(node, tagArrayName)
		if tagArray != nil && tagArray.Kind == yaml.SequenceNode {
			// we found the tags array on this object

			// loop over this tags array to find the tags to remove
			for j := 0; j < len(tagArray.Content); j++ {
				tagNode := tagArray.Content[j]
				if tagNode.Kind == yaml.ScalarNode {
					tag := tagNode.Value

					// is this tag in the list of tags to remove?
					if reverseTags[tag] {
						tagArray.Content = append(tagArray.Content[:j], tagArray.Content[j+1:]...)
						j-- // we're removing a node, so we need to go back one node
					}
				}
			}

			// if the tags array is empty remove it
			if removeEmptyTagArrays && len(tagArray.Content) == 0 {
				yamlbasics.RemoveField(node, tagArrayName)
			}
		}
	}
	return nil
}

// AddTags adds the listed tags to any entity selected. The tags are added in the
// order they are listed.
func (ts *Tagger) AddTags(tags []string) error {
	if len(tags) == 0 {
		logbasics.Debug("no tags to add", "tags", tags)
		return nil
	}

	if err := ts.search(); err != nil {
		return err
	}

	for _, node := range ts.tagOwners {
		// 'node' is a JSONobject that can hold tags, type is yaml.MappingNode

		tagsArray := yamlbasics.GetFieldValue(node, tagArrayName)

		// if we didn't find the tags array, create it
		if tagsArray == nil || tagsArray.Kind != yaml.SequenceNode {
			tagsArray = yamlbasics.NewArray()
			yamlbasics.SetFieldValue(node, tagArrayName, tagsArray)
		}

		// loop over the tags to add
		for _, tag := range tags {
			// loop over this tags array to find the tags to add
			found := false
			for _, tagNode := range tagsArray.Content {
				if tagNode.Value == tag && tagNode.Kind == yaml.ScalarNode {
					found = true
					break
				}
			}

			// if the tag is not already in the array, add it
			if !found {
				_ = yamlbasics.Append(tagsArray, yamlbasics.NewString(tag))
			}
		}
	}
	return nil
}

// ListTags returns a list of the tags in use in the data. The tags are sorted.
func (ts *Tagger) ListTags() ([]string, error) {
	if err := ts.search(); err != nil {
		return nil, err
	}

	tags := make(map[string]bool)
	for _, node := range ts.tagOwners {
		// 'node' is a JSONobject that can hold tags, type is yaml.MappingNode
		// loop over the object to find the tags array
		for i := 0; i < len(node.Content); i += 2 {
			tagsArray := yamlbasics.GetFieldValue(node, tagArrayName)
			if tagsArray != nil && tagsArray.Kind == yaml.SequenceNode {
				// we found the tags array on this object

				// loop over this tags array to find the tags
				for _, tagNode := range tagsArray.Content {
					if tagNode.Kind == yaml.ScalarNode {
						tags[tagNode.Value] = true
					}
				}
			}
		}
	}

	tagList := make([]string, 0, len(tags))
	for tag := range tags {
		tagList = append(tagList, tag)
	}
	sort.Strings(tagList)
	return tagList, nil
}

// RemoveUnknownTags removes all tags that are not in the list of known tags.
// If removeEmptyTagArrays is true, it will also remove any empty tags arrays.
func (ts *Tagger) RemoveUnknownTags(knownTags []string, removeEmptyTagArrays bool) error {
	existingTags, err := ts.ListTags()
	if err != nil {
		return err
	}

	tagsToRemove := make([]string, 0, len(existingTags))
	for _, tag := range existingTags {
		found := false
		for _, knownTag := range knownTags {
			if tag == knownTag {
				found = true
				break
			}
		}
		if !found {
			tagsToRemove = append(tagsToRemove, tag)
		}
	}
	if len(tagsToRemove) > 0 {
		if err := ts.RemoveTags(tagsToRemove, removeEmptyTagArrays); err != nil {
			return err
		}
	}

	return nil
}
