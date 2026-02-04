package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/openapi2mcp"
	"github.com/spf13/cobra"
)

// Executes the CLI command "openapi2mcp"
func executeOpenapi2Mcp(cmd *cobra.Command, _ []string) error {
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

	var mode string
	{
		mode, err = cmd.Flags().GetString("mode")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'mode'; %w", err)
		}
		// Validate mode
		if mode != "" && mode != openapi2mcp.ModeConversion && mode != openapi2mcp.ModeConversionListener {
			return fmt.Errorf("invalid mode '%s': must be '%s' or '%s'",
				mode, openapi2mcp.ModeConversion, openapi2mcp.ModeConversionListener)
		}
	}

	var pathPrefix string
	{
		pathPrefix, err = cmd.Flags().GetString("path-prefix")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'path-prefix'; %w", err)
		}
	}

	var includeDirectRoute bool
	{
		includeDirectRoute, err = cmd.Flags().GetBool("include-direct-route")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'include-direct-route'; %w", err)
		}
	}

	var noID bool
	{
		noID, err = cmd.Flags().GetBool("no-id")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'no-id'; %w", err)
		}
	}

	options := openapi2mcp.O2MOptions{
		Tags:               entityTags,
		DocName:            docName,
		Mode:               mode,
		PathPrefix:         pathPrefix,
		IncludeDirectRoute: includeDirectRoute,
		SkipID:             noID,
	}

	trackInfo := deckformat.HistoryNewEntry("openapi2mcp")
	trackInfo["input"] = inputFilename
	trackInfo["output"] = outputFilename
	trackInfo["uuid-base"] = docName
	trackInfo["mode"] = mode

	// do the work: read/convert/write
	content, err := filebasics.ReadFile(inputFilename)
	if err != nil {
		return err
	}
	result, err := openapi2mcp.Convert(content, options)
	if err != nil {
		return fmt.Errorf("failed converting OpenAPI spec '%s'; %w", inputFilename, err)
	}
	deckformat.HistoryAppend(result, trackInfo)
	return filebasics.WriteSerializedFile(outputFilename, result, filebasics.OutputFormat(outputFormat))
}

//
//
// Define the CLI data for the openapi2mcp command
//
//

var openapi2mcpCmd = &cobra.Command{
	Use:   "openapi2mcp",
	Short: "Convert OpenAPI files to Kong's decK format with MCP (Model Context Protocol) configuration",
	Long: `Convert OpenAPI files to Kong's decK format with ai-mcp-proxy plugin configuration.

This command generates a Kong service with an MCP route that includes the ai-mcp-proxy
plugin configured with tools derived from the OpenAPI specification operations.

Each OpenAPI operation is mapped to an MCP tool definition:
  - operationId -> tool name (kebab-case normalized)
  - summary/description -> tool description
  - parameters -> tool parameters array
  - requestBody -> tool request_body

Supported x-kong extensions:
  - x-kong-name: Custom entity naming
  - x-kong-tags: Tags for all entities
  - x-kong-service-defaults: Service entity defaults
  - x-kong-route-defaults: Route entity defaults
  - x-kong-upstream-defaults: Upstream entity defaults
  - x-kong-plugin-*: Additional plugins

MCP-specific extensions:
  - x-kong-mcp-tool-name: Override generated tool name
  - x-kong-mcp-tool-description: Override tool description
  - x-kong-mcp-exclude: Exclude operation from tool generation (boolean)
  - x-kong-mcp-proxy: Override ai-mcp-proxy plugin config at document level`,
	RunE: executeOpenapi2Mcp,
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(openapi2mcpCmd)
	openapi2mcpCmd.Flags().StringP("spec", "s", "-", "OpenAPI spec file to process. Use - to read from stdin")
	openapi2mcpCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	openapi2mcpCmd.Flags().StringP("format", "", string(filebasics.OutputFormatYaml), "output format: "+
		string(filebasics.OutputFormatJSON)+" or "+string(filebasics.OutputFormatYaml))
	openapi2mcpCmd.Flags().StringP("uuid-base", "", "",
		`the unique base-string for uuid-v5 generation of entity id's (if omitted
will use the root-level "x-kong-name" directive, or fall back to 'info.title')`)
	openapi2mcpCmd.Flags().StringSlice("select-tag", nil,
		`select tags to apply to all entities (if omitted will use the "x-kong-tags"
directive from the file)`)
	openapi2mcpCmd.Flags().StringP("mode", "m", openapi2mcp.ModeConversionListener,
		`ai-mcp-proxy mode: "conversion" (client mode) or "conversion-listener" (server mode)`)
	openapi2mcpCmd.Flags().StringP("path-prefix", "p", "",
		`custom path prefix for the MCP route (default: /{service-name}-mcp)`)
	openapi2mcpCmd.Flags().BoolP("include-direct-route", "", false,
		`also generate non-MCP routes for direct API access`)
	openapi2mcpCmd.Flags().BoolP("no-id", "", false,
		`do not generate UUIDs for entities`)
}
