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
	"strings"
)

// BDImageAnnotationPrefix is the prefix used for BlackDuckAnnotations in image annotations
var BDImageAnnotationPrefix = "quality.image.openshift.io"

// BlackDuckImageAnnotation describes the data model for image annotation
type BlackDuckImageAnnotation struct {
	policyViolationCount int
	vulnerabilityCount   int
	overallStatus        string
	componentsURL        string
	hubVersion           string
	scanClientVersion    string
}

// NewBlackDuckImageAnnotation creates a new BlackDuckImageAnnotation object
func NewBlackDuckImageAnnotation(policyViolationCount int, vulnerabilityCount int, overallStatus string, url string, hubVersion string, scVersion string) *BlackDuckImageAnnotation {
	return &BlackDuckImageAnnotation{
		policyViolationCount: policyViolationCount,
		vulnerabilityCount:   vulnerabilityCount,
		overallStatus:        overallStatus,
		componentsURL:        url,
		hubVersion:           hubVersion,
		scanClientVersion:    scVersion,
	}
}

// HasPolicyViolations returns true if the image has any policy violations
func (bdia *BlackDuckImageAnnotation) HasPolicyViolations() bool {
	return bdia.policyViolationCount > 0
}

// HasVulnerabilities returns true if the image has any vulnerabilities
func (bdia *BlackDuckImageAnnotation) HasVulnerabilities() bool {
	return bdia.vulnerabilityCount > 0
}

// GetVulnerabilityCount returns the number of image vulnerabilities
func (bdia *BlackDuckImageAnnotation) GetVulnerabilityCount() int {
	return bdia.vulnerabilityCount
}

// GetPolicyViolationCount returns the number of image policy violations
func (bdia *BlackDuckImageAnnotation) GetPolicyViolationCount() int {
	return bdia.policyViolationCount
}

// GetComponentsURL returns the image componenets URL
func (bdia *BlackDuckImageAnnotation) GetComponentsURL() string {
	return bdia.componentsURL
}

// GetOverallStatus returns the image overall status
func (bdia *BlackDuckImageAnnotation) GetOverallStatus() string {
	return bdia.overallStatus
}

// GetHubVersion returns the version of the hub that provided the information
func (bdia *BlackDuckImageAnnotation) GetHubVersion() string {
	return bdia.hubVersion
}

// GetScanClientVersion returns the version of the scan client used to scan the image
func (bdia *BlackDuckImageAnnotation) GetScanClientVersion() string {
	return bdia.scanClientVersion
}

// CreateImageLabels returns a map of labels from a BlackDuckImageAnnotation object
func CreateImageLabels(imageAnnotations *BlackDuckImageAnnotation, name string, count int) map[string]string {
	imagePostfix := ""
	labels := make(map[string]string)

	if len(name) > 0 {
		imagePostfix = fmt.Sprintf("%d", count)
		labels[fmt.Sprintf("com.blackducksoftware.image%d", count)] = strings.Replace(name, "/", ".", -1)
	}
	labels[fmt.Sprintf("com.blackducksoftware.image%s.policy-violations", imagePostfix)] = fmt.Sprintf("%d", imageAnnotations.GetPolicyViolationCount())
	labels[fmt.Sprintf("com.blackducksoftware.image%s.has-policy-violations", imagePostfix)] = fmt.Sprintf("%t", imageAnnotations.HasPolicyViolations())
	labels[fmt.Sprintf("com.blackducksoftware.image%s.vulnerabilities", imagePostfix)] = fmt.Sprintf("%d", imageAnnotations.GetVulnerabilityCount())
	labels[fmt.Sprintf("com.blackducksoftware.image%s.has-vulnerabilities", imagePostfix)] = fmt.Sprintf("%t", imageAnnotations.HasVulnerabilities())
	labels[fmt.Sprintf("com.blackducksoftware.image%s.overall-status", imagePostfix)] = imageAnnotations.GetOverallStatus()

	return labels
}

// CreateImageAnnotations returns a map of annotations from a BlackDuckImageAnnotation object
func CreateImageAnnotations(imageAnnotations *BlackDuckImageAnnotation, name string, count int) map[string]string {
	imagePrefix := ""
	newAnnotations := make(map[string]string)

	if len(name) > 0 {
		imagePrefix = fmt.Sprintf("image%d.", count)
		imageName := strings.Replace(name, "/", ".", -1)
		newAnnotations[fmt.Sprintf("%sblackducksoftware.com", imagePrefix)] = imageName
		newAnnotations[fmt.Sprintf("%s%s", imagePrefix, BDImageAnnotationPrefix)] = imageName
	}
	newAnnotations[fmt.Sprintf("%sblackducksoftware.com/hub-scanner-version", imagePrefix)] = imageAnnotations.GetScanClientVersion()
	newAnnotations[fmt.Sprintf("%sblackducksoftware.com/attestation-hub-server", imagePrefix)] = imageAnnotations.GetHubVersion()
	newAnnotations[fmt.Sprintf("%sblackducksoftware.com/project-endpoint", imagePrefix)] = imageAnnotations.GetComponentsURL()

	vulnAnnotations := CreateBlackDuckVulnerabilityAnnotation(imageAnnotations.HasVulnerabilities() == true, imageAnnotations.GetComponentsURL(), imageAnnotations.GetVulnerabilityCount())
	policyAnnotations := CreateBlackDuckPolicyAnnotation(imageAnnotations.HasPolicyViolations() == true, imageAnnotations.GetComponentsURL(), imageAnnotations.GetPolicyViolationCount())

	newAnnotations[fmt.Sprintf("%s%s/vulnerability.blackduck", imagePrefix, BDImageAnnotationPrefix)] = vulnAnnotations.AsString()
	newAnnotations[fmt.Sprintf("%s%s/policy.blackduck", imagePrefix, BDImageAnnotationPrefix)] = policyAnnotations.AsString()

	return newAnnotations
}
