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

package annotator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	bdannotations "github.com/blackducksoftware/perceivers/pkg/annotations"
	"github.com/blackducksoftware/perceivers/pkg/communicator"
	"github.com/blackducksoftware/perceivers/pkg/docker"
	"github.com/blackducksoftware/perceivers/pkg/utils"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/api/image/v1"

	imageclient "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"

	log "github.com/sirupsen/logrus"
)

// ImageAnnotator handles annotating images with vulnerability and policy issues
type ImageAnnotator struct {
	client         *imageclient.ImageV1Client
	scanResultsURL string
}

// NewImageAnnotator creates a new ImageAnnotator object
func NewImageAnnotator(ic *imageclient.ImageV1Client, perceptorURL string) *ImageAnnotator {
	return &ImageAnnotator{
		client:         ic,
		scanResultsURL: fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ScanResultsPath),
	}
}

// Run starts a controller that will annotate images
func (ia *ImageAnnotator) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting image annotator controller")

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		// Get all the scan results from the Perceptor
		log.Infof("attempting to GET %s for image annotation", ia.scanResultsURL)
		scanResults, err := ia.getScanResults()
		if err == nil {
			log.Errorf("error getting scan results: %v", err)
			continue
		}

		// Process the scan results and apply annotations/labels to images
		log.Infof("GET to %s succeeded, about to update annotations on all images", ia.scanResultsURL)
		ia.addAnnotationsToImages(*scanResults)
	}
}

func (ia *ImageAnnotator) getScanResults() (*perceptorapi.ScanResults, error) {
	var results perceptorapi.ScanResults

	bytes, err := communicator.GetPerceptorScanResults(ia.scanResultsURL)
	if err != nil {
		return nil, fmt.Errorf("unable to get scan results: %v", err)
	}

	err = json.Unmarshal(bytes, &results)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal ScanResults from url %s: %v", ia.scanResultsURL, err)
	}

	return &results, nil
}

func (ia *ImageAnnotator) addAnnotationsToImages(results perceptorapi.ScanResults) {
	for _, image := range results.Images {
		var imageName string
		getName := fmt.Sprintf("sha256:%s", image.Sha)
		fullImageName := fmt.Sprintf("%s@%s", image.Name, getName)

		nameStart := strings.LastIndex(image.Name, "/") + 1
		if nameStart >= 0 {
			imageName = image.Name[nameStart:]
		} else {
			imageName = image.Name
		}

		// Get the image
		osImage, err := ia.client.Images().Get(getName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			// This isn't an image in openshift
			continue
		} else if err != nil {
			// Some other kind of error, possibly couldn't communicate, so return
			// an error
			log.Errorf("unexpected error retrieving image %s: %v", fullImageName, err)
			continue
		}

		// Verify the sha of the scanned image matches that of the image we retrieved
		_, imageSha, err := docker.ParseImageIDString(osImage.DockerImageReference)
		if err != nil {
			log.Errorf("unable to parse openshift imageID from image %s: %v", imageName, err)
			continue
		}
		if imageSha != image.Sha {
			log.Errorf("image sha doesn't match for image %s.  Got %s, expected %s", imageName, image.Sha, imageSha)
			continue
		}

		imageAnnotations := bdannotations.NewBlackDuckImageAnnotation(image.PolicyViolations, image.Vulnerabilities, image.OverallStatus, image.ComponentsURL, results.HubVersion, results.HubScanClientVersion)

		// Update the image if any label or annotation isn't correct
		if ia.addImageAnnotations(fullImageName, osImage, imageAnnotations) ||
			ia.addImageLabels(fullImageName, osImage, imageAnnotations) {
			_, err = ia.client.Images().Update(osImage)
			if err != nil {
				log.Errorf("unable to update annotations/labels for image %s: %v", fullImageName, err)
			} else {
				log.Infof("successfully annotated image %s", fullImageName)
			}
		}
	}
}

func (ia *ImageAnnotator) addImageAnnotations(name string, image *v1.Image, imageAnnotations *bdannotations.BlackDuckImageAnnotation) bool {
	// Get existing annotations on the image
	currentAnnotations := image.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = map[string]string{}
	}

	// Generate the annotations that should be on the image
	newAnnotations := bdannotations.CreateImageAnnotations(imageAnnotations, "", 0)

	// Apply updated annotations to the image if the existing annotations don't
	// contain the expected entries
	if !bdannotations.MapContainsBlackDuckEntries(currentAnnotations, newAnnotations) {
		log.Infof("annotations are missing or incorrect on image %s.  Expected %v to contain %v", name, currentAnnotations, newAnnotations)
		image.SetAnnotations(utils.MapMerge(currentAnnotations, newAnnotations))
		return true
	}
	return false
}

func (ia *ImageAnnotator) addImageLabels(name string, image *v1.Image, imageAnnotations *bdannotations.BlackDuckImageAnnotation) bool {
	// Get existing labels on the image
	currentLabels := image.GetLabels()
	if currentLabels == nil {
		currentLabels = map[string]string{}
	}

	// Generate the labels that should be on the image
	newLabels := bdannotations.CreateImageLabels(imageAnnotations, "", 0)

	// Apply updated labels to the image if the existing annotations don't
	// contain the expected entries
	if !bdannotations.MapContainsBlackDuckEntries(currentLabels, newLabels) {
		log.Infof("labels are missing or incorrect on image %s.  Expected %v to contain %v", name, currentLabels, newLabels)
		image.SetLabels(utils.MapMerge(currentLabels, newLabels))
		return true
	}

	return false
}
