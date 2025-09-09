package openapi2kong

import (
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
			result := crossProduct(tc.slices...)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
