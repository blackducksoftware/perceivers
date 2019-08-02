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

// QuayRepo contains a quay image with list of tags
type QuayRepo struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Namespace   string   `json:"namespace"`
	DockerURL   string   `json:"docker_url"`
	Homepage    string   `json:"homepage"`
	UpdatedTags []string `json:"updated_tags"`
}

// QuayTagDigest contains Digest for a particular Quay image
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

// QuayLabels contains a list of returned Labels on an image
type QuayLabels struct {
	Labels []struct {
		Value      string `json:"value"`
		MediaType  string `json:"media_type"`
		ID         string `json:"id"`
		Key        string `json:"key"`
		SourceType string `json:"source_type"`
	} `json:"labels"`
}

// QuayLabel is used for Posting a new label,
// doesn't need to have json metadatas but couldn't hurt
type QuayLabel struct {
	MediaType string `json:"media_type"`
	Value     string `json:"value"`
	Key       string `json:"key"`
}
