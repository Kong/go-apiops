/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/patch"
	"github.com/spf13/cobra"
)

// Executes the CLI command "patch"
func executePatch(cmd *cobra.Command, args []string) {
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

	var valuesPatch patch.DeckPatch
	{
		values, err := cmd.Flags().GetStringArray("value")
		if err != nil {
			log.Fatalf("failed to retrieve '--value' entries; %s", err)
		}
		valuesPatch.Values, valuesPatch.Remove, err = patch.ValidateValuesFlags(values)
		if err != nil {
			log.Fatalf("failed parsing '--value' entry; %s", err)
		}
	}

	{
		s, err := cmd.Flags().GetString("selector")
		if err != nil {
			log.Fatalf("failed to retrieve '--selector' entry; %s", err)
		}
		valuesPatch.SelectorSource = s
	}

	patchFiles := make([]patch.DeckPatchFile, 0)
	{
		for _, filename := range args {
			var patchfile patch.DeckPatchFile
			err := patchfile.ParseFile(filename)
			if err != nil {
				log.Fatalf("failed to parse '%s': %s", filename, err)
			}
			patchFiles = append(patchFiles, patchfile)
		}
	}

	trackInfo := deckformat.HistoryNewEntry("patch")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	if len(valuesPatch.Values) != 0 || len(valuesPatch.Remove) != 0 {
		trackInfo["selector"] = valuesPatch.SelectorSource
	}
	if len(valuesPatch.Values) != 0 {
		trackInfo["values"] = valuesPatch.Values
	}
	if len(valuesPatch.Remove) != 0 {
		trackInfo["remove"] = valuesPatch.Remove
	}
	if len(args) != 0 {
		trackInfo["patchfiles"] = args
	}

	// do the work; read/patch/write
	data := filebasics.MustDeserializeFile(inputFilename)
	deckformat.HistoryAppend(data, trackInfo) // add before patching, so patch can operate on it

	yamlNode := jsonbasics.ConvertToYamlNode(data)

	if (len(valuesPatch.Values) + len(valuesPatch.Remove)) > 0 {
		// apply selector + value flags
		err = valuesPatch.ApplyToNodes(yamlNode)
		if err != nil {
			log.Fatalf("Failed to apply command-line values; %s", err)
		}
	}

	if len(args) > 0 {
		// apply patch files
		for i, patchFile := range patchFiles {
			patchFile.MustApply(yamlNode, args[i])
		}
	}

	data = jsonbasics.ConvertToJSONobject(yamlNode)

	filebasics.MustWriteSerializedFile(outputFilename, data, asYaml)
}

//
//
// Define the CLI data for the patch command
//
//

var patchCmd = &cobra.Command{
	Use:   "patch [flags] [...patch-files]",
	Short: "Applies patches on top of a decK file",
	Long: `Applies patches on top of a decK file.

The input file will be read, the patches will be applied, and if successful, written
to the output file. The patches can be specified by a '--selector' and one or more
'--value' tags, or via patch-files.

When using '--selector' and '--values', the items will be selected by the 'selector' which is
a JSONpath query. From the array of nodes found, only the objects will be updated.
The 'values' will be applied on each of the JSONobjects returned by the 'selector'.

The value part must be a valid JSON snippet, so make sure to use single/double quotes
appropriately. If the value is empty, the field will be removed from the object.
Examples:
  --selector="$..services[*]" --value="read_timeout:10000"
  --selector="$..services[*]" --value='_comment:"comment injected by patching"'
  --selector="$..services[*]" --value='_ignore:["ignore1","ignore2"]'
  --selector="$..services[*]" --value='_ignore:' --value='_comment:'

The patchfiles have the following format (JSON or Yaml) and can contain multiple
patches that will be applied in order;

  { "_format_version": "1.0",
    "patches": [
      { "selector": "$..services[*]",
        "values": {
          "read_timeout": 10000,
          "_comment": "comment injected by patching"
        },
        "remove": [ "_ignore" ]
      }
    ]
  }
`,
	Run: executePatch,
}

func init() {
	rootCmd.AddCommand(patchCmd)
	patchCmd.Flags().StringP("state", "s", "-", "decK file to process. Use - to read from stdin")
	patchCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	patchCmd.Flags().StringP("format", "", outputFormatYaml, "output format: "+outputFormatJSON+" or "+outputFormatYaml)
	patchCmd.Flags().StringP("selector", "", "", "json-pointer identifying element to patch")
	patchCmd.Flags().StringArrayP("value", "", []string{}, "a value to set in the selected entry in "+
		"format <key:value> (can be specified more than once)")
	patchCmd.MarkFlagsRequiredTogether("selector", "value")
}
