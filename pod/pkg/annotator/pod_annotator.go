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
	"time"

	bdannotations "github.com/blackducksoftware/perceivers/pkg/annotations"
	"github.com/blackducksoftware/perceivers/pkg/docker"
	"github.com/blackducksoftware/perceivers/pkg/utils"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	log "github.com/sirupsen/logrus"
)

// PodAnnotator handles annotating pods with vulnerability and policy issues
type PodAnnotator struct {
	coreV1         corev1.CoreV1Interface
	scanResultsURL string
}

// NewPodAnnotator creates a new PodAnnotator object
func NewPodAnnotator(pl corev1.CoreV1Interface, perceptorURL string) *PodAnnotator {
	return &PodAnnotator{
		coreV1:         pl,
		scanResultsURL: fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ScanResultsPath),
	}
}

// Run starts a controller that will annotate pods
func (pa *PodAnnotator) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting pod annotator controller")

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		// Get all the scan results from the Perceptor
		log.Infof("attempting to GET %s for pod annotation", pa.scanResultsURL)
		resp, err := http.Get(pa.scanResultsURL)
		if err != nil {
			log.Errorf("unable to GET %s for pod annotation: %v", pa.scanResultsURL, err)
			continue
		}
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("unable to read resp body from %s: %v", pa.scanResultsURL, err)
			continue
		}

		// Process the scan results and apply annotations/labels to pods
		var scanResults perceptorapi.ScanResults
		err = json.Unmarshal(bodyBytes, &scanResults)
		if err == nil && resp.StatusCode == 200 {
			log.Infof("GET to %s succeeded, about to update annotations on all pods", pa.scanResultsURL)
			for _, pod := range scanResults.Pods {
				podAnnotations := bdannotations.NewBlackDuckPodAnnotation(pod.PolicyViolations, pod.Vulnerabilities, pod.OverallStatus)
				if err = pa.setAnnotationsOnPod(pod.Name, pod.Namespace, podAnnotations, scanResults.Images); err != nil {
					log.Errorf("failed to annotate pod %s/%s: %v", pod.Namespace, pod.Name, err)
				}
			}
		} else {
			log.Errorf("unable to Unmarshal ScanResults from url %s: %v", pa.scanResultsURL, err)
		}
	}
}

func (pa *PodAnnotator) setAnnotationsOnPod(name string, ns string, bdPodAnnotations *bdannotations.BlackDuckPodAnnotation, images []perceptorapi.ScannedImage) error {
	podName := fmt.Sprintf("%s:%s", ns, name)
	kubePod, err := pa.coreV1.Pods(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get pod %s: %v", podName, err)
	}

	// Get the list of annotations currently on the pod
	currentAnnotations := kubePod.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = map[string]string{}
	}

	// Get the list of labels currently on the pod
	currentLabels := kubePod.GetLabels()
	if currentLabels == nil {
		currentLabels = map[string]string{}
	}

	// Generate the annotations and labels that should be on the pod
	newAnnotations := bdannotations.CreatePodAnnotations(bdPodAnnotations)
	newLabels := bdannotations.CreatePodLabels(bdPodAnnotations)

	// Look at all the images in the pod and see if there are scan results
	// for that image.  If there are then create annotations for the image that
	// will be applied to the pod
	for cnt, newCont := range kubePod.Status.ContainerStatuses {
		name, sha, err := docker.ParseImageIDString(newCont.ImageID)
		if err != nil {
			log.Errorf("unable to parse kubernetes imageID string %s from pod %s/%s: %v", newCont.ImageID, kubePod.Namespace, kubePod.Name, err)
			continue
		}
		imageScanResults := pa.findImageAnnotations(name, sha, images)
		if imageScanResults != nil {
			imageAnnotations := pa.createImageAnnotationsFromImageScanResults(imageScanResults)
			newAnnotations = utils.MapMerge(newAnnotations, bdannotations.CreateImageAnnotations(imageAnnotations, name, cnt))
			newLabels = utils.MapMerge(newLabels, bdannotations.CreateImageLabels(imageAnnotations, name, cnt))
		}
	}

	// Apply updated annotations to the pod if the existing annotations don't
	// contain the expected entries
	updatePod := false
	if !bdannotations.MapContainsBlackDuckEntries(currentAnnotations, newAnnotations) {
		log.Infof("annotations are missing or incorrect on pod %s.  Expected %v to contain %v", podName, currentAnnotations, newAnnotations)
		kubePod.SetAnnotations(utils.MapMerge(currentAnnotations, newAnnotations))
		updatePod = true
	}

	// Apply updated labels to the pod if the existing labels don't
	// contain the expected entries
	if !bdannotations.MapContainsBlackDuckEntries(currentLabels, newLabels) {
		log.Infof("labels are missing or incorrect on pod %s.  Expected %v to contain %v", podName, currentLabels, newLabels)
		kubePod.SetLabels(utils.MapMerge(currentLabels, newLabels))
		updatePod = true
	}

	// Update the pod if any label or annotation isn't correct
	if updatePod {
		_, err = pa.coreV1.Pods(ns).Update(kubePod)
		if err != nil {
			return fmt.Errorf("unable to update annotations/labels for pod %s: %v", podName, err)
		}
		log.Infof("successfully annotated pod %s", podName)
	}

	return nil
}

func (pa *PodAnnotator) findImageAnnotations(imageName string, imageSha string, imageList []perceptorapi.ScannedImage) *perceptorapi.ScannedImage {
	for _, image := range imageList {
		if image.Name == imageName && image.Sha == imageSha {
			return &image
		}
	}

	return nil
}

func (pa *PodAnnotator) createImageAnnotationsFromImageScanResults(scannedImage *perceptorapi.ScannedImage) *bdannotations.BlackDuckImageAnnotation {
	return bdannotations.NewBlackDuckImageAnnotation(scannedImage.PolicyViolations,
		scannedImage.Vulnerabilities, scannedImage.OverallStatus, scannedImage.ComponentsURL)
}
