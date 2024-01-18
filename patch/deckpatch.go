package patch

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/yamlbasics"
	"gopkg.in/yaml.v3"
)

var DefaultSelector = []string{"$"}

// DeckPatch models a single DeckPatch that can be applied on a deckfile.
type DeckPatch struct {
	// Format         string                 // Name of the format specified
	Selectors yamlbasics.SelectorSet // compiled JSONpath queries
	ObjValues map[string]interface{} // Values to set on target objects
	ArrValues []interface{}          // Values to set on target arrays
	Remove    []string               // List of keys to remove from the target object
	// Patch          map[string]interface{} // RFC-7396
	// Operations     []interface{}          // RFC-6902
	initialized bool
}

// NewDeckPatch creates a new DeckPatch object. Returns an error if any of the given
// Selectors fails to compile. The Selectors default to "$" if not given.
// Alternatively you can use the Parse() method to initialize the DeckPatch.
func NewDeckPatch(
	selectors []string,
	objValues map[string]interface{},
	arrValues []interface{},
	remove []string,
) (*DeckPatch, error) {
	if len(selectors) == 0 {
		selectors = DefaultSelector
	}

	selectorSet, err := yamlbasics.NewSelectorSet(selectors)
	if err != nil {
		return nil, err
	}

	return &DeckPatch{
		Selectors:   selectorSet,
		ObjValues:   objValues,
		ArrValues:   arrValues,
		Remove:      remove,
		initialized: true,
	}, nil
}

// Parse will parse JSONobject into a DeckPatch.
// selector is optional, default to "$". If given MUST be a string, and a valid JSONpath.
// values is optional, defaults to empty map. If given, MUST be an object.
// remove is optional, defaults to empty array. If given MUST be an array. Non-string entries will be ignored.
func (patch *DeckPatch) Parse(obj map[string]interface{}, breadCrumb string) (err error) {
	selectorSources, err := jsonbasics.GetStringArrayField(obj, "selectors")
	if err != nil {
		// selector is present, but not a string-array, error out
		return fmt.Errorf("%s.selectors is not a string-array", breadCrumb)
	}
	if obj["selectors"] == nil {
		// not present, so set default
		logbasics.Info("No selectors specified", "key", breadCrumb+".selectors", "default", DefaultSelector)
		selectorSources = DefaultSelector
	}

	// compile JSONpath expressions
	patch.Selectors, err = yamlbasics.NewSelectorSet(selectorSources)
	if err != nil {
		return fmt.Errorf("%s.selectors[*] contains an invalid JSONpath expression; %w", breadCrumb, err)
	}

	patch.ObjValues, err = jsonbasics.ToObject(obj["values"])
	if err != nil {
		patch.ObjValues = make(map[string]interface{}) // set default; empty object

		if obj["values"] != nil {
			// "values" is present, but wasn't an object,
			patch.ArrValues, err = jsonbasics.ToArray(obj["values"])
			if err != nil {
				// It's also not an array, error out
				return fmt.Errorf("%s.values is neither an object nor an array", breadCrumb)
			}
		}
	} else {
		patch.ArrValues = make([]interface{}, 0) // set default; empty array
	}

	patch.Remove, err = jsonbasics.GetStringArrayField(obj, "remove")
	if err != nil {
		return fmt.Errorf("%s.remove is not an array", breadCrumb)
	}

	for _, removeKey := range patch.Remove {
		_, found := patch.ObjValues[removeKey]
		if found {
			return fmt.Errorf("%s is trying to change and remove '%s' at the same time", breadCrumb, removeKey)
		}
	}

	patch.initialized = true
	return nil
}

// ApplyToObjectNode applies the DeckPatch on a JSONobject. The yaml.Node MUST
// be of type "MappingNode" (JSONobject), otherwise it panics.
func (patch *DeckPatch) ApplyToObjectNode(node *yaml.Node) error {
	if !patch.initialized {
		panic("DeckPatch not initialized, call patch.Parse() or NewDeckPatch() first")
	}
	if node == nil || node.Kind != yaml.MappingNode {
		panic("expected node to be a yaml.Node type MappingNode")
	}

	// keep track of the fields we already processed
	handledFields := make(map[string]bool)

	// a mapping node has 2 entries for each key-value pair in its
	// node.Content array
	for i := 0; i < len(node.Content); {
		keyNode := node.Content[i]
		key := keyNode.Value

		newData, found := patch.ObjValues[key]
		if found {
			// we have an updated value for this key, set it
			node.Content[i+1] = jsonbasics.ConvertToYamlNode(newData)
			handledFields[key] = true
		}
		i = i + 2

		for _, deleteKey := range patch.Remove {
			if key == deleteKey {
				// Note: not moving pointer forward, since we deleted elements
				i = i - 2
				// delete the entry
				node.Content = append(node.Content[:i], node.Content[i+2:]...)
			}
		}
	}

	// add any field not handled yet (wasn't in the original object)
	for fieldName, newValue := range patch.ObjValues {
		if !handledFields[fieldName] {
			keyNode := yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fieldName,
				Style: yaml.DoubleQuotedStyle,
			}
			valueNode := jsonbasics.ConvertToYamlNode(newValue)
			node.Content = append(node.Content, &keyNode, valueNode)
		}
	}

	return nil
}

// ApplyToArrayNode applies the DeckPatch on a JSONarray. The yaml.Node MUST
// be of type "SequenceNode" (JSONarray), otherwise it panics.
func (patch *DeckPatch) ApplyToArrayNode(node *yaml.Node) error {
	if !patch.initialized {
		panic("DeckPatch not initialized, call patch.Parse() or NewDeckPatch() first")
	}
	if node == nil || node.Kind != yaml.SequenceNode {
		panic("expected node to be a yaml.Node type SequenceNode")
	}

	for _, nodeToAppend := range patch.ArrValues {
		node.Content = append(node.Content, jsonbasics.ConvertToYamlNode(nodeToAppend))
	}

	return nil
}

// ApplyToNodes queries the yamlData using the selector, and applies the patch on every Object
// returned. Any non-objects returned by the selector will be ignored.
// If Selector wasn't set yet, will try and create it from the SelectorSource.
func (patch *DeckPatch) ApplyToNodes(yamlData *yaml.Node) (err error) {
	if !patch.initialized {
		panic("DeckPatch not initialized, call patch.Parse() or NewDeckPatch() first")
	}
	if len(patch.ObjValues) == 0 && len(patch.Remove) == 0 && len(patch.ArrValues) == 0 {
		// return early if there are no changes to apply, to not trip on the selector
		return nil
	}

	if patch.Selectors.IsEmpty() {
		logbasics.Info("Patch has no selectors specified")
	}

	// query the yamlData using the selector
	nodes, err := patch.Selectors.Find(yamlData)
	if err != nil {
		return err
	}

	// 'nodes' is an array of nodes matching the selectors
	for _, node := range nodes {
		if len(patch.ArrValues) > 0 {
			// since we're updating array fields, we'll skip anything that is
			// not a JSONarray
			if node.Kind == yaml.SequenceNode {
				err = patch.ApplyToArrayNode(node)
				if err != nil {
					return err
				}
			}
		} else {
			// since we're updating object fields, we'll skip anything that is
			// not a JSONobject
			if node.Kind == yaml.MappingNode {
				err = patch.ApplyToObjectNode(node)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
