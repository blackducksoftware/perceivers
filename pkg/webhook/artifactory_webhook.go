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

package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	utils "github.com/blackducksoftware/perceivers/pkg/utils"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

// ArtifactoryWebhook handles watching images and sending them to perceptor
type ArtifactoryWebhook struct {
	perceptorURL  string
	registryAuths []*utils.RegistryAuth
}

// NewArtifactoryWebhook creates a new ArtifactoryWebhook object
func NewArtifactoryWebhook(perceptorURL string, credentials []*utils.RegistryAuth) *ArtifactoryWebhook {
	return &ArtifactoryWebhook{
		perceptorURL:  perceptorURL,
		registryAuths: credentials,
	}
}

// Run starts a controller that watches images and sends them to perceptor
func (aw *ArtifactoryWebhook) Run() {
	log.Infof("Webhook: starting artifactory webhook on 443 at /webhook")
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			log.Info("Webhook: Artifactory hook incoming!")
			ahs := &utils.ArtHookStruct{}
			json.NewDecoder(r.Body).Decode(ahs)
			for _, registry := range aw.registryAuths {
				cred, err := utils.PingArtifactoryServer("http://"+registry.URL, registry.User, registry.Password)
				if err != nil {
					log.Debugf("Webhook: URL %s either not a valid Artifactory repository or incorrect credentials: %e", registry.URL, err)
					continue
				}
				aw.webhook(ahs, cred, aw.perceptorURL)
			}
		}
	})
	err := http.ListenAndServe(":443", nil)
	if err != nil {
		log.Error("Webhook: Webhook listener failed!")
	}
}

func (aw *ArtifactoryWebhook) webhook(ahs *utils.ArtHookStruct, cred *utils.RegistryAuth, perceptorURL string) {
	for _, a := range ahs.Artifacts {
		// Trying to find the repo key, cannot split because image may contain '/'
		// So stripping down the returned URL by removing everything
		returnedURL := a.Reference
		if strings.Contains(returnedURL, "null") {
			log.Errorf("Webhook: The plugin needs to setup BaseURL in the webhook json")
		}
		woBase := strings.Replace(returnedURL, cred.URL+"/", "", -1)
		woRepo := strings.Replace(woBase, "/"+a.Name, "", -1)
		repoKey := strings.Replace(woRepo, ":"+a.Version, "", -1)

		imageSHAs := &utils.ArtImageSHAs{}
		url := fmt.Sprintf("%s/api/storage/%s/%s/%s/manifest.json?properties=sha256", cred.URL, repoKey, a.Name, a.Version)
		err := utils.GetResourceOfType(url, cred, "", imageSHAs)
		if err != nil {
			log.Errorf("Webhook: Error in getting SHAs of the artifactory image: %e", err)
			continue
		}
		for _, sha := range imageSHAs.Properties.Sha256 {

			sha, err := m.NewDockerImageSha(sha)
			if err != nil {
				log.Errorf("Webhook: Error in docker SHA: %e", err)
			} else {

				// Remove Tag & HTTPS because image model doesn't require it
				url = fmt.Sprintf("%s/%s/%s", cred.URL, repoKey, a.Name)
				url = strings.Replace(url, "http://", "", -1)
				url = strings.Replace(url, "https://", "", -1)
				artImage := m.NewImage(url, a.Version, sha, 1, url, a.Version)

				err := utils.PutImageOnScanQueue(perceptorURL, artImage)
				if err != nil {
					log.Errorf("Webhook: Error putting artifactory image %v in perceptor queue %e", artImage, err)
				} else {
					log.Infof("Webhook: Successfully put image %s with tag %s in perceptor queue", url, a.Version)
				}
			}
		}

	}
}
