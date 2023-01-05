/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/Kong/fw/convert"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var openapi2kongCmd = &cobra.Command{
	Use:   "openapi2kong",
	Short: "Convert OpenAPI files to Kong's decK format",
	Long:  `Convert OpenAPI files to Kong's decK format`,
	Run: func(cmd *cobra.Command, args []string) {
		inputFilename, _ := cmd.Flags().GetString("input")

		input, err := os.ReadFile(inputFilename)
		if err != nil {
			fmt.Printf("%v", err)
			return
		}

		deckContent, err := convert.ConvertOas3(&input, convert.O2kOptions{
			Tags: &[]string{"OAS3_import"},
		})

		if err != nil {
			fmt.Printf("%v", err)
		} else {
			outputFilename, _ := cmd.Flags().GetString("output")
			YAMLOut, _ := yaml.Marshal(deckContent)
			err = os.WriteFile(outputFilename, YAMLOut, 0666)
			if err != nil {
				fmt.Printf("%v", err)
				return
			}
			fmt.Printf("Wrote %s\n", outputFilename)
		}
	},
}

func init() {
	rootCmd.AddCommand(openapi2kongCmd)
	openapi2kongCmd.Flags().StringP("input", "i", "", "The input file to process")
	openapi2kongCmd.Flags().StringP("output", "o", "", "The output file to write")
	openapi2kongCmd.MarkFlagRequired("input")
	openapi2kongCmd.MarkFlagRequired("output")
}
