// Package plugins provides a way to manage plugins in a declarative Kong configuration file.
// The targets are selected using JSONpath selectors.
// The default is to add the plugins to the main `plugins` array.
// Nested plugins cannot have references to other entities (eg. a plugin nested in a route
// cannot reference a service). This is a limitation of the Kong configuration file format.
package plugins

import (
	"fmt"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/yamlbasics"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"go.yaml.in/yaml/v4"
)

// defaultSelectors is the main `plugins` array.
var defaultSelectors = []string{"$"}

// (constant) list of foreign keys that a plugin can have (field names)
var foreignKeys = []string{"service", "route", "consumer", "consumer_group"}

// the separator when constructing cache-keys for foreign keys
const foreignKeySeparator = ":"

// noForeignKeys is the (constant) key for plugins that have no foreign keys
var noForeignKeys string // dynamically initialized in init()

func init() {
	empty := yamlbasics.NewObject()
	noForeignKeys = ForeignKey(empty)
}

// ForeignKey returns a key combining all foreign key values of the given plugin.
// If the plugin has no foreign keys, it returns an empty string.
func ForeignKey(plugin *yaml.Node) string {
	var sb strings.Builder
	for _, key := range foreignKeys {
		node := yamlbasics.GetFieldValue(plugin, key)
		if node != nil && node.Kind == yaml.ScalarNode {
			sb.WriteString(node.Value)
		}
		sb.WriteString(foreignKeySeparator)
	}
	return sb.String()
}

// HasForeignKeys returns true if the given Plugin node has foreign keys (eg. service,
// route, consumer, consumer_group). These plugins cannot be nested.
func HasForeignKeys(plugin *yaml.Node) bool {
	return ForeignKey(plugin) != noForeignKeys
}

// Plugger is the main struct for managing plugins.
type Plugger struct {
	// list of JSONpointers to entities that can hold plugins, so the selector
	// returns entities that can hold plugins, not the plugin arrays themselves.
	// The default value is the main plugins array (at the file top-level).
	selectors []*jsonpath.JSONPath
	// list of Nodes (selected by the selectors) representing entities that can
	// hold plugins, not the plugin arrays themselves
	pluginOwners []*yaml.Node
	// the array node referencing the top-level "plugins" array. In this array
	// we need to also compare scopes before replacing plugins.
	pluginMain *yaml.Node
	// The document to operate on
	data *yaml.Node
}

// SetData sets the Yaml document to operate on. Cannot be set to nil (panic).
func (ts *Plugger) SetData(data map[string]interface{}) {
	if data == nil {
		panic("data cannot be nil")
	}
	ts.SetYamlData(jsonbasics.ConvertToYamlNode(data))
}

// SetYamlData sets the Yaml document to operate on. Cannot be set to nil (panic).
func (ts *Plugger) SetYamlData(data *yaml.Node) {
	if data == nil {
		panic("data cannot be nil")
	}
	ts.data = data
	ts.pluginOwners = nil // clear previous JSONpointer search results
	ts.pluginMain = nil
}

// GetData returns the (modified) document.
func (ts *Plugger) GetData() map[string]interface{} {
	d := jsonbasics.ConvertToJSONInterface(ts.data)
	return (*d).(map[string]interface{})
}

// GetYamlData returns the (modified) document.
func (ts *Plugger) GetYamlData() *yaml.Node {
	return ts.data
}

// SetSelectors sets the selectors to use. If empty (or nil), the default selectors
// are set.
func (ts *Plugger) SetSelectors(selectors []string) error {
	if len(selectors) == 0 {
		logbasics.Debug("no selectors provided, using defaults", defaultSelectors)
		selectors = defaultSelectors
	}

	compiledSelectors := make([]*jsonpath.JSONPath, len(selectors))
	for i, selector := range selectors {
		logbasics.Debug("compiling JSONpath", "path", selector)
		compiledpath, err := jsonpath.NewPath(selector)
		if err != nil {
			return fmt.Errorf("selector '%s' is not a valid JSONpath expression; %s", selector, err.Error())
		}
		compiledSelectors[i] = compiledpath
	}
	// we're good, they are all valid
	ts.selectors = compiledSelectors
	ts.pluginOwners = nil // clear previous JSONpointer search results
	ts.pluginMain = nil
	logbasics.Debug("successfully compiled JSONpaths", selectors)
	return nil
}

// Search searches the document using the selectors, and stores the results
// internally. Only results that are JSONobjects are stored.
// The search is performed only once, and the results are cached.
// Will panic if data (the document to search) has not been set yet.
func (ts *Plugger) search() error {
	if ts.pluginOwners != nil { // already searched
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
	refs := make(map[*yaml.Node]bool, 0) // keeps references to prevent duplicates

	for idx, selector := range ts.selectors {
		results := selector.Query(ts.data)

		// 'results' contains YAML nodes matching the selector
		objCount := 0
		for _, node := range results {
			// since we're updating object fields, we'll skip anything that is
			// not a JSONobject
			if node.Kind == yaml.MappingNode && !refs[node] {
				refs[node] = true
				targets = append(targets, node)
				objCount++
			}
		}
		logbasics.Debug("selector results", "selector", idx, "results", len(results), "objects", objCount)
	}
	ts.pluginOwners = targets

	// find top-level 'plugins' array
	main := yamlbasics.GetFieldValue(ts.data, "plugins")
	if main != nil && main.Kind == yaml.SequenceNode {
		ts.pluginMain = main
	} else {
		ts.pluginMain = nil
	}

	return nil
}

// AddPlugin adds the given plugin config to any entity selected, unless the
// entity already has a plugin with the same name, in which case it is skipped.
// If 'overwrite' is true, then the existing plugin is overwritten.
func (ts *Plugger) AddPlugin(plugin map[string]interface{}, overwrite bool) error {
	if plugin == nil {
		panic("plugin cannot be nil")
	}
	return ts.AddPlugins([]map[string]interface{}{plugin}, overwrite)
}

// AddPlugins adds the given plugin configs to any entity selected, unless the
// entity already has a plugin with the same name, in which case it is skipped.
// If 'overwrite' is true, then the existing plugin is overwritten.
// If foreign keys are specified, then they are only aloowed in the main plugin array,
// and they will be compared by plugin name AND their scopes.
func (ts *Plugger) AddPlugins(plugins []map[string]interface{}, overwrite bool) error {
	if err := ts.search(); err != nil {
		return err
	}

	// foreignkeys can only be added to the main plugin array, so only allow them if
	// we have 1 plugin owner, and it's the main plugin array.
	// ts.pluginMain might not exist yet, so check the parent
	foreignKeysSupported := len(ts.pluginOwners) == 1 && ts.pluginOwners[0] == ts.data

	// validate the contents, plugins can't be nil, and foreign keys are only allowed in the main plugin array
	pluginNodes := make([]*yaml.Node, len(plugins))
	for i, plugin := range plugins {
		if plugin == nil {
			return fmt.Errorf("plugin %d is nil", i)
		}
		pluginNode := jsonbasics.ConvertToYamlNode(plugin)
		pluginNodes[i] = pluginNode
		if !foreignKeysSupported && ForeignKey(pluginNode) != noForeignKeys {
			return fmt.Errorf("plugin %d has foreign keys, but they are only supported in the main plugin array", i)
		}
	}

	// add the plugins
	for _, plugin := range pluginNodes {
		err := ts.addPluginToOwners(plugin, overwrite)
		if err != nil {
			return err
		}
	}
	return nil
}

// addPluginToOwners adds the given plugin to all owners
func (ts *Plugger) addPluginToOwners(newPlugin *yaml.Node, overwrite bool) error {
	pluginName := "" // the plugin-name of the plugin to add
	{
		nameNode := yamlbasics.GetFieldValue(newPlugin, "name")
		if nameNode != nil && nameNode.Kind == yaml.ScalarNode {
			pluginName = nameNode.Value
			logbasics.Info("adding plugin", "name", pluginName)
		} else {
			logbasics.Info("plugin has no name", "plugin", newPlugin)
		}
	}

	// genericSearcher searches only in the Plugin array by plugin name NOT scope
	genericSearcher := func(pluginNode *yaml.Node) (bool, error) {
		if pluginNode.Kind != yaml.MappingNode {
			return false, nil
		}

		nameNode := yamlbasics.GetFieldValue(pluginNode, "name")
		if nameNode == nil || nameNode.Kind != yaml.ScalarNode {
			return false, nil
		}

		// compare by name only
		return nameNode.Value == pluginName, nil
	}

	// mainSearcher searches in the main plugin array by plugin name AND scope
	var mainSearcher yamlbasics.YamlArrayMatcher
	{
		pluginForeignKey := ForeignKey(newPlugin)
		mainSearcher = func(pluginNode *yaml.Node) (bool, error) {
			if pluginNode.Kind != yaml.MappingNode {
				return false, nil
			}

			nameNode := yamlbasics.GetFieldValue(pluginNode, "name")
			if nameNode == nil || nameNode.Kind != yaml.ScalarNode {
				return false, nil
			}

			// compare by foreign keys and the name
			return ForeignKey(pluginNode) == pluginForeignKey && nameNode.Value == pluginName, nil
		}
	}

	for _, owner := range ts.pluginOwners {
		// find (or create)	plugins array of the 'owner'
		ownerPluginArray := yamlbasics.GetFieldValue(owner, "plugins")
		if ownerPluginArray == nil || ownerPluginArray.Kind != yaml.SequenceNode {
			// no plugins array yet, create an empty one and add it
			ownerPluginArray = yamlbasics.NewArray()
			yamlbasics.SetFieldValue(owner, "plugins", ownerPluginArray)
			if owner == ts.data {
				// we created the main plugin array, so set in our cache
				ts.pluginMain = ownerPluginArray
			}
		}

		// findPlugin the plugin in the owner
		var findPlugin yamlbasics.YamlArrayIterator
		if ownerPluginArray == ts.pluginMain {
			findPlugin = yamlbasics.Search(ownerPluginArray, mainSearcher)
		} else {
			findPlugin = yamlbasics.Search(ownerPluginArray, genericSearcher)
		}

		existingPlugin, idx, err := findPlugin()
		if err != nil {
			return err
		}

		if existingPlugin == nil {
			// plugin not found, add it
			_ = yamlbasics.Append(ownerPluginArray, yamlbasics.CopyNode(newPlugin))
		} else {
			// plugin found, overwrite it if overwrite is true
			if !overwrite {
				continue
			}

			// replace the existing plugin
			ownerPluginArray.Content[idx] = yamlbasics.CopyNode(newPlugin)
		}
	}
	return nil
}
