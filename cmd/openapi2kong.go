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
	"github.com/kong/go-apiops/openapi2kong"
	"github.com/spf13/cobra"
)

// Executes the CLI command "openapi2kong"
func executeOpenapi2Kong(cmd *cobra.Command, _ []string) error {
	verbosity, _ := cmd.Flags().GetInt("verbose")
	logbasics.Initialize(log.LstdFlags, verbosity)

	inputFilename, err := cmd.Flags().GetString("spec")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'spec'; %w", err)
	}

	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'output-file'; %w", err)
	}

	docName, err := cmd.Flags().GetString("uuid-base")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'uuid-base'; %w", err)
	}

	var entityTags []string
	{
		tags, err := cmd.Flags().GetStringSlice("select-tag")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'select-tag'; %w", err)
		}
		entityTags = tags
		if len(entityTags) == 0 {
			entityTags = nil
		}
	}

	var outputFormat string
	{
		outputFormat, err = cmd.Flags().GetString("format")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'format'; %w", err)
		}
		outputFormat = strings.ToUpper(outputFormat)
	}

	var generateSecurity bool
	{
		generateSecurity, err = cmd.Flags().GetBool("generate-security")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'generate-security'; %w", err)
		}
	}

	var ignoreSecurityErrors bool
	{
		ignoreSecurityErrors, err = cmd.Flags().GetBool("ignore-security-errors")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'ignore-security-errors'; %w", err)
		}
	}

	options := openapi2kong.O2kOptions{
		Tags:                 entityTags,
		DocName:              docName,
		OIDC:                 generateSecurity,
		IgnoreSecurityErrors: ignoreSecurityErrors,
	}

	trackInfo := deckformat.HistoryNewEntry("openapi2kong")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	trackInfo["uuid-base"] = docName

	// do the work: read/convert/write
	content, err := filebasics.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	result, err := openapi2kong.Convert(content, options)
	if err != nil {
		return fmt.Errorf("failed converting OpenAPI spec '%s'; %w", inputFilename, err)
	}
	deckformat.HistoryAppend(result, trackInfo)
	return filebasics.WriteSerializedFile(outputFilename, result, filebasics.OutputFormat(outputFormat))
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
See: https://github.com/Kong/go-apiops/blob/main/docs/learnservice_oas.yaml`,
	RunE: executeOpenapi2Kong,
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(openapi2kongCmd)
	openapi2kongCmd.Flags().StringP("spec", "s", "-", "OpenAPI spec file to process. Use - to read from stdin")
	openapi2kongCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	openapi2kongCmd.Flags().StringP("format", "", string(filebasics.OutputFormatYaml), "output format: "+
		string(filebasics.OutputFormatJSON)+" or "+string(filebasics.OutputFormatYaml))
	openapi2kongCmd.Flags().StringP("uuid-base", "", "",
		`the unique base-string for uuid-v5 generation of enity id's (if omitted
will use the root-level "x-kong-name" directive, or fall back to 'info.title')`)
	openapi2kongCmd.Flags().StringSlice("select-tag", nil,
		`select tags to apply to all entities (if omitted will use the "x-kong-tags"
directive from the file)`)
	openapi2kongCmd.Flags().BoolP("generate-security", "", false, "generate OpenIDConnect plugins from the "+
		"security directives")
	openapi2kongCmd.Flags().BoolP("ignore-security-errors", "", false, "ignore errors for unsupported security schemes")
}
