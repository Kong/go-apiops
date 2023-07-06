/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/plugins"
	"github.com/spf13/cobra"
)

// Executes the CLI command "add-plugins"
func executeAddPlugins(cmd *cobra.Command, cfgFiles []string) error {
	verbosity, _ := cmd.Flags().GetInt("verbose")
	logbasics.Initialize(log.LstdFlags, verbosity)

	inputFilename, err := cmd.Flags().GetString("state")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'state'; %w", err)
	}

	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'output-file'; %w", err)
	}

	var outputFormat string
	{
		outputFormat, err = cmd.Flags().GetString("format")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'format'; %w", err)
		}
		outputFormat = strings.ToUpper(outputFormat)
	}

	var selectors []string
	{
		selectors, err = cmd.Flags().GetStringArray("selector")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'selector'; %w", err)
		}
	}

	var pluginConfigs []map[string]interface{}
	{
		strConfigs, err := cmd.Flags().GetStringArray("config")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'config'; %w", err)
		}
		for _, strConfig := range strConfigs {
			pluginConfig, err := filebasics.Deserialize([]byte(strConfig))
			if err != nil {
				return fmt.Errorf("failed to deserialize plugin config '%s'; %w", strConfig, err)
			}
			pluginConfigs = append(pluginConfigs, pluginConfig)
		}
	}

	var overwrite bool
	{
		overwrite, err = cmd.Flags().GetBool("overwrite")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'overwrite'; %w", err)
		}
	}

	if len(cfgFiles) > 0 {
		logbasics.Error(nil, "WARNING: 'config-files' are not yet implemented; ignoring")
	}

	// do the work: read/add-plugins/write
	data, err := filebasics.DeserializeFile(inputFilename)
	if err != nil {
		return fmt.Errorf("failed to read input file '%s'; %w", inputFilename, err)
	}

	plugger := plugins.Plugger{}
	plugger.SetData(data)
	err = plugger.SetSelectors(selectors)
	if err != nil {
		return fmt.Errorf("failed to set selectors; %w", err)
	}
	err = plugger.AddPlugins(pluginConfigs, overwrite)
	if err != nil {
		return fmt.Errorf("failed to add plugins; %w", err)
	}
	data = plugger.GetData()

	trackInfo := deckformat.HistoryNewEntry("add-plugins")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	trackInfo["overwrite"] = overwrite
	if len(pluginConfigs) > 0 {
		trackInfo["configs"] = pluginConfigs
	}
	if len(cfgFiles) > 0 {
		trackInfo["config-files"] = cfgFiles
	}
	trackInfo["selectors"] = selectors
	deckformat.HistoryAppend(data, trackInfo)

	return filebasics.WriteSerializedFile(outputFilename, data, outputFormat)
}

//
//
// Define the CLI data for the add-plugins command
//
//

var addPluginsCmd = &cobra.Command{
	Use:   "add-plugins [flags] [...plugin-files]",
	Short: "Adds plugins to objects in a decK file",
	Long: `Adds plugins to objects in a decK file.

The plugins are added to all objects that match the selector expressions. If no
selectors are given, they will be added to the top-level 'plugins' array.`,
	RunE: executeAddPlugins,
	Args: cobra.MinimumNArgs(0),
}

func init() {
	rootCmd.AddCommand(addPluginsCmd)
	addPluginsCmd.Flags().StringP("state", "s", "-", "decK file to process. Use - to read from stdin")
	addPluginsCmd.Flags().StringArray("selector", []string{},
		"JSON path expression to select plugin-owning objects to add plugins to,\n"+
			"defaults to the top-level (selector '$'). Repeat for multiple selectors.")
	addPluginsCmd.Flags().StringArray("config", []string{},
		"JSON snippet containing the plugin configuration to add. Repeat to add\n"+
			"multiple plugins.")
	addPluginsCmd.Flags().Bool("overwrite", false,
		"specifying this flag will overwrite plugins by the same name if they already\n"+
			"exist in an array. The default is to skip existing plugins.")
	addPluginsCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	addPluginsCmd.Flags().StringP("format", "", filebasics.OutputFormatYaml, "output format: "+
		filebasics.OutputFormatJSON+" or "+filebasics.OutputFormatYaml)
}
