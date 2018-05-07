/*
Copyright (C) 2018 Synopsys, Inc.

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

	"github.com/blackducksoftware/perceivers/docker/pkg/metrics"
	"github.com/blackducksoftware/perceivers/pkg/annotations"
	"github.com/blackducksoftware/perceivers/pkg/communicator"
	"github.com/blackducksoftware/perceivers/pkg/docker"
	"github.com/docker/docker/api/types/swarm"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	dockerClient "github.com/blackducksoftware/perceivers/docker/pkg/docker"
	log "github.com/sirupsen/logrus"
)

// PodAnnotator handles annotating pods with vulnerability and policy issues
type DockerAnnotator struct {
	scanResultsURL string
	h              annotations.PodAnnotatorHandler
	client         *dockerClient.Docker
}

// NewDockerAnnotator creates a new PodAnnotator object
func NewDockerAnnotator(client *dockerClient.Docker, perceptorURL string, handler annotations.PodAnnotatorHandler) *DockerAnnotator {
	return &DockerAnnotator{
		scanResultsURL: fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ScanResultsPath),
		h:              handler,
		client:         client,
	}
}

// Run starts a controller that will annotate pods
func (pa *DockerAnnotator) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting swarm service docker_annotator controller")

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		err := pa.annotate()
		if err != nil {
			log.Errorf("failed to annotate swarm services: %v", err)
		}
	}
}

func (sa *DockerAnnotator) annotate() error {
	// Get all the scan results from the Perceptor
	log.Infof("attempting to get scan results with GET %s for swarm service annotation", sa.scanResultsURL)
	scanResults, err := sa.getScanResults()
	if err != nil {
		metrics.RecordError("swarm_service_annotator", "error getting scan results")
		return fmt.Errorf("error getting scan results: %v", err)
	}

	// Process the scan results and apply annotations/labels to swarm services
	log.Infof("GET to %s succeeded, about to update annotations on all swarm services", sa.scanResultsURL)
	sa.addAnnotationsToServices(*scanResults)
	return nil
}

func (sa *DockerAnnotator) getScanResults() (*perceptorapi.ScanResults, error) {
	var results perceptorapi.ScanResults

	bytes, err := communicator.GetPerceptorScanResults(sa.scanResultsURL)
	if err != nil {
		metrics.RecordError("swarm_service_annotator", "unable to get scan results")
		return nil, fmt.Errorf("unable to get scan results: %v", err)
	}

	err = json.Unmarshal(bytes, &results)
	if err != nil {
		metrics.RecordError("swarm_service_annotator", "unable to Unmarshal ScanResults")
		return nil, fmt.Errorf("unable to Unmarshal ScanResults from url %s: %v", sa.scanResultsURL, err)
	}

	return &results, nil
}

func (sa *DockerAnnotator) addAnnotationsToServices(results perceptorapi.ScanResults) {
	for _, service := range results.Pods {
		serviceId := fmt.Sprintf("%s", service.Name)
		getServiceStart := time.Now()

		swarmService, err := sa.client.GetServices(serviceId)
		if err != nil {
			metrics.RecordError("swarm_service_annotator", "unable to get service")
			log.Errorf("Unable to get %s service because %v \n", serviceId, err)
			continue
		}
		metrics.RecordDuration("get service", time.Now().Sub(getServiceStart))

		serviceAnnotations := annotations.NewImageAnnotationData(service.PolicyViolations, service.Vulnerabilities, service.OverallStatus, "", results.HubVersion, results.HubScanClientVersion)

		sa.addImageLabels(swarmService, serviceAnnotations)
	}
}

func (sa *DockerAnnotator) addImageLabels(swarmService *swarm.Service, imageAnnotations *annotations.ImageAnnotationData) {

	// Get the Swarm service name
	serviceName := swarmService.Spec.Name
	// Get the list of labels that is currently on the service
	getLabelsStart := time.Now()
	currentLabels := swarmService.Spec.TaskTemplate.ContainerSpec.Labels
	log.Infof("Started labelled service %s with labels %v", serviceName, currentLabels)
	metrics.RecordDuration("get service labels", time.Now().Sub(getLabelsStart))
	if currentLabels == nil {
		currentLabels = map[string]string{}
	}

	name, _, _, _ := docker.ParseDockerSwarmImageString(swarmService.Spec.TaskTemplate.ContainerSpec.Image)
	// Generate the labels that should be on the image
	newLabels := annotations.CreateImageLabels(imageAnnotations, name, 0)
	log.Infof("Comparing labelled service %s with new labels %v", serviceName, newLabels)
	if sa.h.CompareMaps(currentLabels, newLabels) {
		setLabelsStart := time.Now()
		log.Infof("Started labelled service %s", serviceName)
		err := sa.client.UpdateServices(swarmService, newLabels)
		metrics.RecordDuration("update services", time.Now().Sub(setLabelsStart))
		if err != nil {
			metrics.RecordError("swarm_service_annotator", "unable to update annotations/labels for service")
			log.Errorf("unable to update annotations/labels for service %s: %v", serviceName, err)
		} else {
			metrics.RecordSwarmServiceAnnotation("swarm_service_annotator", serviceName)
			log.Infof("successfully labelled service %s", serviceName)
		}
	}
}
