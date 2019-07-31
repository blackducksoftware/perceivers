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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"
)

// RegistryAuth stores the credentials for a private docker repo
// and is same as common.RegistryAuth in perceptor-scanner repo
type RegistryAuth struct {
	URL      string
	User     string
	Password string
}

// GetResourceOfType takes in the specified URL with credentials and
// tries to decode returning json to specified interface
func GetResourceOfType(url string, cred *RegistryAuth, bearerToken string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("Error in creating get request %e at url %s", err, url)
	}

	if cred != nil {
		req.SetBasicAuth(cred.User, cred.Password)
	}

	if bearerToken != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// PingArtifactoryServer takes in the specified URL with username & password and checks weather
// it's a valid login for artifactory by pinging the server
func PingArtifactoryServer(url string, username string, password string) (*RegistryAuth, error) {
	url = fmt.Sprintf("%s/artifactory/api/system/ping", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error in pinging artifactory server %e", err)
	}
	req.SetBasicAuth(username, password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error in pinging artifactory server supposed to get %d response code got %d", http.StatusOK, resp.StatusCode)
	}
	return &RegistryAuth{URL: url, User: username, Password: password}, nil
}

// PutImageOnScanQueue pushes the image to the Perceptor queue
func PutImageOnScanQueue(perceptorURL string, im *m.Image) error {
	perceptorURL = fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ImagePath)
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(im)
	req, err := http.NewRequest(http.MethodPost, perceptorURL, buffer)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OK status code not observer from perceptor, status code: %d", resp.StatusCode)
	}

	return nil
}
