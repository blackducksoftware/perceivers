/*
Copyright (C) 2019 Synopsys, Inc.

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

package annotator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/blackducksoftware/perceivers/pkg/communicator"
	"github.com/blackducksoftware/perceivers/pkg/metrics"
	"github.com/blackducksoftware/perceivers/pkg/utils"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

// BlackDuck Annotation names
const (
	quayBDPolicy = "blackduck.policyviolations"
	quayBDVuln   = "blackduck.vulnerabilities"
	quayBDSt     = "blackduck.overallstatus"
	quayBDComp   = "blackduck.componentsurl"
)

// QuayAnnotator handles annotating quay images with vulnerability and policy issues
type QuayAnnotator struct {
	scanResultsURL string
	registryAuths  []*utils.RegistryAuth
}

// NewQuayAnnotator creates a new ArtifactoryAnnotator object
func NewQuayAnnotator(perceptorURL string, registryAuths []*utils.RegistryAuth) *QuayAnnotator {
	return &QuayAnnotator{
		scanResultsURL: fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ScanResultsPath),
		registryAuths:  registryAuths,
	}
}

// Run starts a controller that will annotate images
func (qa *QuayAnnotator) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting quay annotator controller")

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		err := qa.annotate()
		if err != nil {
			log.Errorf("failed to annotate quay images: %v", err)
		}
	}
}

func (qa *QuayAnnotator) annotate() error {
	// Get all the scan results from the Perceptor
	log.Infof("attempting to GET %s for quay image annotation", qa.scanResultsURL)
	scanResults, err := qa.getScanResults()
	if err != nil {
		metrics.RecordError("quay_annotator", "error getting scan results")
		return fmt.Errorf("error getting scan results: %v", err)
	}

	// Process the scan results and apply annotations/labels to images
	log.Infof("GET to %s succeeded, about to update annotations on all quay images", qa.scanResultsURL)
	qa.addAnnotationsToImages(*scanResults)
	return nil
}

func (qa *QuayAnnotator) getScanResults() (*perceptorapi.ScanResults, error) {
	var results perceptorapi.ScanResults

	bytes, err := communicator.GetPerceptorScanResults(qa.scanResultsURL)
	if err != nil {
		metrics.RecordError("quay_annotator", "unable to get scan results")
		return nil, fmt.Errorf("unable to get scan results: %v", err)
	}

	err = json.Unmarshal(bytes, &results)
	if err != nil {
		metrics.RecordError("quay_annotator", "unable to Unmarshal ScanResults")
		return nil, fmt.Errorf("unable to Unmarshal ScanResults from url %s: %v", qa.scanResultsURL, err)
	}

	return &results, nil
}

func (qa *QuayAnnotator) addAnnotationsToImages(results perceptorapi.ScanResults) {

}

// AnnotateImage takes the specific Quay URL and applies the properties/annotations given by BD
func (qa *QuayAnnotator) AnnotateImage(uri string, im *perceptorapi.ScannedImage, cred *utils.RegistryAuth) {

}
