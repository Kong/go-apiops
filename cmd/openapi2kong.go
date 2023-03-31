/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/kong/go-apiops/convertoas3"
	"github.com/kong/go-apiops/filebasics"
	"github.com/spf13/cobra"
)

// Executes the CLI command "openapi2kong"
func executeOpenapi2Kong(cmd *cobra.Command, _ []string) {
	inputFilename, err := cmd.Flags().GetString("spec")
	if err != nil {
		log.Fatalf("failed getting cli argument 'spec'; %s", err)
	}

	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		log.Fatalf("failed getting cli argument 'output-file'; %s", err)
	}

	docName, err := cmd.Flags().GetString("uuid-base")
	if err != nil {
		log.Fatalf("failed getting cli argument 'uuid-base'; %s", err)
	}

	var entityTags *[]string
	{
		tags, err := cmd.Flags().GetStringSlice("select-tag")
		if err != nil {
			log.Fatalf("failed getting cli argument 'select-tag'; %s", err)
		}
		entityTags = &tags
		if len(*entityTags) == 0 {
			entityTags = nil
		}
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

	options := convertoas3.O2kOptions{
		Tags:    entityTags,
		DocName: docName,
	}

	// do the work: read/convert/write
	result := convertoas3.MustConvert(filebasics.MustReadFile(inputFilename), options)
	filebasics.MustWriteSerializedFile(outputFilename, result, asYaml)
}

//
//
// Define the CLI data for the openapi2kong command
//
//

var openapi2kongCmd = &cobra.Command{
	Use:   "openapi2kong",
	Short: "Convert OpenAPI files to Kong's decK format",
	Long: `Convert OpenAPI files to Kong's decK format.

The example file has extensive annotations explaining the conversion
process, as well as all supported custom annotations (x-kong-... directives).
See: https://github.com/Kong/kced/blob/main/docs/learnservice_oas.yaml`,
	Run: executeOpenapi2Kong,
}

func init() {
	rootCmd.AddCommand(openapi2kongCmd)
	openapi2kongCmd.Flags().StringP("spec", "s", "-", "OpenAPI spec file to process. Use - to read from stdin")
	openapi2kongCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	openapi2kongCmd.Flags().StringP("format", "", outputFormatYaml, "output format: "+
		outputFormatJSON+" or "+outputFormatYaml)
	openapi2kongCmd.Flags().StringP("uuid-base", "", "",
		`the unique base-string for uuid-v5 generation of enity id's (if omitted
will use the root-level "x-kong-name" directive, or fall back to 'info.title')`)
	openapi2kongCmd.Flags().StringSlice("select-tag", nil,
		`select tags to apply to all entities (if omitted will use the "x-kong-tags"
directive from the file)`)
}
