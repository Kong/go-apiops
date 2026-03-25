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
