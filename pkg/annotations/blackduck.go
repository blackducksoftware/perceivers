package annotations

import (
	"encoding/json"
	"fmt"
	"time"
)

// BlackDuckAnnotation create annotations that correspong to the
// Openshift Containr Security guide (https://people.redhat.com/aweiteka/docs/preview/20170510/security/container_content.html)
type BlackDuckAnnotation struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Timestamp   time.Time           `json:"timestamp"`
	Reference   string              `json:"reference"`
	Compliant   bool                `json:"compliant"`
	Summary     []map[string]string `json:"summary"`
}

// AsString makes a map corresponding to the Openshift
// Container Security guide (https://people.redhat.com/aweiteka/docs/preview/20170510/security/container_content.html)
func (bda *BlackDuckAnnotation) AsString() string {
	m := make(map[string]string)
	m["name"] = bda.Name
	m["description"] = bda.Description
	m["timestamp"] = fmt.Sprintf("%v", bda.Timestamp)
	if len(bda.Reference) > 0 {
		m["reference"] = bda.Reference
	}
	m["compliant"] = fmt.Sprintf("%v", bda.Compliant)
	m["summary"] = fmt.Sprintf("%s", bda.Summary)
	mp, _ := json.Marshal(m)
	return string(mp)
}

// CreateBlackDuckVulnerabilityAnnotation returns an annotation containing
// vulnerabilities
func CreateBlackDuckVulnerabilityAnnotation(hasVulns bool, url string, vulnCount int) *BlackDuckAnnotation {
	return &BlackDuckAnnotation{
		"blackducksoftware",
		"Vulnerability Info",
		time.Now(),
		url,
		!hasVulns, // no vunls -> compliant.
		[]map[string]string{
			{
				"label":         "high",
				"score":         fmt.Sprintf("%d", vulnCount),
				"severityIndex": fmt.Sprintf("%v", 1),
			},
		},
	}
}

// CreateBlackDuckPolicyAnnotation returns an annotation containing
// policy violations
func CreateBlackDuckPolicyAnnotation(hasPolicyViolations bool, url string, policyCount int) *BlackDuckAnnotation {
	return &BlackDuckAnnotation{
		"blackducksoftware",
		"Policy Info",
		time.Now(),
		url,
		!hasPolicyViolations, // no violations -> compliant
		[]map[string]string{
			{
				"label":         "important",
				"score":         fmt.Sprintf("%d", policyCount),
				"severityIndex": fmt.Sprintf("%v", 1),
			},
		},
	}
}
