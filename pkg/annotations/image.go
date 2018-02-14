package annotations

import (
	"fmt"
	"strings"
)

// BlackDuckImageAnnotation describes the data model for image annotation
type BlackDuckImageAnnotation struct {
	policyViolationCount int
	vulnerabilityCount   int
	overallStatus        string
	componentsURL        string
}

// NewBlackDuckImageAnnotation creates a new BlackDuckImageAnnotation object
func NewBlackDuckImageAnnotation(policyViolationCount int, vulnerabilityCount int, overallStatus string, url string) *BlackDuckImageAnnotation {
	return &BlackDuckImageAnnotation{
		policyViolationCount: policyViolationCount,
		vulnerabilityCount:   vulnerabilityCount,
		overallStatus:        overallStatus,
		componentsURL:        url,
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
		newAnnotations[fmt.Sprintf("%sblackducksoftware.com", imagePrefix)] = strings.Replace(name, "/", ".", -1)
		newAnnotations[fmt.Sprintf("%squality.image.openshift.io", imagePrefix)] = strings.Replace(name, "/", ".", -1)
	}
	/*
		newAnnotations[fmt.Sprintf("%sblackducksoftware.com/hub-scanner-version", imagePrefix] = imageAnnotations.GetScannerVersion()
		newAnnotations[fmt.Sprintf("%sblackducksoftware.com/attestation-hub-server", imagePrefix] = imageAnnotations.GetHubServer()
	*/
	newAnnotations[fmt.Sprintf("%sblackducksoftware.com/project-endpoint", imagePrefix)] = imageAnnotations.GetComponentsURL()
	/*
		if len(imageAnnotations.GetScanID()) > 0 {
			newAnnotations[fmt.Sprintf("blackducksoftware.com/%sscan-id", imagePostfix)] = imageAnnotations.GetScanID()
		}
	*/

	vulnAnnotations := CreateBlackDuckVulnerabilityAnnotation(imageAnnotations.HasVulnerabilities() == true, imageAnnotations.GetVulnerabilityCount())
	policyAnnotations := CreateBlackDuckPolicyAnnotation(imageAnnotations.HasPolicyViolations() == true, imageAnnotations.GetPolicyViolationCount())

	newAnnotations[fmt.Sprintf("%squality.image.openshift.io/vulnerability.blackduck", imagePrefix)] = vulnAnnotations.AsString()
	newAnnotations[fmt.Sprintf("%squality.image.openshift.io/policy.blackduck", imagePrefix)] = policyAnnotations.AsString()

	return newAnnotations
}
