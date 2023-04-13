package patch

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

const DefaultSelector = "$"

// DeckPatch models a single DeckPatch that can be applied on a deckfile.
type DeckPatch struct {
	// Format         string                 // Name of the format specified
	SelectorSource string                 // Source query for the JSONpath object
	Selector       *yamlpath.Path         // JSONpath object
	Values         map[string]interface{} // Values to set on target objects
	Remove         []string               // List of keys to remove from the target object
	// Patch          map[string]interface{} // RFC-7396
	// Operations     []interface{}          // RFC-6902
}

// Parse will parse JSONobject into a DeckPatch.
// selector is optional, default to "$". If given MUST be a string, and a valid JSONpath.
// values is optional, defaults to empty map. If given, MUST be an object.
// remove is optional, defaults to empty array. If given MUST be an array. Non-string entries will be ignored.
func (patch *DeckPatch) Parse(obj map[string]interface{}, breadCrumb string) (err error) {
	patch.SelectorSource, err = jsonbasics.GetStringField(obj, "selector")
	if err != nil {
		if obj["selector"] != nil {
			// selector is present, but not a string, error out
			return fmt.Errorf("%s.selector is not a string", breadCrumb)
		}
		// not present, so set default
		patch.SelectorSource = DefaultSelector
	}
	patch.Selector, err = yamlpath.NewPath(patch.SelectorSource)
	if err != nil {
		return fmt.Errorf("%s.selector is not a valid JSONpath expression; %w", breadCrumb, err)
	}

	patch.Values, err = jsonbasics.ToObject(obj["values"])
	if err != nil {
		if obj["values"] != nil {
			// selector is present, but not an object, error out
			return fmt.Errorf("%s.values is not an object", breadCrumb)
		}
		// not present, so set default; empty object
		patch.Values = make(map[string]interface{})
	}

	patch.Remove, err = jsonbasics.GetStringArrayField(obj, "remove")
	if err != nil {
		return fmt.Errorf("%s.remove is not an array", breadCrumb)
	}

	for _, removeKey := range patch.Remove {
		_, found := patch.Values[removeKey]
		if found {
			return fmt.Errorf("%s is trying to change and remove '%s' at the same time", breadCrumb, removeKey)
		}
	}

	return nil
}

// ApplyToNode applies the DeckPatch on a JSONobject. The yaml.Node MUST
// be of type "MappingNode" (JSONobject), otherwise it panics.
func (patch *DeckPatch) ApplyToNode(node *yaml.Node) error {
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

		newData, found := patch.Values[key]
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
	for fieldName, newValue := range patch.Values {
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

// ApplyToNodes queries the yamlData using the selector, and applies the patch on every Object
// returned. Any non-objects returned by the selector will be ignored.
// If Selector wasn't set yet, will try and create it from the SelectorSource.
func (patch *DeckPatch) ApplyToNodes(yamlData *yaml.Node) (err error) {
	if len(patch.Values) == 0 && len(patch.Remove) == 0 {
		// return early if there are no changes to apply, to not trip on the selector
		return nil
	}

	if patch.Selector == nil {
		patch.Selector, err = yamlpath.NewPath(patch.SelectorSource)
		if err != nil {
			return fmt.Errorf("selector '%s' is not a valid JSONpath expression; %w", patch.SelectorSource, err)
		}
	}

	nodes, err := patch.Selector.Find(yamlData)
	if err != nil {
		return err
	}

	// 'nodes' is an array of nodes matching the selector
	for _, node := range nodes {
		// since we're updating object fields, we'll skip anything that is
		// not a JSONobject
		if node.Kind == yaml.MappingNode {
			err = patch.ApplyToNode(node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
