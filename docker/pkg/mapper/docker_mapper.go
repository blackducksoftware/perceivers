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

package mapper

import (
	"fmt"
	"log"

	"github.com/blackducksoftware/perceivers/pkg/docker"
	"github.com/docker/docker/api/types/swarm"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	metrics "github.com/blackducksoftware/perceivers/docker/pkg/metrics"
)

// NewPerceptorPodFromSwarmServices will convert a Swarm Service object to a
// perceptor pod object
func NewPerceptorPodFromSwarmServices(swarmService swarm.Service) (*perceptorapi.Pod, error) {
	containers := []perceptorapi.Container{}

	if len(swarmService.ID) > 0 {
		imageName := swarmService.Spec.TaskTemplate.ContainerSpec.Image
		log.Printf("Converting Swarm service image %s to Perceptor pod", imageName)
		name, tag, sha, err := docker.ParseDockerSwarmImageString(imageName)
		serviceName := swarmService.Spec.Name
		log.Printf("Service name: %s, Name: %s, tag: %s, sha: %s, err: %v", serviceName, name, tag, sha, err)
		if err != nil {
			metrics.RecordError("swarm_service_mapper", "unable to parse docker swarm imageId")
			return nil, fmt.Errorf("unable to parse docker swarm imageId string %s from service %s: %v", imageName, serviceName, err)
		}
		image := fmt.Sprintf("%s:%s", name, tag)
		addedCont := perceptorapi.NewContainer(*perceptorapi.NewImage(name, sha, image), serviceName)
		containers = append(containers, *addedCont)
	} else {
		metrics.RecordError("swarm_service_mapper", "empty docker swarm imageId")
		return nil, fmt.Errorf("empty docker swarm imageId from service %s, id %s", swarmService.Spec.Name, swarmService.ID)
	}
	return perceptorapi.NewPod(swarmService.Spec.Name, swarmService.ID, "perceptor", containers), nil
}
