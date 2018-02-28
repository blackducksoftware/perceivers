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

package communicator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SendPerceptorAddEvent sends an add event to perceptor at the dest endpoint
func SendPerceptorAddEvent(dest string, obj interface{}) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		recordError("unable to serialize JSON")
		return fmt.Errorf("unable to serialize %v: %v", obj, err)
	}
	resp, err := http.Post(dest, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		recordError("unable to create POST request")
		return fmt.Errorf("unable to POST to %s: %v", dest, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		recordError("add POST request failed")
		return fmt.Errorf("http POST request to %s failed with status code %d", dest, resp.StatusCode)
	}
	return nil
}

// SendPerceptorDeleteEvent sends a delete event to perceptor at the dest endpoint
func SendPerceptorDeleteEvent(dest string, name string) error {
	jsonBytes, err := json.Marshal(name)
	if err != nil {
		recordError("unable to serialize JSON")
		return fmt.Errorf("unable to serialize %s: %v", name, err)
	}
	req, err := http.NewRequest("DELETE", dest, bytes.NewBuffer(jsonBytes))
	if err != nil {
		recordError("unable to create DELETE request")
		return fmt.Errorf("unable to create DELETE request for %s: %v", dest, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		recordError("unable to issue delete request")
		return fmt.Errorf("unable to DELETE to %s: %v", dest, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		recordError("delete request failed")
		return fmt.Errorf("http DELETE request to %s failed with status code %d", dest, resp.StatusCode)
	}
	return nil
}
