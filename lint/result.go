package lint

import (
	"fmt"
	"sort"
	"strings"
)

// LintResult represents a single linting violation found during evaluation.
type LintResult struct {
	// Message is a human-readable description of the violation.
	Message string `json:"message" yaml:"message"`
	// Path is the JSONPath segments to the violation location.
	Path string `json:"path" yaml:"path"`
	// Severity is the severity level of the violation.
	Severity Severity `json:"severity" yaml:"severity"`
	// Line is the 1-based line number in the source document where the violation occurred.
	Line int `json:"line" yaml:"line"`
	// Column is the 1-based column number in the source document where the violation occurred.
	Column int `json:"column" yaml:"column"`
	// RuleName is the name of the rule that produced this result.
	RuleName string `json:"rule_name" yaml:"rule_name"`
	// Source is the source file name (if available).
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// String returns a human-readable representation of the lint result.
// Format: [severity][line:col] description: message
func (r LintResult) String() string {
	location := ""
	if r.Line > 0 {
		location = fmt.Sprintf("[%d:%d] ", r.Line, r.Column)
	}
	return fmt.Sprintf("[%s]%s%s: %s", r.Severity, location, r.RuleName, r.Message)
}

// SortResults sorts lint results by severity (highest first), then by line number.
func SortResults(results []LintResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Severity != results[j].Severity {
			return results[i].Severity < results[j].Severity
		}
		if results[i].Line != results[j].Line {
			return results[i].Line < results[j].Line
		}
		return results[i].Column < results[j].Column
	})
}

// FormatResults formats a slice of lint results into a plain text string.
func FormatResults(results []LintResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, r := range results {
		sb.WriteString(r.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// CountBySeverity counts the number of results at or above the given severity threshold.
// A lower Severity numeric value means higher severity. So CountBySeverity(SeverityWarn)
// will count all results with severity Error or Warn.
func CountBySeverity(results []LintResult, threshold Severity) int {
	count := 0
	for i := range results {
		if results[i].Severity <= threshold {
			count++
		}
	}
	return count
}
