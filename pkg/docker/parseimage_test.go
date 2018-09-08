/*
Copyright (C) 2018 Synopsys, Inc.

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

package docker

import (
	"fmt"
	"testing"
)

func TestParseImageIDString(t *testing.T) {
	testcases := []struct {
		description string
		prefix      string
		name        string
		shaPrefix   string
		sha         string
		shouldPass  bool
	}{
		{
			description: "valid format",
			prefix:      "docker-pullable://",
			name:        "abc",
			shaPrefix:   "@",
			sha:         "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043",
			shouldPass:  true,
		},
		{
			description: "valid format with 2 directories",
			prefix:      "docker-pullable://",
			name:        "abc/def",
			shaPrefix:   "@",
			sha:         "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043",
			shouldPass:  true,
		},
		{
			description: "valid format with private registry",
			prefix:      "docker-pullable://",
			name:        "docker-registry.default.svc:5000/def/ghi",
			shaPrefix:   "@",
			sha:         "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043",
			shouldPass:  true,
		},
		{
			description: "missing prefix",
			prefix:      "",
			name:        "abc/def",
			shaPrefix:   "@",
			sha:         "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043",
			shouldPass:  true,
		},
		{
			description: "missing image name",
			prefix:      "docker-pullable://",
			name:        "",
			shaPrefix:   "@",
			sha:         "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043",
			shouldPass:  false,
		},
		{
			description: "missing sha",
			prefix:      "docker-pullable://",
			name:        "abc/def",
			shaPrefix:   "@",
			sha:         "",
			shouldPass:  false,
		},
		{
			description: "Docker prefix",
			prefix:      "docker://",
			name:        "",
			shaPrefix:   "",
			sha:         "cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043",
			shouldPass:  false,
		},
	}

	for _, tc := range testcases {
		imageID := fmt.Sprintf("%s%s%ssha256:%s", tc.prefix, tc.name, tc.shaPrefix, tc.sha)
		name, sha, err := ParseImageIDString(imageID)
		//fmt.Printf("Test: %s, err: %s, imageID: %s, name: %s, sha: %s \n", tc.description, err, imageID, name, sha)
		if err != nil && tc.shouldPass {
			t.Errorf("[%s] unexpected error: %v, imageID %s", tc.description, err, imageID)
		}
		if name != tc.name && tc.shouldPass {
			t.Errorf("[%s] name is wrong.  Expected %s got %s", tc.description, tc.name, name)
		}
		if sha != tc.sha && tc.shouldPass {
			t.Errorf("[%s] sha is wrong.  Expected %s got %s", tc.description, tc.sha, sha)
		}
		if !tc.shouldPass && err == nil {
			t.Errorf("The error should not be empty, description: %s , prefix: %s, name: %s, sha: %s", tc.description, tc.prefix, name, sha)
		}
	}
}

func TestParseImageString(t *testing.T) {
	testcases := []struct {
		description string
		repo        string
		tag         string
	}{
		{
			description: "repo with path and tag",
			repo:        "url.com/imagename",
			tag:         "latest",
		},
		{
			description: "repo with path without tag",
			repo:        "url.com/imagename",
			tag:         "",
		},
		{
			description: "repo with path and port and tag",
			repo:        "url.com:80/imagename",
			tag:         "latest",
		},
		{
			description: "repo with path and port without tag",
			repo:        "url.com:80/imagename",
			tag:         "",
		},
		{
			description: "image name only with tag",
			repo:        "imagename",
			tag:         "1.2.3",
		},
		{
			description: "image name only without tag",
			repo:        "imagename",
			tag:         "",
		},
	}

	for _, tc := range testcases {
		var image string
		if len(tc.tag) > 0 {
			image = fmt.Sprintf("%s:%s", tc.repo, tc.tag)
		} else {
			image = tc.repo
		}
		repo, tag := ParseImageString(image)
		if repo != tc.repo {
			t.Errorf("expected repo %s got %s", tc.repo, repo)
		}

		if tag != tc.tag {
			t.Errorf("expected tag %s got %s", tc.tag, tag)
		}
	}
}

func TestParseShaImageStrings(t *testing.T) {
	imageString := "gcr.io/gke-verification/blackducksoftware/perceptor@sha256:9914478c9642be49e7791a7a29207c0a6194c8bf6e9690ab5902008cce8af39f"
	repo, tag := ParseImageString(imageString)
	expectedRepo := "gcr.io/gke-verification/blackducksoftware/perceptor"
	if repo != expectedRepo {
		t.Errorf("repo: expected %s, got %s", expectedRepo, repo)
	}
	expectedTag := ""
	if tag != expectedTag {
		t.Errorf("tag: expected %s, got %s", expectedTag, tag)
	}
}
