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
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/namespace"
	"github.com/spf13/cobra"
)

// Executes the CLI command "namespace"
func executeNamespace(cmd *cobra.Command, _ []string) error {
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

	var matchPrefix string
	{
		matchPrefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			return fmt.Errorf("failed to retrieve '--prefix' value; %w", err)
		}
		_, err = namespace.CheckPrefix(matchPrefix)
		if err != nil {
			return err
		}
	}

	var namespaceStr string
	{
		namespaceStr, err = cmd.Flags().GetString("namespace")
		if err != nil {
			return fmt.Errorf("failed to retrieve '--namespace' value; %w", err)
		}
		_, err = namespace.CheckNamespace(namespaceStr)
		if err != nil {
			return err
		}
	}

	trackInfo := deckformat.HistoryNewEntry("namespace")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	trackInfo["prefix"] = matchPrefix
	trackInfo["namespace"] = namespaceStr

	// do the work; read/prefix/write
	data, err := filebasics.DeserializeFile(inputFilename)
	if err != nil {
		return err
	}

	yamlNode := jsonbasics.ConvertToYamlNode(data)
	err = namespace.Apply(yamlNode, matchPrefix, namespaceStr)
	if err != nil {
		log.Fatalf("failed to apply the namespace: %s", err)
	}
	data = jsonbasics.ConvertToJSONobject(yamlNode)

	deckformat.HistoryAppend(data, trackInfo)
	return filebasics.WriteSerializedFile(outputFilename, data, filebasics.OutputFormat(outputFormat))
}

//
//
// Define the CLI data for the namespace command
//
//

var namespaceCmd = &cobra.Command{
	Use:   "namespace [flags]",
	Short: "Namespaces API paths by prefixing it",
	Long: `Namespaces API paths by prefixing it.

By prefixing paths with a specific segment, colliding paths to services can be
namespaced to prevent the collisions. Eg. 2 API definitions that both expose a
'/list' path. By prefixing one with '/addressbook' and the other with '/cookbook'
the resulting paths '/addressbook/list' and '/cookbook/list' can be exposed without
colliding.

A "pre-function" plugin will be added to remove the prefix from the path before
the request is routed to the service. If the prefix is matching the 'service.path'
suffix, then that property is updated, and no plugin is injected.
`,
	Args: cobra.NoArgs,
	RunE: executeNamespace,
}

func init() {
	rootCmd.AddCommand(namespaceCmd)
	namespaceCmd.Flags().StringP("state", "s", "-", "decK file to process. Use - to read from stdin.")
	namespaceCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout.")
	namespaceCmd.Flags().StringP("format", "", string(filebasics.OutputFormatYaml), "output format: "+
		string(filebasics.OutputFormatJSON)+" or "+string(filebasics.OutputFormatYaml))
	namespaceCmd.Flags().StringP("prefix", "", "", "the existing path-prefix to match. Only matching paths "+
		"will be namespaced (plain or regex based)")
	namespaceCmd.Flags().StringP("namespace", "", "", "the namespace to apply")
}
