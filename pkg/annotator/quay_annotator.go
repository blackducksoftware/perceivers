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
	"strings"
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
	quayBDComURL = "blackduck.componentsurl"
)

// QuayAnnotator handles annotating quay images with vulnerability and policy issues
type QuayAnnotator struct {
	scanResultsURL  string
	registryAuths   []*utils.RegistryAuth
	quayAccessToken string
}

// NewQuayAnnotator creates a new QuayAnnotator object
func NewQuayAnnotator(perceptorURL string, registryAuths []*utils.RegistryAuth, quayAccessToken string) *QuayAnnotator {
	return &QuayAnnotator{
		scanResultsURL:  fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ScanResultsPath),
		registryAuths:   registryAuths,
		quayAccessToken: quayAccessToken,
	}
}

// Run starts a controller that will annotate images
func (qa *QuayAnnotator) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting quay annotation controller")

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

// This method tries to annotate all the images
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

// This method gets the scan results from perceptor and tries to unmarshal it
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

// This method tries to annotate all the Images found in BD by matching their SHAs
func (qa *QuayAnnotator) addAnnotationsToImages(results perceptorapi.ScanResults) {
	regs := 0
	imgs := 0

	for _, registry := range qa.registryAuths {
		auth, err := utils.PingQuayServer("https://"+registry.URL, qa.quayAccessToken)

		if err != nil {
			log.Debugf("Annotator: URL %s either not a valid quay repository or incorrect token: %e", registry.URL, err)
			continue
		}

		regs = regs + 1
		for _, image := range results.Images {

			// The base URL may contain something in their instance/registry, splitting has no loss
			if !strings.Contains(image.Repository, strings.Split(registry.URL, "/")[0]) {
				continue
			}

			repoSlice := strings.Split(image.Repository, "/")[1:]
			repo := strings.Join(repoSlice, "/")
			labelList := &utils.QuayLabels{}
			// Look for SHA
			url := fmt.Sprintf("%s/api/v1/repository/%s/manifest/%s/labels", auth.URL, repo, fmt.Sprintf("sha256:%s", image.Sha))
			log.Infof("Getting labels from: %s", url)
			err = utils.GetResourceOfType(url, nil, auth.Password, labelList)
			if err != nil {
				log.Errorf("Error in getting labels for repo %s: %e", repo, err)
				continue
			}

			imgs = imgs + 1

			// Create a map of BD tags and retrieved values
			nt := make(map[string]string)
			nt[quayBDComURL] = image.ComponentsURL
			nt[quayBDPolicy] = fmt.Sprintf("%d", image.PolicyViolations)
			nt[quayBDSt] = image.OverallStatus
			nt[quayBDVuln] = fmt.Sprintf("%d", image.Vulnerabilities)

			// Create a map of Quay tags and retrieved values
			ot := make(map[string]string)
			for _, label := range labelList.Labels {
				ot[label.Key] = label.Value
			}

			// Merge them with updated BD values
			tags := utils.MapMerge(ot, nt)
			for key, value := range tags {
				// Don't need to touch other tags apart form BD ones
				if _, ok := nt[key]; ok {
					imageInfo := fmt.Sprintf("%s:%s with SHA %s", image.Repository, image.Tag, image.Sha)
					qa.UpdateAnnotation(url, key, value, imageInfo)
				}
			}

		}

		log.Infof("Total scanned images in Quay with URL %s: %d", registry.URL, imgs)
	}

	log.Infof("Total valid Quay Registries: %d", regs)
}

// UpdateAnnotation takes the specific Quay URL and applies the properties/annotations given by BD
func (qa *QuayAnnotator) UpdateAnnotation(url string, labelKey string, newValue string, imageInfo string) {

	filterURL := fmt.Sprintf("%s?filter=%s", url, labelKey)
	labelList := &utils.QuayLabels{}
	err := utils.GetResourceOfType(filterURL, nil, qa.quayAccessToken, labelList)
	if err != nil {
		log.Errorf("Error in getting labels at URL %s for update: %e", url, err)
		return
	}

	for _, label := range labelList.Labels {
		deleteURL := fmt.Sprintf("%s/%s", url, label.ID)
		err = utils.DeleteQuayLabel(deleteURL, qa.quayAccessToken, label.ID)
		if err != nil {
			log.Errorf("Error in deleting label %s at URL %s: %e", label.Key, deleteURL, err)
			log.Errorf("Images may contain duplicate labels!")
		}
	}

	err = utils.AddQuayLabel(url, qa.quayAccessToken, labelKey, newValue)
	if err != nil {
		log.Errorf("Error in adding label %s at URL %s after deleting: %e", labelKey, url, err)
		return
	}

	labelInfo := fmt.Sprintf("%s:%s", labelKey, newValue)
	log.Infof("Successfully annotated %s with %s!", imageInfo, labelInfo)
}
