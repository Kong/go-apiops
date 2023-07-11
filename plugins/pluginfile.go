package plugins

import (
	"fmt"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"gopkg.in/yaml.v3"
)

//
//
//  A single patch which can be repeated in a file
//
//

// AddPluginPatch represents a single-patch that adds plugins to a set of objects.
type AddPluginPatch struct {
	Selectors []string                 // JSONpath selectors
	Plugins   []map[string]interface{} // plugin objects to add
	Overwrite bool                     // overwrite existing plugins
}

// Parse parses a patch. It will return an error if the patch is invalid.
// 'source' will be used to format the error in case of an error.
func (patch *AddPluginPatch) Parse(patchData map[string]interface{}, source string) error {
	var err error
	patch.Selectors, err = jsonbasics.GetStringArrayField(patchData, "selectors")
	if err != nil {
		return fmt.Errorf("%s.selectors: is not an array; %w", source, err)
	}

	// try and compile the json paths
	var tester Plugger
	err = tester.SetSelectors(patch.Selectors)
	if err != nil {
		return fmt.Errorf("%s.selectors: %w", source, err)
	}

	patch.Plugins, err = jsonbasics.GetObjectArrayField(patchData, "add-plugins")
	if err != nil {
		return fmt.Errorf("%s.add-plugins is not an array; %w", source, err)
	}

	patch.Overwrite = false
	if patchData["overwrite"] != nil {
		patch.Overwrite, err = jsonbasics.GetBoolField(patchData, "overwrite")
		if err != nil {
			return fmt.Errorf("%s.overwrite: %w", source, err)
		}
	}

	return nil
}

// Apply applies the patch to the yaml.Node given.
func (patch *AddPluginPatch) Apply(yamlData *yaml.Node) error {
	var plugger Plugger
	err := plugger.SetSelectors(patch.Selectors)
	if err != nil {
		return err
	}

	plugger.SetYamlData(yamlData)

	return plugger.AddPlugins(patch.Plugins, patch.Overwrite)
}

//
//
//  A file which can hold multiple AddPluginPatches
//
//

// DeckPluginFile represents a list of Add-Plugin patches.
type DeckPluginFile struct {
	VersionMajor int // 0 if not present
	VersionMinor int // 0 if not present
	Plugins      []AddPluginPatch
}

// ParseFile parses a pluginfile. Any non-object in the 'patches' array will be
// ignored. If the array doesn't exist, it returns an empty array.
func (pluginFile *DeckPluginFile) ParseFile(filename string) error {
	data, err := filebasics.DeserializeFile(filename)
	if err != nil {
		return err
	}

	if data[deckformat.VersionKey] != nil {
		logbasics.Debug("parsed plugin file", "file", filename, "version", data[deckformat.VersionKey])
		pluginFile.VersionMajor, pluginFile.VersionMinor, err = deckformat.ParseFormatVersion(data)
		if err != nil {
			return fmt.Errorf("%s: has an invalid "+deckformat.VersionKey+" specified; %w", filename, err)
		}
	} else {
		logbasics.Debug("parsed unversioned plugin-file", "file", filename)
	}

	patchesRead, err := jsonbasics.GetObjectArrayField(data, "plugins")
	if err != nil {
		return fmt.Errorf("%s: field 'plugins' is not an array; %w", filename, err)
	}

	for i, patch := range patchesRead {
		var addPluginPatch AddPluginPatch
		err := addPluginPatch.Parse(patch, fmt.Sprintf("%s: plugins[%d]", filename, i))
		if err != nil {
			return err
		}
		pluginFile.Plugins = append(pluginFile.Plugins, addPluginPatch)
	}

	return nil
}

// Apply applies the set of patches on the yaml.Node given.
func (pluginFile *DeckPluginFile) Apply(yamlData *yaml.Node) error {
	for i, patch := range pluginFile.Plugins {
		err := patch.Apply(yamlData)
		if err != nil {
			return fmt.Errorf("failed to apply add-plugin patch %d; %w", i, err)
		}
	}
	return nil
}
