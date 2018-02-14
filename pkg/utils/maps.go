package utils

// MapMerge merges two maps together and returns the results.
// If both maps contain the same key, then the value of the
// existing key will be overwritten with the value from the new map
func MapMerge(base map[string]string, new map[string]string) map[string]string {
	newMap := make(map[string]string)
	if base != nil {
		for k, v := range base {
			newMap[k] = v
		}
	}
	for k, v := range new {
		newMap[k] = v
	}
	return newMap
}

// SameStringMap will return true if the two maps are identical
func SameStringMap(old map[string]string, new map[string]string) bool {
	for k, v := range new {
		if val, ok := old[k]; !ok {
			return true
		} else if val != v {
			return true
		}
	}

	return false
}
