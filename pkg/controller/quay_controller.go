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

package controller

import (
	//"fmt"
	"time"

	utils "github.com/blackducksoftware/perceivers/pkg/utils"
	//perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"
	//m "github.com/blackducksoftware/perceptor/pkg/core/model"

	log "github.com/sirupsen/logrus"
)

// QuayController handles watching images and sending them to perceptor
type QuayController struct {
	perceptorURL    string
	registryAuths   []*utils.RegistryAuth
	quayAccessToken string
}

// NewQuayController creates a new ArtifactoryController object
func NewQuayController(perceptorURL string, credentials []*utils.RegistryAuth, quayAccessToken string) *QuayController {
	return &QuayController{
		perceptorURL:    perceptorURL,
		registryAuths:   credentials,
		quayAccessToken: quayAccessToken,
	}
}

// Run starts a controller that watches images and sends them to perceptor
func (qc *QuayController) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting quay controller")
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		err := qc.imageLookup()
		if err != nil {
			log.Errorf("failed to add images to scan queue: %v", err)
		}

		time.Sleep(interval)
	}
}

func (qc *QuayController) imageLookup() error {

	return nil
}
