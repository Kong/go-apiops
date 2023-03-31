/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/kong/go-apiops/filebasics"
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

	var valuesMap map[string]interface{}
	{
		values, err := cmd.Flags().GetStringArray("value")
		if err != nil {
			log.Fatalf("failed to retrieve '--value' entries; %s", err)
		}
		valuesMap, err = patch.ValidateValuesFlags(values)
		if err != nil {
			log.Fatalf("failed parsing '--value' entry; %s", err)
		}
	}

	var selector string
	{
		s, err := cmd.Flags().GetString("selector")
		if err != nil {
			log.Fatalf("failed to retrieve '--selector' entry; %s", err)
		}
		selector = s
	}

	if len(args) == 0 && len(valuesMap) == 0 {
		log.Fatal("expected at least one patch-file or '--value' flag")
	} else if len(args) > 0 && len(valuesMap) > 0 {
		log.Fatal("patch-files and '--value' flags are mutually exclusive")
	}

	// do the work; read/patch/write
	if len(args) == 0 {
		// apply selector + value flags
		data := filebasics.MustDeserializeFile(inputFilename)
		updated := patch.MustApplyValues(data, selector, valuesMap)
		filebasics.MustWriteSerializedFile(outputFilename, updated, asYaml)
	} else {
		// apply patch files
		data := filebasics.MustDeserializeFile(inputFilename)
		updated := patch.MustApplyPatches(data, args)
		filebasics.MustWriteSerializedFile(outputFilename, updated, asYaml)
	}
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
to the output file.

The patches can be specified by a '--selector' and one or more '--value' tags, or via
patch-files.`,
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
