package merge

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge2FilesEmpty(t *testing.T) {
	data1 := make(map[string]interface{})
	data2 := make(map[string]interface{})

	merged := merge2Files(data1, data2)

	if len(merged) != 0 {
		t.Errorf("Expected empty map, but got %v", merged)
	}
}

func TestMerge2Files(t *testing.T) {
	// given
	data1 := map[string]any{
		"items": []any{
			map[string]any{"id": 1, "name": "Item 1"},
			map[string]any{"id": 2, "name": "Item 2"},
		},
	}
	data2 := map[string]any{}
	//var want map[string]interface{} = nil
	// when
	merged := merge2Files(data1, data2)
	// then
	require.NotNil(t, merged)
	require.NotEmpty(t, merged)
}

func TestMerge2FilesFirstNonEmpty(t *testing.T) {
	data1 := map[string]interface{}{
		"name": "John",
		"age":  30,
	}
	data2 := make(map[string]interface{})

	merged := merge2Files(data1, data2)

	// verify the merged map has the same key-value pairs as the first parameter
	for key, value := range data1 {
		if merged[key] != value {
			t.Errorf("Expected merged map to contain %v:%v, but got %v:%v", key, value, key, merged[key])
		}
	}

	// verify the merged map has the same length as the first parameter
	if len(merged) != len(data1) {
		t.Errorf("Expected merged map length to be %v, but got %v", len(data1), len(merged))
	}
}

func TestMerge2FilesSecondNonEmpty(t *testing.T) {
	data1 := make(map[string]interface{})
	data2 := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	merged := merge2Files(data1, data2)

	// verify the merged map has the same key-value pairs as the second parameter
	for key, value := range data2 {
		if merged[key] != value {
			t.Errorf("Expected merged map to contain %v:%v, but got %v:%v", key, value, key, merged[key])
		}
	}

	// verify the merged map has the same length as the second parameter
	if len(merged) != len(data2) {
		t.Errorf("Expected merged map length to be %v, but got %v", len(data2), len(merged))
	}
}

func TestMerge2FilesNested(t *testing.T) {
	data1 := map[string]interface{}{
		"name": "John",
		"age":  30,
		"address": map[string]interface{}{
			"street": "123 Main St",
			"city":   "New York",
			"state":  "NY",
			"zip":    "10001",
		},
	}

	data2 := map[string]interface{}{
		"age": 35,
		"address": map[string]interface{}{
			"street": "456 Elm St",
			"state":  "CA",
		},
		"phone": "555-1234",
	}

	expected := map[string]interface{}{
		"name": "John",
		"age":  35,
		"address": map[string]interface{}{
			"street": "456 Elm St",
			"city":   "New York",
			"state":  "CA",
			"zip":    "10001",
		},
		"phone": "555-1234",
	}

	merged := merge2Files(data1, data2)

	// convert merged and expected maps to JSON strings for comparison
	mergedJSON, _ := json.Marshal(merged)
	expectedJSON, _ := json.Marshal(expected)

	// verify that merged and expected JSON strings are equal
	if string(mergedJSON) != string(expectedJSON) {
		t.Errorf("Expected merged map:\n\n%v\n\nbut got:\n\n%v", string(expectedJSON), string(mergedJSON))
	}
}

func TestMerge2FilesKongConfig(t *testing.T) {
	data1 := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"name": "example-service",
				"url":  "http://example.com",
			},
		},
		"plugins": []interface{}{
			map[string]interface{}{
				"name": "key-auth",
				"config": map[string]interface{}{
					"key_names": []interface{}{"apikey"},
				},
			},
		},
	}

	data2 := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"name": "example-service",
				"url":  "https://example.com",
				"routes": []interface{}{
					map[string]interface{}{
						"name":          "example-route",
						"paths":         []interface{}{"/example"},
						"strip_path":    false,
						"preserve_host": true,
					},
				},
			},
		},
		"plugins": []interface{}{
			map[string]interface{}{
				"name": "jwt",
				"config": map[string]interface{}{
					"uri_param_names": []interface{}{"jwt"},
				},
			},
		},
	}

	expected := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"name": "example-service",
				"url":  "https://example.com",
				"routes": []interface{}{
					map[string]interface{}{
						"name":          "example-route",
						"paths":         []interface{}{"/example"},
						"strip_path":    false,
						"preserve_host": true,
					},
				},
			},
		},
		"plugins": []interface{}{
			map[string]interface{}{
				"name": "key-auth",
				"config": map[string]interface{}{
					"key_names": []interface{}{"apikey"},
				},
			},
			map[string]interface{}{
				"name": "jwt",
				"config": map[string]interface{}{
					"uri_param_names": []interface{}{"jwt"},
				},
			},
		},
	}

	merged := merge2Files(data1, data2)

	// convert merged and expected maps to JSON strings for comparison
	mergedJSON, _ := json.Marshal(merged)
	expectedJSON, _ := json.Marshal(expected)

	// verify that merged and expected JSON strings are equal
	if string(mergedJSON) != string(expectedJSON) {
		t.Errorf("Expected merged map to be %v, but got %v", string(expectedJSON), string(mergedJSON))
	}
}
