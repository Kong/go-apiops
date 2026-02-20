package lint

import "fmt"

// coreTruthy checks that the value is truthy (not false, "", 0, nil).
func coreTruthy(targetVal interface{}, _ map[string]interface{}) []string {
	if !isTruthy(targetVal) {
		return []string{fmt.Sprintf("value must be truthy, got %v", targetVal)}
	}
	return nil
}

// coreFalsy checks that the value is falsy (false, "", 0, nil).
func coreFalsy(targetVal interface{}, _ map[string]interface{}) []string {
	if isTruthy(targetVal) {
		return []string{fmt.Sprintf("value must be falsy, got %v", targetVal)}
	}
	return nil
}

// coreDefined checks that the value is defined (not nil / not absent).
func coreDefined(targetVal interface{}, _ map[string]interface{}) []string {
	if targetVal == nil {
		return []string{"value must be defined"}
	}
	return nil
}

// coreUndefined checks that the value is undefined (nil / absent).
func coreUndefined(targetVal interface{}, _ map[string]interface{}) []string {
	if targetVal != nil {
		return []string{fmt.Sprintf("value must be undefined, got %v", targetVal)}
	}
	return nil
}
