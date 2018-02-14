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

package app

import (
	"fmt"
	"time"

	"github.com/blackducksoftware/perceivers/image/pkg/annotator"
	"github.com/blackducksoftware/perceivers/image/pkg/controller"
	"github.com/blackducksoftware/perceivers/image/pkg/dumper"

	"k8s.io/client-go/rest"

	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"

	log "github.com/sirupsen/logrus"
)

// ImagePerceiver handles watching and annotating Images
type ImagePerceiver struct {
	client *imagev1.ImageV1Client

	ImageController *controller.ImageController

	ImageAnnotator     *annotator.ImageAnnotator
	annotationInterval time.Duration

	ImageDumper  *dumper.ImageDumper
	dumpInterval time.Duration
}

// NewImagePerceiver creates a new ImagePerceiver object
func NewImagePerceiver(config *ImagePerceiverConfig) (*ImagePerceiver, error) {
	// Create a kube client from in cluster configuration
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to build config from cluster: %v", err)
	}
	imageClient, err := imagev1.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create image client: %v", err)
	}

	perceptorURL := fmt.Sprintf("http://%s:%d", config.PerceptorHost, config.PerceptorPort)
	p := ImagePerceiver{
		ImageController:    controller.NewImageController(imageClient, perceptorURL),
		ImageAnnotator:     annotator.NewImageAnnotator(imageClient, perceptorURL),
		annotationInterval: time.Second * time.Duration(config.AnnotationIntervalSeconds),
		ImageDumper:        dumper.NewImageDumper(imageClient, perceptorURL),
		dumpInterval:       time.Minute * time.Duration(config.DumpIntervalMinutes),
	}

	return &p, nil
}

// Run starts the ImagePerceiver watching and annotating Images
func (kp *ImagePerceiver) Run(stopCh <-chan struct{}) {
	log.Infof("starting image controllers")
	go kp.ImageController.Run(5, stopCh)
	go kp.ImageAnnotator.Run(kp.annotationInterval, stopCh)
	go kp.ImageDumper.Run(kp.dumpInterval, stopCh)

	<-stopCh
}
