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

type QuayRepo struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Namespace   string   `json:"namespace"`
	DockerURL   string   `json:"docker_url"`
	Homepage    string   `json:"homepage"`
	UpdatedTags []string `json:"updated_tags"`
}

type QuayTagDigest struct {
	HasAdditional bool `json:"has_additional"`
	Page          int  `json:"page"`
	Tags          []struct {
		Name           string `json:"name"`
		Reversion      bool   `json:"reversion"`
		StartTs        int    `json:"start_ts"`
		ImageID        string `json:"image_id"`
		LastModified   string `json:"last_modified"`
		ManifestDigest string `json:"manifest_digest"`
		DockerImageID  string `json:"docker_image_id"`
		IsManifestList bool   `json:"is_manifest_list"`
		Size           int    `json:"size"`
	} `json:"tags"`
}
