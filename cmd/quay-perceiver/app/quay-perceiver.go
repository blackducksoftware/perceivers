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

package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blackducksoftware/perceivers/pkg/annotator"
	"github.com/blackducksoftware/perceivers/pkg/controller"
	utils "github.com/blackducksoftware/perceivers/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// QuayPerceiver handles watching and annotating Images
type QuayPerceiver struct {
	controller         *controller.QuayController
	annotator          *annotator.QuayAnnotator
	annotationInterval time.Duration
	dumpInterval       time.Duration
	quayAccessToken    string
	perceptorURL       string
	metricsURL         string
}

// NewQuayPerceiver creates a new ImagePerceiver object
func NewQuayPerceiver(configPath string) (*QuayPerceiver, error) {
	config, err := GetConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	// Configure prometheus for metrics
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())
	http.Handle("/metrics", prometheus.Handler())

	perceptorURL := fmt.Sprintf("http://%s:%d", config.Perceptor.Host, config.Perceptor.Port)
	qp := QuayPerceiver{
		controller:         controller.NewQuayController(perceptorURL, config.PrivateDockerRegistries, config.QuayAccessToken),
		annotator:          annotator.NewQuayAnnotator(perceptorURL, config.PrivateDockerRegistries),
		annotationInterval: time.Second * time.Duration(config.Perceiver.AnnotationIntervalSeconds),
		dumpInterval:       time.Minute * time.Duration(config.Perceiver.DumpIntervalMinutes),
		quayAccessToken:    config.QuayAccessToken,
		perceptorURL:       perceptorURL,
		metricsURL:         fmt.Sprintf(":%d", config.Perceiver.Port),
	}
	return &qp, nil
}

// Run starts the QuayPerceiver watching and annotating Images
func (qp *QuayPerceiver) Run(stopCh <-chan struct{}) {
	log.Infof("starting quay controllers")
	go qp.controller.Run(qp.dumpInterval, stopCh)
	go qp.annotator.Run(qp.annotationInterval, stopCh)

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			log.Info("Quay hook incoming!")
			qr := &utils.QuayRepo{}
			json.NewDecoder(r.Body).Decode(qr)
			webhook(qp.quayAccessToken, qp.perceptorURL, qr)
		}
	})
	log.Infof("starting webhook on 443 at /webhook")
	err := http.ListenAndServe(":443", nil)
	if err != nil {
		log.Error("Webhook listener failed!")
		log.Fatal(err)
	}

	<-stopCh
}
