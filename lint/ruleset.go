package lint

import (
	"fmt"

	"sigs.k8s.io/yaml"
)

// Ruleset represents a parsed Spectral-compatible ruleset.
type Ruleset struct {
	Rules map[string]Rule
}

// Rule represents a single linting rule in a ruleset.
type Rule struct {
	// Description is a human-readable description of the rule.
	Description string
	// Message is an optional custom message template for violations.
	Message string
	// Severity is the severity level for violations of this rule (default: warn).
	Severity Severity
	// Given is one or more JSONPath expressions identifying target locations.
	Given []string
	// Then is one or more function applications to run on each target.
	Then []ThenEntry
}

// ThenEntry represents a single function application within a rule's "then" clause.
type ThenEntry struct {
	// Field is an optional property name to drill into after JSONPath selection.
	Field string
	// Function is the name of the core function to apply.
	Function string
	// FunctionOptions are the function-specific configuration options.
	FunctionOptions map[string]interface{}
}

// ParseRuleset parses a Spectral-compatible ruleset from YAML or JSON bytes.
func ParseRuleset(data []byte) (*Ruleset, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse ruleset: %w", err)
	}

	rulesRaw, ok := raw["rules"]
	if !ok {
		return nil, fmt.Errorf("ruleset is missing required 'rules' key")
	}

	rulesMap, ok := rulesRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'rules' must be an object, got %T", rulesRaw)
	}

	ruleset := &Ruleset{
		Rules: make(map[string]Rule, len(rulesMap)),
	}

	for name, ruleRaw := range rulesMap {
		rule, err := parseRule(ruleRaw)
		if err != nil {
			return nil, fmt.Errorf("error parsing rule %q: %w", name, err)
		}
		ruleset.Rules[name] = rule
	}

	return ruleset, nil
}

func parseRule(raw interface{}) (Rule, error) {
	ruleMap, ok := raw.(map[string]interface{})
	if !ok {
		return Rule{}, fmt.Errorf("rule must be an object, got %T", raw)
	}

	rule := Rule{
		Severity: SeverityWarn, // default severity
	}

	// Parse description
	if desc, ok := ruleMap["description"]; ok {
		rule.Description = fmt.Sprintf("%v", desc)
	}

	// Parse message
	if msg, ok := ruleMap["message"]; ok {
		rule.Message = fmt.Sprintf("%v", msg)
	}

	// Parse severity
	if sev, ok := ruleMap["severity"]; ok {
		switch v := sev.(type) {
		case string:
			parsed, err := ParseSeverity(v)
			if err != nil {
				return Rule{}, fmt.Errorf("invalid severity: %w", err)
			}
			rule.Severity = parsed
		case int:
			rule.Severity = Severity(v)
		case int64:
			rule.Severity = Severity(v)
		case float64:
			rule.Severity = Severity(int(v))
		default:
			return Rule{}, fmt.Errorf("severity must be a string or number, got %T", sev)
		}
	}

	// Parse given (can be a string or array of strings)
	givenRaw, ok := ruleMap["given"]
	if !ok {
		return Rule{}, fmt.Errorf("rule is missing required 'given' field")
	}
	given, err := parseGiven(givenRaw)
	if err != nil {
		return Rule{}, err
	}
	rule.Given = given

	// Parse then (can be a single entry or array of entries)
	thenRaw, ok := ruleMap["then"]
	if !ok {
		return Rule{}, fmt.Errorf("rule is missing required 'then' field")
	}
	then, err := parseThen(thenRaw)
	if err != nil {
		return Rule{}, err
	}
	rule.Then = then

	return rule, nil
}

func parseGiven(raw interface{}) ([]string, error) {
	switch v := raw.(type) {
	case string:
		return []string{v}, nil
	case []interface{}:
		result := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("given[%d] must be a string, got %T", i, item)
			}
			result = append(result, s)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("'given' must be a string or array of strings, got %T", raw)
	}
}

func parseThen(raw interface{}) ([]ThenEntry, error) {
	switch v := raw.(type) {
	case map[string]interface{}:
		entry, err := parseThenEntry(v)
		if err != nil {
			return nil, err
		}
		return []ThenEntry{entry}, nil
	case []interface{}:
		result := make([]ThenEntry, 0, len(v))
		for i, item := range v {
			entryMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("then[%d] must be an object, got %T", i, item)
			}
			entry, err := parseThenEntry(entryMap)
			if err != nil {
				return nil, fmt.Errorf("then[%d]: %w", i, err)
			}
			result = append(result, entry)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("'then' must be an object or array of objects, got %T", raw)
	}
}

func parseThenEntry(raw map[string]interface{}) (ThenEntry, error) {
	entry := ThenEntry{}

	// Parse field
	if field, ok := raw["field"]; ok {
		entry.Field = fmt.Sprintf("%v", field)
	}

	// Parse function
	funcRaw, ok := raw["function"]
	if !ok {
		return ThenEntry{}, fmt.Errorf("'then' entry is missing required 'function' field")
	}
	funcName, ok := funcRaw.(string)
	if !ok {
		return ThenEntry{}, fmt.Errorf("'function' must be a string, got %T", funcRaw)
	}
	entry.Function = funcName

	// Parse functionOptions
	if opts, ok := raw["functionOptions"]; ok {
		optsMap, ok := opts.(map[string]interface{})
		if !ok {
			return ThenEntry{}, fmt.Errorf("'functionOptions' must be an object, got %T", opts)
		}
		entry.FunctionOptions = optsMap
	}

	return entry, nil
}
