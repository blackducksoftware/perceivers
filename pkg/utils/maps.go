package utils

import (
	"strings"
)

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

// StringMapContains will return true all the key/value pairs in subset
// exist and are the same in bigMap
func StringMapContains(bigMap map[string]string, subset map[string]string) bool {
	for k, v := range subset {
		if val, ok := bigMap[k]; !ok {
			return false
		} else if val != v {
			return false
		}
	}

	return true
}

// MapContainsBDEntries returns true if the newMap contains all the important
// blackduck entries from the origMap
func MapContainsBDEntries(origMap map[string]string, newMap map[string]string) bool {
	important := make(map[string]string)

	for k, v := range origMap {
		if strings.Contains(k, "blackduck") {
			important[k] = v
		}
	}

	return StringMapContains(newMap, important)
}
