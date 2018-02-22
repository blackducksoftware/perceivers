/*
Copyright (C) 2018 Black Duck Software, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package annotations

import (
	"strings"
)

// MapContainsBlackDuckEntries returns true if the origMap contains all the important
// blackduck entries from the newMap
func MapContainsBlackDuckEntries(origMap map[string]string, newMap map[string]string) bool {
	important := make(map[string]string)

	for k, v := range newMap {
		if strings.Contains(k, "blackduck") || strings.Contains(k, BDImageAnnotationPrefix) {
			important[k] = v
		}
	}

	return StringMapContains(origMap, important)
}

// StringMapContains will return true all the key/value pairs in subset
// exist and are the same in bigMap
func StringMapContains(bigMap map[string]string, subset map[string]string) bool {
	for k, v := range subset {
		if val, ok := bigMap[k]; !ok {
			return false
		} else if strings.Contains(k, BDImageAnnotationPrefix) || strings.Contains(k, BDPodAnnotationPrefix) {
			// These keys can be either a BlackDuckAnnotation or just a string
			if !CompareBlackDuckAnnotationJSONStrings(bigMap[k], v) && val != v {
				return false
			}
		} else if val != v {
			return false
		}
	}

	return true
}
