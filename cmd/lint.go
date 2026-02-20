package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/lint"
	"github.com/kong/go-apiops/logbasics"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// executeLint executes the CLI command "lint"
func executeLint(cmd *cobra.Command, args []string) error {
	verbosity, _ := cmd.Flags().GetInt("verbose")
	logbasics.Initialize(log.LstdFlags, verbosity)

	if len(args) == 0 {
		return fmt.Errorf("expected a ruleset file as argument")
	}
	rulesetFilename := args[0]

	stateFilename, err := cmd.Flags().GetString("state")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'state'; %w", err)
	}

	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'output-file'; %w", err)
	}

	failSeverityStr, err := cmd.Flags().GetString("fail-severity")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'fail-severity'; %w", err)
	}

	failSeverity, err := lint.ParseSeverity(failSeverityStr)
	if err != nil {
		return fmt.Errorf("invalid fail-severity: %w", err)
	}

	displayOnlyFailures, err := cmd.Flags().GetBool("display-only-failures")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'display-only-failures'; %w", err)
	}

	outputFormat, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'format'; %w", err)
	}

	// Read the ruleset file
	rulesetData, err := filebasics.ReadFile(rulesetFilename)
	if err != nil {
		return fmt.Errorf("failed to read ruleset file %q: %w", rulesetFilename, err)
	}

	// Read the input document
	documentData, err := filebasics.ReadFile(stateFilename)
	if err != nil {
		return fmt.Errorf("failed to read input file %q: %w", stateFilename, err)
	}

	// Run the linter
	results, err := lint.Lint(rulesetData, documentData, stateFilename)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	// Filter results if display-only-failures is set
	if displayOnlyFailures {
		filtered := make([]lint.LintResult, 0, len(results))
		for _, r := range results {
			if r.Severity <= failSeverity {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Format output
	var output []byte
	switch outputFormat {
	case "json":
		output, err = json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results to JSON: %w", err)
		}
		output = append(output, '\n')
	case "yaml":
		output, err = yaml.Marshal(results)
		if err != nil {
			return fmt.Errorf("failed to marshal results to YAML: %w", err)
		}
	default: // "plain"
		if len(results) > 0 {
			failCount := lint.CountBySeverity(results, failSeverity)
			output = []byte(fmt.Sprintf("Linting Violations: %d\nFailures: %d\n\n%s",
				len(results), failCount, lint.FormatResults(results)))
		}
	}

	// Write output
	if len(output) > 0 {
		if err := filebasics.WriteFile(outputFilename, output); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	// Exit with error code if there are failures at or above the fail-severity
	failCount := lint.CountBySeverity(results, failSeverity)
	if failCount > 0 {
		os.Exit(1)
	}

	return nil
}

//
// Define the CLI data for the lint command
//

var lintCmd = &cobra.Command{
	Use:   "lint [flags] ruleset-file",
	Short: "Validates a file against a Spectral-compatible ruleset",
	Long: `Validates a file against a Spectral-compatible ruleset.

The lint command is a validation tool to analyze declarative configuration files for
errors or undesirable configurations. It is compatible with Spectral rulesets and can
operate on either JSON or YAML format files including OpenAPI specifications or
decK configuration files.

The command is invoked by providing a ruleset file as an argument which is evaluated
against an input file.`,
	RunE: executeLint,
	Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(lintCmd)
	lintCmd.Flags().StringP("state", "s", "-",
		"input file to lint. Use - to read from stdin")
	lintCmd.Flags().StringP("output-file", "o", "-",
		"output file to write to. Use - to write to stdout")
	lintCmd.Flags().StringP("fail-severity", "F", "error",
		`results of this level or above will trigger a failure exit code
[choices: "error", "warn", "info", "hint"]`)
	lintCmd.Flags().BoolP("display-only-failures", "D", false,
		"only output results equal to or greater than --fail-severity")
	lintCmd.Flags().StringP("format", "", "plain",
		`output format [choices: "plain", "json", "yaml"]`)
}
