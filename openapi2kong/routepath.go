package openapi2kong

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// getRoutePaths returns the `x-kong-route-path` property, validated and normalized
// into a list of Kong route paths. The extension can be either a single string,
// or an array of strings. Each entry must be a valid Kong route path, so either a
// plain (prefix) path starting with "/", or a regex path starting with "~/".
// Returns nil (and no error) if the extension is not present.
func getRoutePaths(extensions *orderedmap.Map[string, *yaml.Node]) ([]string, error) {
	if extensions == nil {
		return nil, nil
	}

	xKongRoutePath, ok := extensions.Get("x-kong-route-path")
	if !ok || xKongRoutePath == nil {
		return nil, nil
	}

	const typeError = "expected 'x-kong-route-path' to be a string or a non-empty array of strings"

	var paths []string
	switch xKongRoutePath.Kind {
	case yaml.ScalarNode:
		if xKongRoutePath.ShortTag() != "!!str" {
			return nil, fmt.Errorf("%s, got '%s'", typeError, xKongRoutePath.Value)
		}
		paths = []string{xKongRoutePath.Value}
	case yaml.SequenceNode:
		if len(xKongRoutePath.Content) == 0 {
			return nil, fmt.Errorf("%s, got an empty array", typeError)
		}
		for _, entry := range xKongRoutePath.Content {
			if entry.Kind != yaml.ScalarNode || entry.ShortTag() != "!!str" {
				return nil, fmt.Errorf("%s, got a non-string array entry", typeError)
			}
			paths = append(paths, entry.Value)
		}
	default:
		return nil, fmt.Errorf("%s", typeError)
	}

	for _, p := range paths {
		if err := validateRoutePath(p); err != nil {
			return nil, fmt.Errorf("invalid 'x-kong-route-path' value '%s'; %w", p, err)
		}
	}

	return paths, nil
}

// validateRoutePath validates a single Kong route path, mirroring the checks of the
// Kong Gateway route schema ('router_path' in kong/db/schema/typedefs.lua), so that
// errors are reported at conversion time instead of when syncing to Kong:
//
//   - it must start with "/" (plain path) or "~/" (regex path)
//   - it must not have empty segments ("//")
//   - a plain path must be properly percent-encoded, and normalized
//   - a regex path must be a valid regex
func validateRoutePath(path string) error {
	if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~/") {
		return errors.New("a route path must start with '/' (plain path) or '~/' (regex path)")
	}

	if strings.Contains(path, "//") {
		return errors.New("a route path must not have empty segments ('//')")
	}

	if strings.HasPrefix(path, "~") {
		return validateRegexRoutePath(path[1:])
	}

	return validatePlainRoutePath(path)
}

// rfc3986PathChars matches a path consisting of only RFC 3986 unreserved, sub-delims,
// and path characters (same set as the Kong Gateway 'path' typedef).
var rfc3986PathChars = regexp.MustCompile(`^/[a-zA-Z0-9._~/%:@!$&'()*+,;=-]*$`)

// percentTriplet matches a percent-encoded character (case-insensitive hex).
var percentTriplet = regexp.MustCompile(`%[0-9a-fA-F]{2}`)

// validatePlainRoutePath validates a plain (non-regex) path the same way Kong does.
func validatePlainRoutePath(path string) error {
	// Kong only verifies the percent-encoding if the path consists of RFC 3986 characters,
	// other characters are accepted (and will be escaped by Kong when proxying).
	if rfc3986PathChars.MatchString(path) {
		raw := percentTriplet.ReplaceAllString(path, "___")
		if idx := strings.Index(raw, "%"); idx >= 0 {
			end := idx + 3
			if end > len(raw) {
				end = len(raw)
			}
			return fmt.Errorf("invalid url-encoded value: '%s'", raw[idx:end])
		}
	}

	if normalized := normalizeRoutePath(path); normalized != path {
		return fmt.Errorf("a plain route path must be normalized, consider using '%s' instead", normalized)
	}

	return nil
}

// reservedChars are the RFC 3986 reserved characters, these remain percent-encoded
// when normalizing a path, all others are decoded.
const reservedChars = "!*'();:@&=+$,/?%#[]"

// normalizeRoutePath normalizes a plain path, the same way as Kong Gateway does
// ('normalize' in kong/tools/uri.lua):
//
//   - percent-encoded reserved characters are kept, with upper-case hex digits
//   - other percent-encoded characters are decoded
//   - dot-segments ("." and "..") are removed
func normalizeRoutePath(path string) string {
	if path == "" || path == "/" {
		return path
	}

	if strings.Contains(path, "%") {
		path = percentTriplet.ReplaceAllStringFunc(path, func(triplet string) string {
			num, _ := strconv.ParseUint(triplet[1:], 16, 8)
			if strings.IndexByte(reservedChars, byte(num)) >= 0 {
				return strings.ToUpper(triplet)
			}
			return string([]byte{byte(num)})
		})
	}

	if !strings.Contains(path, ".") {
		return path
	}

	return removeDotSegments(path)
}

// removeDotSegments implements the "remove_dot_segments" algorithm from
// RFC 3986 section 5.2.4.
func removeDotSegments(input string) string {
	output := make([]string, 0)
	for input != "" {
		switch {
		case strings.HasPrefix(input, "../"):
			input = input[3:]
		case strings.HasPrefix(input, "./"):
			input = input[2:]
		case strings.HasPrefix(input, "/./"):
			input = input[2:]
		case input == "/.":
			input = "/"
		case strings.HasPrefix(input, "/../"):
			input = input[3:]
			if len(output) > 0 {
				output = output[:len(output)-1]
			}
		case input == "/..":
			input = "/"
			if len(output) > 0 {
				output = output[:len(output)-1]
			}
		case input == "." || input == "..":
			input = ""
		default:
			// move the first segment (including its leading "/", if any) to the output
			start := 0
			if input[0] == '/' {
				start = 1
			}
			end := strings.IndexByte(input[start:], '/')
			if end < 0 {
				end = len(input)
			} else {
				end += start
			}
			output = append(output, input[:end])
			input = input[end:]
		}
	}
	return strings.Join(output, "")
}

// definiteRegexErrors are the Go regex parse errors that are also errors in PCRE
// (traditional router) and Rust regex (expressions router). Other Go parse errors
// are typically PCRE features that are not supported by Go's RE2 syntax (eg.
// look-arounds, back-references, atomic groups, possessive quantifiers), those
// are left to be validated by Kong itself.
var definiteRegexErrors = map[syntax.ErrorCode]bool{
	syntax.ErrMissingParen:          true,
	syntax.ErrUnexpectedParen:       true,
	syntax.ErrMissingBracket:        true,
	syntax.ErrInvalidCharRange:      true,
	syntax.ErrTrailingBackslash:     true,
	syntax.ErrMissingRepeatArgument: true,
}

// validateRegexRoutePath validates the regex of a regex path (without the "~" prefix).
func validateRegexRoutePath(regex string) error {
	_, err := syntax.Parse(regex, syntax.Perl)
	if err == nil {
		return nil
	}

	var syntaxErr *syntax.Error
	if !errors.As(err, &syntaxErr) || !definiteRegexErrors[syntaxErr.Code] {
		return nil // possibly a valid PCRE construct, let Kong validate it
	}

	if syntaxErr.Code == syntax.ErrMissingRepeatArgument && strings.Contains(regex, "(*") {
		return nil // PCRE verbs like "(*SKIP)(*FAIL)" are invalid in Go, but valid in PCRE
	}

	return fmt.Errorf("invalid regex '%s': %s", regex, syntaxErr.Code)
}
