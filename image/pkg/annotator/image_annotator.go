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
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	bdannotations "github.com/blackducksoftware/perceivers/pkg/annotations"
	"github.com/blackducksoftware/perceivers/pkg/docker"
	"github.com/blackducksoftware/perceivers/pkg/utils"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		resp, err := http.Get(ia.scanResultsURL)
		if err != nil {
			log.Errorf("unable to GET %s for image annotation: %v", ia.scanResultsURL, err)
			continue
		}
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("unable to read resp body from %s: %v", ia.scanResultsURL, err)
			continue
		}

		// Process the scan results and apply annotations/labels to images
		var scanResults perceptorapi.ScanResults
		err = json.Unmarshal(bodyBytes, &scanResults)
		if err == nil && resp.StatusCode == 200 {
			log.Infof("GET to %s succeeded, about to update annotations on all images", ia.scanResultsURL)
			for _, image := range scanResults.Images {
				imageAnnotations := bdannotations.NewBlackDuckImageAnnotation(image.PolicyViolations, image.Vulnerabilities, image.OverallStatus, image.ComponentsURL, scanResults.HubVersion, scanResults.HubScanClientVersion)
				if err = ia.setAnnotationsOnImage(image.Name, image.Sha, imageAnnotations); err != nil {
					log.Errorf("failed to annotate image %s@sha256%s: %v", image.Name, image.Sha, err)
				}
			}
		} else {
			log.Errorf("unable to unmarshal ScanResults from url %s: %v", ia.scanResultsURL, err)
		}
	}
}

func (ia *ImageAnnotator) setAnnotationsOnImage(name string, sha string, bdImageAnnotations *bdannotations.BlackDuckImageAnnotation) error {
	var imageName string
	getName := fmt.Sprintf("sha256:%s", sha)
	fullImageName := fmt.Sprintf("%s@%s", name, getName)

	nameStart := strings.LastIndex(name, "/") + 1
	if nameStart >= 0 {
		imageName = name[nameStart:]
	} else {
		imageName = name
	}

	// Get the image
	image, err := ia.client.Images().Get(getName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// This isn't an image in openshift
		return nil
	} else if err != nil {
		// Some other kind of error, possibly couldn't communicate, so return
		// an error
		return fmt.Errorf("unexpected error retrieving image %s: %v", fullImageName, err)
	}

	// Verify the sha of the scanned image matches that of the image we retrieved
	_, imageSha, err := docker.ParseImageIDString(image.DockerImageReference)
	if err != nil {
		return fmt.Errorf("unable to parse openshift imageID from image %s: %v", imageName, err)
	}
	if imageSha != sha {
		return fmt.Errorf("image sha doesn't match for image %s.  Got %s, expected %s", imageName, sha, imageSha)
	}

	// Get existing annotations on the image
	currentAnnotations := image.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = map[string]string{}
	}

	// Get existing labels on the image
	currentLabels := image.GetLabels()
	if currentLabels == nil {
		currentLabels = map[string]string{}
	}

	// Generate the annotations and labels that should be on the image
	newLabels := bdannotations.CreateImageLabels(bdImageAnnotations, "", 0)
	newAnnotations := bdannotations.CreateImageAnnotations(bdImageAnnotations, "", 0)

	// Apply updated annotations to the image if the existing annotations don't
	// contain the expected entries
	updateImage := false
	if !bdannotations.MapContainsBlackDuckEntries(currentAnnotations, newAnnotations) {
		log.Infof("annotations are missing or incorrect on image %s.  Expected %v to contain %v", fullImageName, currentAnnotations, newAnnotations)
		image.SetAnnotations(utils.MapMerge(currentAnnotations, newAnnotations))
		updateImage = true
	}

	// Apply updated labels to the image if the existing annotations don't
	// contain the expected entries
	if !bdannotations.MapContainsBlackDuckEntries(currentLabels, newLabels) {
		log.Infof("labels are missing or incorrect on image %s.  Expected %v to contain %v", fullImageName, currentLabels, newLabels)
		image.SetLabels(utils.MapMerge(currentLabels, newLabels))
		updateImage = true
	}

	// Update theimage if any label or annotation isn't correct
	if updateImage {
		_, err = ia.client.Images().Update(image)
		if err != nil {
			return fmt.Errorf("unable to update annotations/labels for image %s: %v", fullImageName, err)
		}
		log.Infof("successfully annotated image %s", fullImageName)
	}

	return nil
}
