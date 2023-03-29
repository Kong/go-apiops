/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/merge"
	"github.com/spf13/cobra"
)

// Executes the CLI command "openapi2kong"
func executeMerge(cmd *cobra.Command, args []string) {
	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		log.Fatalf(fmt.Sprintf("failed getting cli argument 'output-file'; %%w"), err)
	}

	var asYaml bool
	{
		outputFormat, err := cmd.Flags().GetString("format")
		if err != nil {
			log.Fatalf(fmt.Sprintf("failed getting cli argument 'format'; %%w"), err)
		}
		if outputFormat == "yaml" {
			asYaml = true
		} else if outputFormat == "json" {
			asYaml = false
		} else {
			log.Fatalf("expected '--format' to be either 'yaml' or 'json', got: '%s'", outputFormat)
		}
	}

	// do the work: read/merge
	filebasics.MustWriteSerializedFile(outputFilename, merge.MustFiles(args), asYaml)
}

//
//
// Define the CLI data for the openapi2kong command
//
//

var mergeCmd = &cobra.Command{
	Use:   "merge [flags] filename [...filename]",
	Short: "Merges multiple decK files into one",
	Long: `Merges multiple decK files into one.

The files can be either json or yaml format. Will merge all top-level arrays by simply
concatenating them. Any other keys will be copied. The files will be processed in order
of the '_format_version' field in the file (an omitted version defaults to "0.0"). An error
will be returned if files are incompatible.
There are no checks on duplicates, etc... garbage-in-garbage-out.`,
	Run:  executeMerge,
	Args: cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	mergeCmd.Flags().StringP("format", "", "yaml", "output format: json or yaml")
}
