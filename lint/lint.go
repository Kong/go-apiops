package lint

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kong/go-apiops/logbasics"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"go.yaml.in/yaml/v4"
	sigsyaml "sigs.k8s.io/yaml"
)

// Lint evaluates a ruleset against an input document and returns any violations.
// The rulesetData should be the raw bytes of a Spectral-compatible ruleset (YAML or JSON).
// The documentData should be the raw bytes of the document to lint (YAML or JSON).
// The source parameter is an optional filename used for error reporting.
func Lint(rulesetData []byte, documentData []byte, source string) ([]LintResult, error) {
	// Parse the ruleset
	ruleset, err := ParseRuleset(rulesetData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ruleset: %w", err)
	}

	return LintWithRuleset(ruleset, documentData, source)
}

// LintWithRuleset evaluates a pre-parsed ruleset against an input document and returns violations.
// The documentData should be the raw bytes of the document to lint (YAML or JSON).
// The source parameter is an optional filename used for error reporting.
func LintWithRuleset(ruleset *Ruleset, documentData []byte, source string) ([]LintResult, error) {
	// Parse the document as a yaml.Node tree (for line/column info)
	var yamlDoc yaml.Node
	if err := yaml.Unmarshal(documentData, &yamlDoc); err != nil {
		return nil, fmt.Errorf("failed to parse input document as YAML: %w", err)
	}

	// Get the root content node (unwrap document node)
	rootNode := &yamlDoc
	if rootNode.Kind == yaml.DocumentNode && len(rootNode.Content) > 0 {
		rootNode = rootNode.Content[0]
	}

	// Also parse to interface{} for function evaluation
	var docInterface interface{}
	if err := sigsyaml.Unmarshal(documentData, &docInterface); err != nil {
		return nil, fmt.Errorf("failed to parse input document: %w", err)
	}

	// Sort rule names for deterministic output
	ruleNames := make([]string, 0, len(ruleset.Rules))
	for name := range ruleset.Rules {
		ruleNames = append(ruleNames, name)
	}
	sort.Strings(ruleNames)

	var allResults []LintResult

	for _, ruleName := range ruleNames {
		rule := ruleset.Rules[ruleName]
		logbasics.Debug("evaluating rule", "rule", ruleName, "given", rule.Given)

		results, err := evaluateRule(ruleName, rule, rootNode, docInterface, source)
		if err != nil {
			logbasics.Debug("error evaluating rule", "rule", ruleName, "error", err)
			continue
		}

		allResults = append(allResults, results...)
	}

	SortResults(allResults)
	return allResults, nil
}

// evaluateRule evaluates a single rule against the document.
func evaluateRule(
	ruleName string,
	rule Rule,
	rootNode *yaml.Node,
	docInterface interface{},
	source string,
) ([]LintResult, error) {
	var allResults []LintResult

	// Evaluate each "given" JSONPath expression
	for _, givenPath := range rule.Given {
		// Compile the JSONPath
		jp, err := jsonpath.NewPath(givenPath)
		if err != nil {
			return nil, fmt.Errorf("invalid JSONPath %q: %w", givenPath, err)
		}

		// Query the yaml.Node tree for line/col info
		yamlMatches := jp.Query(rootNode)
		logbasics.Debug("JSONPath query results", "path", givenPath, "#matches", len(yamlMatches))

		// Also query the interface{} version for function evaluation
		interfaceNode := toYamlNodeFromInterface(docInterface)
		interfaceMatches := jp.Query(interfaceNode)

		// Process each match
		for i, yamlMatch := range yamlMatches {
			if yamlMatch == nil {
				continue
			}

			// Get the corresponding interface{} value for this match
			var matchValue interface{}
			if i < len(interfaceMatches) && interfaceMatches[i] != nil {
				matchValue = yamlNodeToInterface(interfaceMatches[i])
			}

			// Apply each "then" entry
			for _, then := range rule.Then {
				results := applyThenEntry(ruleName, rule, then, yamlMatch, matchValue, docInterface, source)
				allResults = append(allResults, results...)
			}
		}

		// If no matches and we have "then" entries, this might be expected
		// (e.g., checking that something should be undefined)
		if len(yamlMatches) == 0 {
			// For each then entry, if the function handles nil values (like "defined"),
			// we should still call it
			for _, then := range rule.Then {
				if then.Function == "defined" || then.Function == "truthy" {
					fn, err := GetCoreFunction(then.Function)
					if err != nil {
						continue
					}
					messages := fn(nil, then.FunctionOptions)
					for _, msg := range messages {
						description := rule.Description
						if rule.Message != "" {
							description = rule.Message
						}
						allResults = append(allResults, LintResult{
							Message:  formatMessage(description, msg),
							Path:     givenPath,
							Severity: rule.Severity,
							Line:     0,
							Column:   0,
							RuleName: ruleName,
							Source:   source,
						})
					}
				}
			}
		}
	}

	return allResults, nil
}

// applyThenEntry applies a single "then" entry to a matched node.
func applyThenEntry(
	ruleName string,
	rule Rule,
	then ThenEntry,
	yamlMatch *yaml.Node,
	matchValue interface{},
	docInterface interface{},
	source string,
) []LintResult {
	fn, err := GetCoreFunction(then.Function)
	if err != nil {
		return []LintResult{{
			Message:  err.Error(),
			Path:     "",
			Severity: SeverityError,
			RuleName: ruleName,
			Source:   source,
		}}
	}

	// If a field is specified, drill into it
	targetVal := matchValue
	targetNode := yamlMatch
	fieldPath := ""

	if then.Field != "" {
		fieldPath = "." + then.Field
		targetVal = getFieldFromInterface(matchValue, then.Field)
		targetNode = getFieldFromYamlNode(yamlMatch, then.Field)
	}

	// For unreferencedReusableObject, inject the full document into options
	opts := then.FunctionOptions
	if then.Function == "unreferencedReusableObject" {
		if opts == nil {
			opts = make(map[string]interface{})
		} else {
			// Copy to avoid mutating the original
			optsCopy := make(map[string]interface{}, len(opts)+1)
			for k, v := range opts {
				optsCopy[k] = v
			}
			opts = optsCopy
		}
		opts["__document__"] = docInterface
	}

	// Call the core function
	messages := fn(targetVal, opts)

	// Build results
	var results []LintResult
	for _, msg := range messages {
		line, col := getNodePosition(targetNode)

		description := rule.Description
		if rule.Message != "" {
			description = rule.Message
		}

		results = append(results, LintResult{
			Message:  formatMessage(description, msg),
			Path:     fieldPath,
			Severity: rule.Severity,
			Line:     line,
			Column:   col,
			RuleName: ruleName,
			Source:   source,
		})
	}

	return results
}

// formatMessage creates the violation message combining the rule description and function message.
func formatMessage(description, functionMsg string) string {
	if description == "" {
		return functionMsg
	}
	return description + ": " + functionMsg
}

// getFieldFromInterface retrieves a field value from a map[string]interface{}.
// Returns nil if the value is not a map or the field doesn't exist.
func getFieldFromInterface(val interface{}, field string) interface{} {
	if val == nil {
		return nil
	}
	obj, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}
	result, exists := obj[field]
	if !exists {
		return nil
	}
	return result
}

// getFieldFromYamlNode retrieves a field's value node from a yaml mapping node.
// Returns nil if the node is not a mapping or the field doesn't exist.
func getFieldFromYamlNode(node *yaml.Node, field string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}

// getNodePosition returns the line and column of a yaml.Node.
// Returns (0, 0) if the node is nil.
func getNodePosition(node *yaml.Node) (int, int) {
	if node == nil {
		return 0, 0
	}
	return node.Line, node.Column
}

// toYamlNodeFromInterface converts an interface{} value to a *yaml.Node
// by marshaling and unmarshaling through YAML.
func toYamlNodeFromInterface(data interface{}) *yaml.Node {
	encoded, err := yaml.Marshal(data)
	if err != nil {
		return &yaml.Node{}
	}
	var node yaml.Node
	if err := yaml.Unmarshal(encoded, &node); err != nil {
		return &yaml.Node{}
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return &node
}

// yamlNodeToInterface converts a *yaml.Node to an interface{} value.
func yamlNodeToInterface(node *yaml.Node) interface{} {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return yamlNodeToInterface(node.Content[0])
		}
		return nil

	case yaml.MappingNode:
		result := make(map[string]interface{})
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i].Value
			result[key] = yamlNodeToInterface(node.Content[i+1])
		}
		return result

	case yaml.SequenceNode:
		result := make([]interface{}, len(node.Content))
		for i, child := range node.Content {
			result[i] = yamlNodeToInterface(child)
		}
		return result

	case yaml.ScalarNode:
		return parseScalar(node)

	case yaml.AliasNode:
		return yamlNodeToInterface(node.Alias)

	default:
		return node.Value
	}
}

// parseScalar converts a yaml scalar node to the appropriate Go type.
func parseScalar(node *yaml.Node) interface{} {
	if node.Tag == "" || node.Tag == "!!str" {
		return node.Value
	}

	// Try to parse as native types based on tag
	switch node.Tag {
	case "!!null":
		return nil
	case "!!bool":
		return node.Value == "true"
	case "!!int":
		var i int64
		if _, err := fmt.Sscanf(node.Value, "%d", &i); err == nil {
			return i
		}
		return node.Value
	case "!!float":
		var f float64
		if _, err := fmt.Sscanf(node.Value, "%g", &f); err == nil {
			return f
		}
		return node.Value
	}

	// Fallback: try to auto-detect
	// Check for booleans
	lower := strings.ToLower(node.Value)
	if lower == "true" || lower == "yes" {
		return true
	}
	if lower == "false" || lower == "no" {
		return false
	}
	if lower == "null" || lower == "~" || node.Value == "" {
		return nil
	}

	return node.Value
}
