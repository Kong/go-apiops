package jsonbasics

import (
	"testing"
)

func Test_toObject(t *testing.T) {
	defaultTests := []struct {
		data     interface{}
		isObject bool
	}{
		{map[string]interface{}{"hello": "world"}, true},
		{"something not an object", false},
	}

	for _, tst := range defaultTests {
		var obj map[string]interface{}
		obj, _ = ToObject(tst.data)
		if (obj == nil) == tst.isObject {
			if tst.isObject {
				t.Errorf("Expected to return an object")
			} else {
				t.Errorf("Expected to NOT return an object")
			}
		}
	}
}

func Test_toArray(t *testing.T) {
	defaultTests := []struct {
		data    interface{}
		isArray bool
	}{
		{map[string]interface{}{"hello": "world"}, false},
		{[]interface{}{"hello", "world"}, true},
	}

	for _, tst := range defaultTests {
		var arr []interface{}
		arr, _ = ToArray(tst.data)
		if (arr == nil) == tst.isArray {
			if tst.isArray {
				t.Errorf("Expected to return an array")
			} else {
				t.Errorf("Expected to NOT return an array")
			}
		}
	}
}
