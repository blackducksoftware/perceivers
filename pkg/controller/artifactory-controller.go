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
	"fmt"
	"time"

	utils "github.com/blackducksoftware/perceivers/pkg/utils"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"

	log "github.com/sirupsen/logrus"
)

// ArtifactoryController handles watching images and sending them to perceptor
type ArtifactoryController struct {
	perceptorURL  string
	registryAuths []*utils.RegistryAuth
}

// NewArtifactoryController creates a new ArtifactoryController object
func NewArtifactoryController(perceptorURL string, credentials []*utils.RegistryAuth) *ArtifactoryController {
	return &ArtifactoryController{
		perceptorURL:  perceptorURL,
		registryAuths: credentials,
	}
}

// Run starts a controller that watches images and sends them to perceptor
func (ic *ArtifactoryController) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting artifactory controller")
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		err := ic.imageLookup()
		if err != nil {
			log.Errorf("failed to add artifactory images to scan queue: %v", err)
		}

		time.Sleep(interval)
	}
}

func (ic *ArtifactoryController) imageLookup() error {
	log.Infof("Total %d private registries found!", len(ic.registryAuths))
	for _, registry := range ic.registryAuths {

		baseURL := fmt.Sprintf("https://%s", registry.URL)
		cred, err := utils.PingArtifactoryServer(baseURL, registry.User, registry.Password)
		if err != nil {
			log.Warnf("Controller: URL %s either not a valid Artifactory repository or incorrect credentials: %e", baseURL, err)
			break
		}

		dockerRepos := &utils.ArtDockerRepo{}
		images := &utils.ArtImages{}
		imageTags := &utils.ArtImageTags{}
		imageSHAs := &utils.ArtImageSHAs{}

		url := fmt.Sprintf("%s/artifactory/api/repositories?packageType=docker", baseURL)
		err = utils.GetResourceOfType(url, cred, "", dockerRepos)
		if err != nil {
			log.Errorf("Error in getting docker repo: %e", err)
			break
		}

		for _, repo := range *dockerRepos {
			url = fmt.Sprintf("%s/artifactory/api/docker/%s/v2/_catalog", baseURL, repo.Key)
			err = utils.GetResourceOfType(url, cred, "", images)
			if err != nil {
				log.Errorf("Error in getting catalog in repo: %e", err)
				break
			}

			for _, image := range images.Repositories {
				url = fmt.Sprintf("%s/artifactory/api/docker/%s/v2/%s/tags/list", baseURL, repo.Key, image)
				err = utils.GetResourceOfType(url, cred, "", imageTags)
				if err != nil {
					log.Errorf("Error in getting image: %e", err)
					break
				}

				for _, tag := range imageTags.Tags {
					url = fmt.Sprintf("%s/artifactory/api/storage/%s/%s/%s/manifest.json?properties=sha256", baseURL, repo.Key, image, tag)
					err = utils.GetResourceOfType(url, cred, "", imageSHAs)
					if err != nil {
						log.Errorf("Error in getting SHAs of the artifactory image: %e", err)
						break
					}

					for _, sha := range imageSHAs.Properties.Sha256 {

						sha, err := m.NewDockerImageSha(sha)
						if err != nil {
							log.Errorf("Error in docker SHA: %e", err)
						} else {

							// Remove Tag & HTTPS because image model doesn't require it
							url = fmt.Sprintf("%s/%s/%s", registry.URL, repo.Key, image)
							artImage := m.NewImage(url, tag, sha, 1, url, tag)

							err := utils.PutImageOnScanQueue(ic.perceptorURL, artImage)
							if err != nil {
								log.Errorf("Error putting artifactory image %v in perceptor queue %e", artImage, err)
							} else {
								log.Infof("Successfully put image %s in perceptor queue", url, tag)
							}
						}
					}
				}
			}
		}

		log.Infof("There were total %d docker repositories found in artifactory.", len(images.Repositories))

	}

	return nil
}
