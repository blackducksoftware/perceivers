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

package dumper

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/blackducksoftware/perceivers/docker/pkg/mapper"
	"github.com/blackducksoftware/perceivers/docker/pkg/metrics"
	"github.com/blackducksoftware/perceivers/pkg/communicator"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	log "github.com/sirupsen/logrus"

	dockerClient "github.com/blackducksoftware/perceivers/docker/pkg/docker"
)

// DockerDumper handles sending all pods to the perceptor periodically
type DockerDumper struct {
	cli        *dockerClient.Docker
	coreV1     corev1.CoreV1Interface
	allPodsURL string
}

// NewDockerDumper creates a new DockerDumper object
func NewDockerDumper(client *dockerClient.Docker, core corev1.CoreV1Interface, perceptorURL string) *DockerDumper {
	return &DockerDumper{
		cli:        client,
		coreV1:     core,
		allPodsURL: fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.AllPodsPath),
	}
}

// Run starts a controller that will send all pods to the perceptor periodically
func (pd *DockerDumper) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting docker service dumper controller")

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		// Get all the pods in the format perceptor uses
		swarmservices, err := pd.getAllServicesAsPerceptorPods()
		if err != nil {
			metrics.RecordError("swarm_service_dumper", "unable to get all swarm services")
			log.Errorf("unable to get all swarm services: %v", err)
			continue
		}
		log.Infof("about to PUT all swarm services -- found %d services", len(swarmservices))

		jsonBytes, err := json.Marshal(perceptorapi.NewAllPods(swarmservices))
		if err != nil {
			metrics.RecordError("swarm_service_dumper", "unable to serialize all pods")
			log.Errorf("unable to serialize all pods: %v", err)
			continue
		}

		// Send all the swarm service information to the perceptor
		err = communicator.SendPerceptorData(pd.allPodsURL, jsonBytes)
		metrics.RecordHTTPStats(pd.allPodsURL, err == nil)
		if err != nil {
			metrics.RecordError("swarm_service_dumper", "unable to send services")
			log.Errorf("failed to send services: %v", err)
		} else {
			log.Infof("http POST request to %s succeeded", pd.allPodsURL)
		}
	}
}

func (pd *DockerDumper) getAllServicesAsPerceptorPods() ([]perceptorapi.Pod, error) {
	perceptorPods := []perceptorapi.Pod{}

	// Get all pods from kubernetes
	getServicesStart := time.Now()
	swarmServices, err := pd.cli.ListServices()
	metrics.RecordDuration("get swarm services", time.Now().Sub(getServicesStart))
	if err != nil {
		return nil, err
	}

	// Translate the pods from kubernetes to perceptor format
	for _, swarmService := range swarmServices {
		perceptorPod, err := mapper.NewPerceptorPodFromSwarmServices(swarmService)
		if err != nil {
			metrics.RecordError("swarm_service_dumper", "unable to convert swarm service to perceptor pod")
			continue
		}
		perceptorPods = append(perceptorPods, *perceptorPod)
	}
	return perceptorPods, nil
}
