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

package dumper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blackducksoftware/perceivers/pod/pkg/mapper"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	log "github.com/sirupsen/logrus"
)

// PodDumper handles sending all pods to the perceptor periodically
type PodDumper struct {
	coreV1     corev1.CoreV1Interface
	allPodsURL string
}

// NewPodDumper creates a new PodDumper object
func NewPodDumper(core corev1.CoreV1Interface, perceptorURL string) *PodDumper {
	return &PodDumper{
		coreV1:     core,
		allPodsURL: fmt.Sprintf("%s%s", perceptorURL, perceptorapi.AllPodsPath),
	}
}

// Run starts a controller that will send all pods to the perceptor periodically
func (pd *PodDumper) Run(interval time.Duration, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		// Get all the pods in the format perceptor uses
		pods, err := pd.getAllPodsAsPerceptorPods()
		if err != nil {
			log.Errorf("unable to get all pods: %v", err)
			continue
		}
		log.Infof("about to PUT all pods -- found %d pods", len(pods))

		jsonBytes, err := json.Marshal(perceptorapi.NewAllPods(pods))
		if err != nil {
			log.Errorf("unable to serialize all pods: %v", err)
			continue
		}

		// Send all the pod information to the perceptor
		req, err := http.NewRequest("PUT", pd.allPodsURL, bytes.NewBuffer(jsonBytes))
		if err != nil {
			log.Errorf("unable to create PUT request for %s: %v", pd.allPodsURL, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Errorf("unable to PUT to %s: %v", pd.allPodsURL, err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			log.Infof("http POST request to %s succeeded", pd.allPodsURL)
		} else {
			log.Errorf("http POST request to %s failed with status code %d", pd.allPodsURL, resp.StatusCode)
		}
	}
}

func (pd *PodDumper) getAllPodsAsPerceptorPods() ([]perceptorapi.Pod, error) {
	perceptorPods := []perceptorapi.Pod{}

	// Get all pods from kubernetes
	pods, err := pd.coreV1.Pods(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Translate the pods from kubernetes to perceptor format
	for _, pod := range pods.Items {
		perceptorPod, err := mapper.NewPerceptorPodFromKubePod(&pod)
		if err != nil {
			return nil, err
		}
		perceptorPods = append(perceptorPods, *perceptorPod)
	}
	return perceptorPods, nil
}
