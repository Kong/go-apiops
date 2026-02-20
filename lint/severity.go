package lint

import (
	"fmt"
	"strings"
)

// Severity represents the severity level of a lint result.
// Lower numeric values indicate higher severity.
type Severity int

const (
	// SeverityError is the highest severity level.
	SeverityError Severity = 0
	// SeverityWarn indicates a warning.
	SeverityWarn Severity = 1
	// SeverityInfo indicates an informational message.
	SeverityInfo Severity = 2
	// SeverityHint indicates a hint or suggestion.
	SeverityHint Severity = 3
)

// ParseSeverity converts a string to a Severity value.
// Accepted values (case-insensitive): "error", "warn", "info", "hint".
// Returns an error for unrecognized values.
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return SeverityError, nil
	case "warn", "warning":
		return SeverityWarn, nil
	case "info", "information":
		return SeverityInfo, nil
	case "hint":
		return SeverityHint, nil
	default:
		return SeverityWarn, fmt.Errorf("unknown severity: %q, expected one of: error, warn, info, hint", s)
	}
}

// String returns the string representation of a Severity.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarn:
		return "warn"
	case SeverityInfo:
		return "info"
	case SeverityHint:
		return "hint"
	default:
		return "unknown"
	}
}
