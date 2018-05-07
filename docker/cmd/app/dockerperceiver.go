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

package app

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blackducksoftware/perceivers/docker/pkg/annotator"
	"github.com/blackducksoftware/perceivers/docker/pkg/controller"
	dockerClient "github.com/blackducksoftware/perceivers/docker/pkg/docker"
	"github.com/blackducksoftware/perceivers/docker/pkg/dumper"
	"github.com/blackducksoftware/perceivers/pkg/annotations"
	docker "github.com/fsouza/go-dockerclient"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

// PodPerceiver handles watching and annotating pods
type DockerPerceiver struct {
	dockerController   *controller.EventHandler
	dockerAnnotator    *annotator.DockerAnnotator
	annotationInterval time.Duration

	dockerDumper *dumper.DockerDumper
	dumpInterval time.Duration

	metricsURL string
}

type Handlers struct {
	handledEvents chan *docker.APIEvents
	handlerFunc   func(event *docker.APIEvents) error
}

func (th *Handlers) Handle(event *docker.APIEvents) error {
	return th.handlerFunc(event)
}

// NewPodPerceiver creates a new PodPerceiver object
func NewDockerPerceiver(handler annotations.PodAnnotatorHandler, configPath string) (*DockerPerceiver, error) {
	config, err := GetDockerPerceiverConfig(configPath)
	if err != nil {
		panic(fmt.Errorf("failed to read config: %v", err))
	}

	client, err := dockerClient.NewDocker()

	handledEvents := make(chan *docker.APIEvents, 10)
	hFn := func(event *docker.APIEvents) error {
		handledEvents <- event
		return nil
	}

	eventHandler := &Handlers{
		handlerFunc: hFn,
	}
	eventHandlers := map[string][]controller.Handler{"create": {eventHandler}}

	// Configure prometheus for metrics
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())
	http.Handle("/metrics", prometheus.Handler())

	perceptorURL := fmt.Sprintf("http://%s:%d", config.PerceptorHost, config.PerceptorPort)
	p := DockerPerceiver{
		dockerController:   controller.NewEventHandler(10, 10, client, eventHandlers, perceptorURL),
		dockerAnnotator:    annotator.NewDockerAnnotator(client, perceptorURL, handler),
		annotationInterval: time.Second * time.Duration(config.AnnotationIntervalSeconds),
		dockerDumper:       dumper.NewDockerDumper(client, perceptorURL),
		dumpInterval:       time.Minute * time.Duration(config.DumpIntervalMinutes),
		metricsURL:         fmt.Sprintf(":%d", config.Port),
	}

	return &p, nil
}

// Run starts the PodPerceiver watching and annotating pods
func (pp *DockerPerceiver) Run(stopCh <-chan struct{}) {
	log.Infof("starting Docker controllers")
	// go pp.dockerController.Run()
	go pp.dockerAnnotator.Run(pp.annotationInterval, stopCh)
	go pp.dockerDumper.Run(pp.dumpInterval, stopCh)

	log.Infof("starting prometheus on %d", pp.metricsURL)
	http.ListenAndServe(pp.metricsURL, nil)

	<-stopCh
}
