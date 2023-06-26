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
	"github.com/kong/go-apiops/tags"
	"github.com/spf13/cobra"
)

// Executes the CLI command "remove-tags"
func executeRemoveTags(cmd *cobra.Command, tagsToRemove []string) error {
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

	var keepEmptyArrays bool
	{
		keepEmptyArrays, err = cmd.Flags().GetBool("keep-empty")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'keep-array'; %w", err)
		}
	}

	var keepOnlyTags bool
	{
		keepOnlyTags, err = cmd.Flags().GetBool("keep-only")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'keep-only'; %w", err)
		}
	}

	if !keepOnlyTags && len(tagsToRemove) == 0 {
		return fmt.Errorf("no tags to remove")
	}

	// do the work: read/remove-tags/write
	data, err := filebasics.DeserializeFile(inputFilename)
	if err != nil {
		return fmt.Errorf("failed to read input file '%s'; %w", inputFilename, err)
	}

	tagger := tags.Tagger{}
	tagger.SetData(data)
	err = tagger.SetSelectors(selectors)
	if err != nil {
		return fmt.Errorf("failed to set selectors; %w", err)
	}
	if keepOnlyTags {
		err = tagger.RemoveUnknownTags(tagsToRemove, !keepEmptyArrays)
	} else {
		err = tagger.RemoveTags(tagsToRemove, !keepEmptyArrays)
	}
	if err != nil {
		return fmt.Errorf("failed to remove tags; %w", err)
	}
	data = tagger.GetData()

	trackInfo := deckformat.HistoryNewEntry("remove-tags")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	trackInfo["tags"] = tagsToRemove
	trackInfo["keep-empty"] = keepEmptyArrays
	trackInfo["selectors"] = selectors
	deckformat.HistoryAppend(data, trackInfo)

	return filebasics.WriteSerializedFile(outputFilename, data, outputFormat)
}

//
//
// Define the CLI data for the remove-tags command
//
//

var RemoveTagsCmd = &cobra.Command{
	Use:   "remove-tags [flags] tag [...tag]",
	Short: "Removes tags from objects in a decK file",
	Long: `Removes tags from objects in a decK file.

The listed tags are removed from all objects that match the selector expressions.
If no selectors are given, all Kong entities will be selected.`,
	RunE: executeRemoveTags,
}

func init() {
	rootCmd.AddCommand(RemoveTagsCmd)
	RemoveTagsCmd.Flags().Bool("keep-empty", false, "keep empty tag-arrays in output")
	RemoveTagsCmd.Flags().Bool("keep-only", false, "setting this flag will remove all tags except the ones listed\n"+
		"(if none are listed, all tags will be removed)")
	RemoveTagsCmd.Flags().StringP("state", "s", "-", "decK file to process. Use - to read from stdin")
	RemoveTagsCmd.Flags().StringArray("selector", []string{}, "JSON path expression to select "+
		"objects to remove tags from,\ndefaults to all Kong entities (repeat for multiple selectors)")
	RemoveTagsCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	RemoveTagsCmd.Flags().StringP("format", "", filebasics.OutputFormatYaml, "output format: "+
		filebasics.OutputFormatJSON+" or "+filebasics.OutputFormatYaml)
}
