package lint

import (
	"fmt"
	"regexp"
	"strings"
)

// corePattern validates a string value against match/notMatch regex patterns.
// Supports the Spectral pattern format: plain regex like "[a-z]+" or delimited like "/[a-z]+/i".
func corePattern(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	strVal := fmt.Sprintf("%v", targetVal)

	var results []string

	if matchPattern, ok := getStringOption(opts, "match"); ok {
		re, err := compilePattern(matchPattern)
		if err != nil {
			return []string{fmt.Sprintf("invalid 'match' pattern %q: %s", matchPattern, err)}
		}
		if !re.MatchString(strVal) {
			results = append(results, fmt.Sprintf("`%s` does not match the expression `%s`", strVal, matchPattern))
		}
	}

	if notMatchPattern, ok := getStringOption(opts, "notMatch"); ok {
		re, err := compilePattern(notMatchPattern)
		if err != nil {
			return []string{fmt.Sprintf("invalid 'notMatch' pattern %q: %s", notMatchPattern, err)}
		}
		if re.MatchString(strVal) {
			results = append(results, fmt.Sprintf("`%s` must not match the expression `%s`", strVal, notMatchPattern))
		}
	}

	return results
}

// compilePattern compiles a Spectral-style regex pattern.
// Supports plain patterns like "[a-z]+" or delimited patterns with flags like "/[a-z]+/i".
// Maps JavaScript regex flags to Go inline flags where possible:
//   - i -> (?i) case-insensitive
//   - m -> (?m) multiline
//   - s -> (?s) dot matches newline
func compilePattern(pattern string) (*regexp.Regexp, error) {
	// Check for delimited pattern like /pattern/flags
	if strings.HasPrefix(pattern, "/") {
		lastSlash := strings.LastIndex(pattern, "/")
		if lastSlash > 0 {
			regexBody := pattern[1:lastSlash]
			flags := pattern[lastSlash+1:]
			goFlags := convertFlags(flags)
			if goFlags != "" {
				return regexp.Compile("(?" + goFlags + ")" + regexBody)
			}
			return regexp.Compile(regexBody)
		}
	}
	return regexp.Compile(pattern)
}

// convertFlags converts JavaScript regex flags to Go inline flag syntax.
func convertFlags(jsFlags string) string {
	var goFlags strings.Builder
	for _, f := range jsFlags {
		switch f {
		case 'i':
			goFlags.WriteRune('i')
		case 'm':
			goFlags.WriteRune('m')
		case 's':
			goFlags.WriteRune('s')
			// g, u, y are JavaScript-specific and have no Go equivalent (or are default behavior)
		}
	}
	return goFlags.String()
}

// coreCasing validates that a string value matches a specific casing style.
// Options:
//   - type: one of flat, camel, pascal, kebab, cobol, snake, macro (required)
//   - disallowDigits: if true, digits are not allowed (optional, default false)
//   - separator.char: additional character to separate groups of words (optional)
//   - separator.allowLeading: whether separator char can appear at the start (optional)
func coreCasing(targetVal interface{}, opts map[string]interface{}) []string {
	if targetVal == nil {
		return nil
	}

	strVal, ok := targetVal.(string)
	if !ok {
		return []string{"value must be a string for casing check"}
	}

	if strVal == "" {
		return nil
	}

	casingType, ok := getStringOption(opts, "type")
	if !ok {
		return []string{"casing function requires 'type' option"}
	}

	disallowDigits := getBoolOption(opts, "disallowDigits")

	var separatorChar string
	var separatorAllowLeading bool
	if sepRaw, ok := opts["separator"]; ok {
		if sepMap, ok := sepRaw.(map[string]interface{}); ok {
			if ch, ok := sepMap["char"]; ok {
				separatorChar = fmt.Sprintf("%v", ch)
			}
			if al, ok := sepMap["allowLeading"]; ok {
				if b, ok := al.(bool); ok {
					separatorAllowLeading = b
				}
			}
		}
	}

	re, err := buildCasingRegex(casingType, disallowDigits, separatorChar, separatorAllowLeading)
	if err != nil {
		return []string{err.Error()}
	}

	// Special case: if the value is a single character matching the separator, and leading is allowed
	if len(strVal) == 1 && separatorChar != "" && separatorAllowLeading && strVal == separatorChar {
		return nil
	}

	if !re.MatchString(strVal) {
		return []string{fmt.Sprintf("must be %s case", casingType)}
	}

	return nil
}

// buildCasingRegex constructs the validation regex for a specific casing type.
func buildCasingRegex(casingType string, disallowDigits bool,
	separatorChar string, allowLeading bool,
) (*regexp.Regexp, error) {
	digits := ""
	if !disallowDigits {
		digits = "0-9"
	}

	var basePattern string
	switch casingType {
	case "flat":
		basePattern = "[a-z][a-z" + digits + "]*"
	case "camel":
		basePattern = "[a-z][a-z" + digits + "]*(?:[A-Z" + digits + "](?:[a-z" + digits + "]+|$))*"
	case "pascal":
		basePattern = "[A-Z][a-z" + digits + "]*(?:[A-Z" + digits + "](?:[a-z" + digits + "]+|$))*"
	case "kebab":
		basePattern = "[a-z][a-z" + digits + "]*(?:-[a-z" + digits + "]+)*"
	case "cobol":
		basePattern = "[A-Z][A-Z" + digits + "]*(?:-[A-Z" + digits + "]+)*"
	case "snake":
		basePattern = "[a-z][a-z" + digits + "]*(?:_[a-z" + digits + "]+)*"
	case "macro":
		basePattern = "[A-Z][A-Z" + digits + "]*(?:_[A-Z" + digits + "]+)*"
	default:
		return nil, fmt.Errorf("unknown casing type: %q, expected one of: flat, camel, pascal, kebab, cobol, snake, macro",
			casingType)
	}

	if separatorChar != "" {
		escapedSep := regexp.QuoteMeta(separatorChar)
		sepPattern := "[" + escapedSep + "]"
		leading := ""
		if allowLeading {
			leading = sepPattern + "?"
		}
		return regexp.Compile("^" + leading + basePattern + "(?:" + sepPattern + basePattern + ")*$")
	}

	return regexp.Compile("^" + basePattern + "$")
}
