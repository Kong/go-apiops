package openapitools

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrossProduct(t *testing.T) {
	testCases := []struct {
		name           string
		slices         [][]any
		expectedResult [][]any
	}{
		{
			name:           "Empty slices",
			slices:         [][]any{},
			expectedResult: [][]any{{}},
		},
		{
			name:   "Single slice",
			slices: [][]any{{"a", "b", "c"}},
			expectedResult: [][]any{
				{"a"},
				{"b"},
				{"c"},
			},
		},
		{
			name: "Mixed types and different length input slices",
			slices: [][]any{
				{"a", "b", "c"},
				{1, 2},
			},
			expectedResult: [][]any{
				{"a", 1},
				{"a", 2},
				{"b", 1},
				{"b", 2},
				{"c", 1},
				{"c", 2},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CrossProduct(tc.slices...)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func Test_ToKebabCase(t *testing.T) {
	// Test tool name kebab-case normalization
	testCases := []struct {
		input    string
		expected string
	}{
		{"getFlights", "get-flights"},
		{"get_flights", "get-flights"},
		{"GetFlights", "get-flights"},
		{"get-flights", "get-flights"},
		{"listAllUsers", "list-all-users"},
		{"CreateNewItem", "create-new-item"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ToKebabCase(tc.input)
			assert.Equal(t, tc.expected, result, fmt.Sprintf("ToKebabCase(%s)", tc.input))
		})
	}
}

func Test_DeduplicateHeaderEnumValues(t *testing.T) {
	testCases := []struct {
		name     string
		input    []any
		expected []any
	}{
		{
			name:     "No duplicates - already lowercase",
			input:    []any{"us-east", "us-west", "eu-central"},
			expected: []any{"us-east", "us-west", "eu-central"},
		},
		{
			name:     "Case-insensitive duplicates - normalized to lowercase",
			input:    []any{"us-east", "US-EAST", "Us-East"},
			expected: []any{"us-east"},
		},
		{
			name:     "Mixed case duplicates and unique - all normalized to lowercase",
			input:    []any{"us-east", "US-EAST", "us-west", "US-WEST", "eu-central"},
			expected: []any{"us-east", "us-west", "eu-central"},
		},
		{
			name:     "Non-string values preserved",
			input:    []any{1, 2, 3, "test", "TEST"},
			expected: []any{1, 2, 3, "test"},
		},
		{
			name:     "Mixed types with duplicates",
			input:    []any{"v1", "V1", 1, "v2", 2},
			expected: []any{"v1", 1, "v2", 2},
		},
		{
			name:     "Uppercase values normalized to lowercase",
			input:    []any{"US-EAST", "us-east", "Us-East"},
			expected: []any{"us-east"},
		},
		{
			name:     "All uppercase converted to lowercase",
			input:    []any{"US-EAST", "US-WEST"},
			expected: []any{"us-east", "us-west"},
		},
		{
			name:     "Empty string filtered out",
			input:    []any{"", "prod", "staging"},
			expected: []any{"prod", "staging"},
		},
		{
			name:     "Only empty string returns empty slice",
			input:    []any{""},
			expected: []any{},
		},
		{
			name:     "Multiple empty strings filtered out",
			input:    []any{"", "prod", "", "staging", ""},
			expected: []any{"prod", "staging"},
		},
		{
			name:     "Empty string with duplicates",
			input:    []any{"", "PROD", "prod", ""},
			expected: []any{"prod"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DeduplicateHeaderEnumValues(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
