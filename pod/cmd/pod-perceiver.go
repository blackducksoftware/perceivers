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

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/blackducksoftware/perceivers/pod/cmd/app"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// TODO metrics
// number of namespaces found
// number of pods per namespace
// number of images per pod
// number of occurrences of each pod
// number of successes, failures, of each perceptor endpoint
// ??? number of scan results fetched from perceptor

func main() {
	log.Info("starting pod-perceiver")

	config, err := app.GetPodPerceiverConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %v", err))
	}

	// Create the Pod Perceiver
	perceiver, err := app.NewPodPerceiver(config)
	if err != nil {
		panic(fmt.Errorf("failed to create pod-perceiver: %v", err))
	}

	go setUpPrometheus(config.Port)

	// Run the perceiver
	stopCh := make(chan struct{})
	perceiver.Run(stopCh)
}

func setUpPrometheus(port int) {
	log.Info("setting up prometheus on port %d", port)
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())
	http.Handle("/metrics", prometheus.Handler())

	addr := fmt.Sprintf(":%d", port)
	http.ListenAndServe(addr, nil)
}
