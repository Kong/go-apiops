/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"strings"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/namespace"
	"github.com/kong/go-apiops/patch"
	"github.com/spf13/cobra"
)

// Executes the CLI command "namespace"
func executeNamespace(cmd *cobra.Command, _ []string) {
	inputFilename, err := cmd.Flags().GetString("state")
	if err != nil {
		log.Fatalf("failed getting cli argument 'state'; %s", err)
	}

	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		log.Fatalf("failed getting cli argument 'output-file'; %s", err)
	}

	var asYaml bool
	{
		outputFormat, err := cmd.Flags().GetString("format")
		if err != nil {
			log.Fatalf("failed getting cli argument 'format'; %s", err)
		}
		if outputFormat == outputFormatYaml {
			asYaml = true
		} else if outputFormat == outputFormatJSON {
			asYaml = false
		} else {
			log.Fatalf("expected '--format' to be either '"+outputFormatYaml+
				"' or '"+outputFormatJSON+"', got: '%s'", outputFormat)
		}
	}

	var matchPrefix string
	{
		matchPrefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			log.Fatalf("failed to retrieve '--prefix' value; %s", err)
		}
		if !(strings.HasPrefix(matchPrefix, "/") || strings.HasPrefix(matchPrefix, "~/")) {
			log.Fatalf("invalid prefix; the prefix MUST start with '/', got: '%s'", matchPrefix)
		}
	}

	var namespaceStr string
	{
		namespace, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Fatalf("failed to retrieve '--namespace' value; %s", err)
		}
		if !strings.HasPrefix(namespace, "/") {
			log.Fatalf("invalid namespace; the namepsace MUST start with '/', got: '%s'", namespace)
		}
	}

	trackInfo := deckformat.HistoryNewEntry("namespace")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	trackInfo["prefix"] = matchPrefix
	trackInfo["namespace"] = namespaceStr

	// do the work; read/patch/write
	data := filebasics.MustDeserializeFile(inputFilename)
	deckformat.HistoryAppend(data, trackInfo) // add before patching, so patch can operate on it

	yamlNode := patch.ConvertToYamlNode(data)

	err = namespace.Apply(yamlNode, matchPrefix, namespaceStr)
	if err != nil {
		log.Fatalf("failed to apply the namespace: %s", err)
	}

	data = patch.ConvertToJSONobject(yamlNode)

	filebasics.MustWriteSerializedFile(outputFilename, data, asYaml)
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

'strip-path' settings will be added to the route-config to ensure it is stripped
again from the path before sending it upstream.

NOTE: all paths within a route must match for the route to be updated.
`,
	Args: cobra.NoArgs,
	Run:  executeNamespace,
}

func init() {
	rootCmd.AddCommand(namespaceCmd)
	namespaceCmd.Flags().StringP("state", "s", "-", "decK file to process. Use - to read from stdin.")
	namespaceCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout.")
	namespaceCmd.Flags().StringP("format", "", outputFormatYaml, "output format: "+outputFormatJSON+
		" or "+outputFormatYaml)
	namespaceCmd.Flags().StringP("prefix", "", "/", "the existing path-prefix to match. Only matching paths "+
		"will be namespaced (plain or regex based)")
	namespaceCmd.Flags().StringP("namespace", "", "", "the namespace to apply")
}
