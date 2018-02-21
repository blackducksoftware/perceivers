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
	"fmt"
)

// BDPodAnnotationPrefix is the prefix used for BlackDuckAnnotations in pod annotations
var BDPodAnnotationPrefix = "quality.pod.openshift.io"

// BlackDuckPodAnnotation describes the data model for pod annotation
type BlackDuckPodAnnotation struct {
	policyViolationCount int
	vulnerabilityCount   int
	overallStatus        string
}

// NewBlackDuckPodAnnotation creates a new BlackDuckPodAnnotation object
func NewBlackDuckPodAnnotation(policyViolationCount int, vulnerabilityCount int, overallStatus string) *BlackDuckPodAnnotation {
	return &BlackDuckPodAnnotation{
		policyViolationCount: policyViolationCount,
		vulnerabilityCount:   vulnerabilityCount,
		overallStatus:        overallStatus,
	}
}

// HasPolicyViolations returns true if the pod has any policy violations
func (bdpa *BlackDuckPodAnnotation) HasPolicyViolations() bool {
	return bdpa.policyViolationCount > 0
}

// HasVulnerabilities returns true if the pod has any vulnerabilities
func (bdpa *BlackDuckPodAnnotation) HasVulnerabilities() bool {
	return bdpa.vulnerabilityCount > 0
}

// GetVulnerabilityCount returns the number of pod vulnerabilities
func (bdpa *BlackDuckPodAnnotation) GetVulnerabilityCount() int {
	return bdpa.vulnerabilityCount
}

// GetPolicyViolationCount returns the number of pod policy violations
func (bdpa *BlackDuckPodAnnotation) GetPolicyViolationCount() int {
	return bdpa.policyViolationCount
}

// GetOverallStatus returns the pod overall status
func (bdpa *BlackDuckPodAnnotation) GetOverallStatus() string {
	return bdpa.overallStatus
}

// CreatePodLabels returns a map of labels from a BlackDuckPodAnnotation object
func CreatePodLabels(podAnnotations *BlackDuckPodAnnotation) map[string]string {
	labels := make(map[string]string)
	labels["com.blackducksoftware.pod.policy-violations"] = fmt.Sprintf("%d", podAnnotations.GetPolicyViolationCount())
	labels["com.blackducksoftware.pod.has-policy-violations"] = fmt.Sprintf("%t", podAnnotations.HasPolicyViolations())
	labels["com.blackducksoftware.pod.vulnerabilities"] = fmt.Sprintf("%d", podAnnotations.GetVulnerabilityCount())
	labels["com.blackducksoftware.pod.has-vulnerabilities"] = fmt.Sprintf("%t", podAnnotations.HasVulnerabilities())
	labels["com.blackducksoftware.pod.overall-status"] = podAnnotations.GetOverallStatus()

	return labels
}

// CreatePodAnnotations returns a map of annotations from a BlackDuckPodAnnotation object
func CreatePodAnnotations(podAnnotations *BlackDuckPodAnnotation) map[string]string {
	newAnnotations := make(map[string]string)
	vulnAnnotations := CreateBlackDuckVulnerabilityAnnotation(podAnnotations.HasVulnerabilities() == true, "", podAnnotations.GetVulnerabilityCount())
	policyAnnotations := CreateBlackDuckPolicyAnnotation(podAnnotations.HasPolicyViolations() == true, "", podAnnotations.GetPolicyViolationCount())

	newAnnotations[fmt.Sprintf("%s/vulnerability.blackduck", BDPodAnnotationPrefix)] = vulnAnnotations.AsString()
	newAnnotations[fmt.Sprintf("%s/policy.blackduck", BDPodAnnotationPrefix)] = policyAnnotations.AsString()

	return newAnnotations
}
