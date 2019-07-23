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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	utils "github.com/blackducksoftware/perceivers/pkg/utils"
	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"

	log "github.com/sirupsen/logrus"
)

// ArtifactoryController handles watching images and sending them to perceptor
type ArtifactoryController struct {
	perceptorURL  string
	registryAuths []*utils.ArtifactoryCredentials
}

// DockerRepo contains list of docker repos in artifactory
type DockerRepo []struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	PackageType string `json:"packageType"`
}

// Images contain list of images inside the docker repo
type Images struct {
	Repositories []string `json:"repositories"`
}

// ImageTags lists out all the tags for the image
type ImageTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// ImageSHAs gets all the sha256 of an image
type ImageSHAs struct {
	Properties struct {
		Sha256 []string `json:"sha256"`
	} `json:"properties"`
	URI string `json:"uri"`
}

// NewArtifactoryController creates a new ArtifactoryController object
func NewArtifactoryController(perceptorURL string, credentials []*utils.ArtifactoryCredentials) *ArtifactoryController {
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
			log.Errorf("failed to add images to scan queue: %v", err)
		}

		time.Sleep(interval)
	}
}

func (ic *ArtifactoryController) imageLookup() error {
	for _, registry := range ic.registryAuths {

		baseURL := fmt.Sprintf("https://%s", registry.URL)
		cred, err := utils.PingArtifactoryServer(baseURL, registry.User, registry.Password)
		if err != nil {
			log.Warnf("Controller: URL %s either not a valid Artifactory repository or incorrect credentials: %e", baseURL, err)
			break
		}

		dockerRepos := &DockerRepo{}
		images := &Images{}
		imageTags := &ImageTags{}
		imageSHAs := &ImageSHAs{}

		url := fmt.Sprintf("%s/artifactory/api/repositories?packageType=docker", baseURL)
		err = utils.GetResourceOfType(url, cred, dockerRepos)
		if err != nil {
			log.Errorf("Error in getting docker repo: %e", err)
			break
		}

		for _, repo := range *dockerRepos {
			url = fmt.Sprintf("%s/artifactory/api/docker/%s/v2/_catalog", baseURL, repo.Key)
			err = utils.GetResourceOfType(url, cred, images)
			if err != nil {
				log.Errorf("Error in getting catalog in repo: %e", err)
				break
			}

			for _, image := range images.Repositories {
				url = fmt.Sprintf("%s/artifactory/api/docker/%s/v2/%s/tags/list", baseURL, repo.Key, image)
				err = utils.GetResourceOfType(url, cred, imageTags)
				if err != nil {
					log.Errorf("Error in getting image: %e", err)
					break
				}

				for _, tag := range imageTags.Tags {
					url = fmt.Sprintf("%s/artifactory/api/storage/%s/%s/%s/manifest.json?properties=sha256", baseURL, repo.Key, image, tag)
					err = utils.GetResourceOfType(url, cred, imageSHAs)
					if err != nil {
						log.Errorf("Error in getting SHAs of the image: %e", err)
						break
					}

					for _, sha := range imageSHAs.Properties.Sha256 {

						url = fmt.Sprintf("%s/%s:%s", baseURL, image, tag)
						log.Infof("URL: %s", url)
						log.Infof("Tag: %s", tag)
						log.Infof("SHA: %s", sha)
						log.Infof("Priority: %d", 1)
						log.Infof("BlackDuckProjectName: %s/%s/%s", registry.URL, repo.Key, image)
						log.Infof("BlackDuckProjectVersion: %s", tag)

						sha, err := m.NewDockerImageSha(sha)
						if err != nil {
							log.Errorf("Error in docker SHA: %e", err)
						} else {

							// Remove Tag & HTTPS because image model doesn't require it
							url = fmt.Sprintf("%s/%s/%s", registry.URL, repo.Key, image)
							projectName := fmt.Sprintf("%s/%s/%s", registry.URL, repo.Key, image)
							artImage := m.NewImage(url, tag, sha, 0, projectName, tag)
							ic.putImageOnScanQueue(artImage, cred)
						}
					}
				}
			}
		}

		log.Infof("There were total %d docker repositories found in artifactory.", len(images.Repositories))

	}

	return nil
}

func (ic *ArtifactoryController) putImageOnScanQueue(im *m.Image, cred *utils.ArtifactoryCredentials) {
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(im)
	url := fmt.Sprintf("%s/%s", ic.perceptorURL, perceptorapi.ImagePath)
	req, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		log.Errorf("Error in creating post request %e", err)
	}
	req.SetBasicAuth(cred.User, cred.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Error in sending request %e", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Infof("Success in posting image to the queue")
	} else {
		log.Errorf("OK status code not observer from perceptor, status code: %d", resp.StatusCode)
	}
}
