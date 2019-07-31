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
	"fmt"
	"strings"

	utils "github.com/blackducksoftware/perceivers/pkg/utils"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"

	log "github.com/sirupsen/logrus"
)

func webhook(bearerToken string, perceptorURL string, qr *utils.QuayRepo) {

	log.Info(qr.Name)
	log.Info(qr.Repository)
	log.Info(qr.Namespace)
	log.Info(qr.DockerURL)
	log.Info(qr.Homepage)
	log.Info(qr.UpdatedTags)

	rt := &utils.QuayTagDigest{}
	url := strings.Replace(qr.Homepage, "repository", "api/v1/repository", -1)
	url = fmt.Sprintf("%s/tag", url)
	err := utils.GetResourceOfType(url, nil, bearerToken, rt)
	if err != nil {
		log.Errorf("Error in getting docker repo: %e", err)
	}

	for _, tagDigest := range rt.Tags {
		sha, err := m.NewDockerImageSha(strings.Replace(tagDigest.ManifestDigest, "sha256:", "", -1))
		if err != nil {
			log.Errorf("Error in docker SHA: %e", err)
		} else {
			quayImage := m.NewImage(qr.DockerURL, tagDigest.Name, sha, 1, qr.DockerURL, tagDigest.Name)
			err := utils.PutImageOnScanQueue(perceptorURL, quayImage)
			if err != nil {
				log.Errorf("Error putting image %v in perceptor queue %e", quayImage, err)
			} else {
				log.Infof("Successfully put image %s with tag %s in perceptor queue", url, tagDigest.Name)
			}
		}
	}

}
