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

package annotator

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/blackducksoftware/perceivers/pkg/annotations"
	"github.com/blackducksoftware/perceivers/pkg/utils"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var scannedImages = []perceptorapi.ScannedImage{
	{
		Repository:       "image1",
		Sha:              "ASDJ4FSF3FSFK3SF450",
		PolicyViolations: 100,
		Vulnerabilities:  5,
		OverallStatus:    "STATUS3",
		ComponentsURL:    "http://url.com",
	},
	{
		Repository:       "this.name.includes.registry.name/imagenameis/short/butthefulllengthwithregistryistoolong",
		Sha:              "HAFGW2392FJGNE3FFK04",
		PolicyViolations: 5,
		Vulnerabilities:  15,
		OverallStatus:    "STATUS4",
		ComponentsURL:    "http://new.com",
	},
	{
		Repository:       "this.name.includes.registry.name/and/many/directories/and/is/way/too/long/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Sha:              "HAFGW2392FJGNE3FFK04",
		PolicyViolations: 0,
		Vulnerabilities:  0,
		OverallStatus:    "STATUS5",
		ComponentsURL:    "http://thisurlisreallylongtoo.com/andwouldfailthe63characterlimit/butshouldntbeneeded",
	},
	{
		Repository:       "this/name/and/many/directories/and/is/way/too/long/tofitin/the63character/limitofalabel",
		Sha:              "HAFGW2392FJGNE3FFK04",
		PolicyViolations: 1,
		Vulnerabilities:  40,
		OverallStatus:    "STATUS6",
		ComponentsURL:    "http://thisurlisreallylongtoo.com/andwouldfailthe63characterlimit/butshouldntbeneeded",
	},
	{
		Repository:       "registry:port/imagenameis/short/butthefulllengthwithregistryistoolong",
		Sha:              "HAFGW2392FJGNE3FFK04",
		PolicyViolations: 10,
		Vulnerabilities:  1,
		OverallStatus:    "STATUS7",
		ComponentsURL:    "http://registry.com",
	},
}

var scannedPods = []perceptorapi.ScannedPod{
	{
		Name:             "pod1",
		Namespace:        "ns1",
		PolicyViolations: 10,
		Vulnerabilities:  0,
		OverallStatus:    "STATUS1",
	},
	{
		Name:             "pod2",
		Namespace:        "ns2",
		PolicyViolations: 0,
		Vulnerabilities:  20,
		OverallStatus:    "STATUS2",
	},
}

var results = perceptorapi.ScanResults{
	Pods:   scannedPods,
	Images: scannedImages,
}

func makeImageAnnotationObj(pos int) *annotations.ImageAnnotationData {
	image := scannedImages[pos]
	return annotations.NewImageAnnotationData(image.PolicyViolations, image.Vulnerabilities, image.OverallStatus, image.ComponentsURL, "", "")
}

func makePodAnnotationObj(pos int) *annotations.PodAnnotationData {
	pod := scannedPods[pos%len(scannedPods)]
	return annotations.NewPodAnnotationData(pod.PolicyViolations, pod.Vulnerabilities, pod.OverallStatus, "", "")
}

func makePodWithImage(pos int, name string, sha string) *v1.Pod {
	scannedPod := scannedPods[pos%len(scannedPods)]
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scannedPod.Name,
			Namespace: scannedPod.Namespace,
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:    name,
					ImageID: fmt.Sprintf("docker-pullable://%s@sha256:%s", name, sha),
				},
			},
		},
	}
}

func makePod(pos int) *v1.Pod {
	scannedImage := scannedImages[pos]
	return makePodWithImage(pos, scannedImage.Repository, scannedImage.Sha)
}

func createPA() *PodAnnotator {
	return &PodAnnotator{h: annotations.PodAnnotatorHandlerFuncs{
		PodLabelCreationFunc:      annotations.CreatePodLabels,
		PodAnnotationCreationFunc: annotations.CreatePodAnnotations,
		ImageAnnotatorHandlerFuncs: annotations.ImageAnnotatorHandlerFuncs{
			ImageLabelCreationFunc:      annotations.CreateImageLabels,
			ImageAnnotationCreationFunc: annotations.CreateImageAnnotations,
			MapCompareHandlerFuncs: annotations.MapCompareHandlerFuncs{
				MapCompareFunc: annotations.StringMapContains,
			},
		},
	}}
}

func TestPodAnnotatorGetScanResults(t *testing.T) {
	testcases := []struct {
		description   string
		statusCode    int
		body          *perceptorapi.ScanResults
		expectedScans *perceptorapi.ScanResults
		shouldPass    bool
	}{
		{
			description:   "successful GET with actual results",
			statusCode:    200,
			body:          &results,
			expectedScans: &results,
			shouldPass:    true,
		},
		{
			description:   "successful GET with empty results",
			statusCode:    200,
			body:          &perceptorapi.ScanResults{},
			expectedScans: &perceptorapi.ScanResults{},
			shouldPass:    true,
		},
		{
			description:   "bad status code",
			statusCode:    401,
			body:          nil,
			expectedScans: nil,
			shouldPass:    false,
		},
		{
			description:   "nil body on successful GET",
			statusCode:    200,
			body:          nil,
			expectedScans: &perceptorapi.ScanResults{},
			shouldPass:    true,
		},
	}

	endpoint := "RESTEndpoint"
	for _, tc := range testcases {
		bytes, _ := json.Marshal(tc.body)
		handler := utils.FakeHandler{
			StatusCode:  tc.statusCode,
			RespondBody: string(bytes),
			T:           t,
		}
		server := httptest.NewServer(&handler)
		defer server.Close()

		annotator := PodAnnotator{
			scanResultsURL: fmt.Sprintf("%s/%s", server.URL, endpoint),
		}
		scanResults, err := annotator.getScanResults()
		if err != nil && tc.shouldPass {
			t.Fatalf("[%s] unexpected error: %v", tc.description, err)
		}
		if !reflect.DeepEqual(tc.expectedScans, scanResults) {
			t.Errorf("[%s] received %v expected %v", tc.description, scanResults, tc.expectedScans)
		}
	}
}

func TestAddPodAnnotations(t *testing.T) {
	podAnnotationSet := func(pos int) map[string]string {
		return annotations.CreatePodAnnotations(makePodAnnotationObj(pos))
	}

	imageAnnotationSet := func(pos int) map[string]string {
		return annotations.CreateImageAnnotations(makeImageAnnotationObj(pos), scannedImages[pos].Repository, 0)
	}

	fullAnnotationSet := func(pos int) map[string]string {
		return utils.MapMerge(podAnnotationSet(pos), imageAnnotationSet(pos))
	}

	partialPodAnnotationSet := func(pos int) map[string]string {
		annotations := make(map[string]string)
		for k, v := range podAnnotationSet(pos) {
			if !strings.Contains(k, "scanner-version") {
				annotations[k] = v
			}
		}
		return annotations
	}

	partialImageAnnotationSet := func(pos int) map[string]string {
		annotations := make(map[string]string)
		for k, v := range imageAnnotationSet(pos) {
			if !strings.Contains(k, "project-endpoint") {
				annotations[k] = v
			}
		}
		return annotations
	}

	otherAnnotations := map[string]string{"key1": "value1", "key2": "value2"}

	testcases := []struct {
		description         string
		pod                 *v1.Pod
		position            int
		existingAnnotations map[string]string
		expectedAnnotations map[string]string
		shouldAdd           bool
	}{
		{
			description:         "pod with no annotations",
			pod:                 makePod(0),
			position:            0,
			existingAnnotations: make(map[string]string),
			expectedAnnotations: fullAnnotationSet(0),
			shouldAdd:           true,
		},
		{
			description:         "pod with existing annotations, no overlap",
			pod:                 makePod(0),
			position:            0,
			existingAnnotations: otherAnnotations,
			expectedAnnotations: utils.MapMerge(otherAnnotations, fullAnnotationSet(0)),
			shouldAdd:           true,
		},
		{
			description:         "pod with existing annotations, some pod overlap",
			pod:                 makePod(0),
			position:            0,
			existingAnnotations: partialPodAnnotationSet(0),
			expectedAnnotations: fullAnnotationSet(0),
			shouldAdd:           true,
		},
		{
			description:         "pod with existing annotations, some image overlap",
			pod:                 makePod(0),
			position:            0,
			existingAnnotations: partialImageAnnotationSet(0),
			expectedAnnotations: fullAnnotationSet(0),
			shouldAdd:           true,
		},
		{
			description:         "pod with exact existing annotations",
			pod:                 makePod(0),
			position:            0,
			existingAnnotations: fullAnnotationSet(0),
			expectedAnnotations: fullAnnotationSet(0),
			shouldAdd:           false,
		},
		{
			description:         "pod with image that hasn't been scanned",
			pod:                 makePodWithImage(0, "imageName", "234F8sdgj235jsdf923"),
			position:            0,
			existingAnnotations: make(map[string]string),
			expectedAnnotations: podAnnotationSet(0),
			shouldAdd:           true,
		},
		{
			description:         "pod with image that hasn't been scanned, existing pod annotations",
			pod:                 makePodWithImage(0, "imageName", "234F8sdgj235jsdf923"),
			position:            0,
			existingAnnotations: podAnnotationSet(0),
			expectedAnnotations: podAnnotationSet(0),
			shouldAdd:           false,
		},
	}

	for _, tc := range testcases {
		annotationObj := makePodAnnotationObj(tc.position)
		tc.pod.SetAnnotations(tc.existingAnnotations)
		result := createPA().addPodAnnotations(tc.pod, annotationObj, scannedImages)
		if result != tc.shouldAdd {
			t.Fatalf("[%s] expected %t, got %t", tc.description, tc.shouldAdd, result)
		}
		updated := tc.pod.GetAnnotations()
		for k, v := range tc.expectedAnnotations {
			if val, ok := updated[k]; !ok {
				t.Errorf("[%s] key %s doesn't exist in pod annotations %v", tc.description, k, updated)
			} else if val != v {
				t.Errorf("[%s] key %s has wrong value in pod annotation.  Expected %s got %s", tc.description, k, tc.expectedAnnotations[k], updated[k])
			}
		}
	}
}

func TestAddPodLabels(t *testing.T) {
	podLabelSet := func(pos int) map[string]string {
		return annotations.CreatePodLabels(makePodAnnotationObj(pos))
	}

	imageLabelSet := func(pos int) map[string]string {
		return annotations.CreateImageLabels(makeImageAnnotationObj(pos), scannedImages[pos].Repository, 0)
	}

	fullLabelSet := func(pos int) map[string]string {
		return utils.MapMerge(podLabelSet(pos), imageLabelSet(pos))
	}

	partialPodLabelSet := func(pos int) map[string]string {
		labels := make(map[string]string)
		for k, v := range podLabelSet(pos) {
			if !strings.Contains(k, "policy-violations") {
				labels[k] = v
			}
		}
		return labels
	}

	partialImageLabelSet := func(pos int) map[string]string {
		labels := make(map[string]string)
		for k, v := range imageLabelSet(pos) {
			if !strings.Contains(k, "vulnerabilities") {
				labels[k] = v
			}
		}
		return labels
	}

	otherLabels := map[string]string{"key1": "value1", "key2": "value2"}

	testcases := []struct {
		description    string
		pod            *v1.Pod
		position       int
		existingLabels map[string]string
		expectedLabels map[string]string
		shouldAdd      bool
	}{
		{
			description:    "pod with no labels",
			pod:            makePod(0),
			position:       0,
			existingLabels: make(map[string]string),
			expectedLabels: fullLabelSet(0),
			shouldAdd:      true,
		},
		{
			description:    "pod with existing labels, no overlap",
			pod:            makePod(0),
			position:       0,
			existingLabels: otherLabels,
			expectedLabels: utils.MapMerge(otherLabels, fullLabelSet(0)),
			shouldAdd:      true,
		},
		{
			description:    "pod with existing labels, some pod overlap",
			pod:            makePod(0),
			position:       0,
			existingLabels: partialPodLabelSet(0),
			expectedLabels: fullLabelSet(0),
			shouldAdd:      true,
		},
		{
			description:    "pod with existing labels, some image overlap",
			pod:            makePod(0),
			position:       0,
			existingLabels: partialImageLabelSet(0),
			expectedLabels: fullLabelSet(0),
			shouldAdd:      true,
		},
		{
			description:    "pod with exact existing labels",
			pod:            makePod(0),
			position:       0,
			existingLabels: fullLabelSet(0),
			expectedLabels: fullLabelSet(0),
			shouldAdd:      false,
		},
		{
			description:    "pod with no scanned images",
			pod:            makePodWithImage(0, "imageName", "234F8sdgj235jsdf923"),
			position:       0,
			existingLabels: make(map[string]string),
			expectedLabels: podLabelSet(0),
			shouldAdd:      true,
		},
		{
			description:    "pod with no scanned images, existing pod labels",
			pod:            makePodWithImage(0, "imageName", "234F8sdgj235jsdf923"),
			position:       0,
			existingLabels: podLabelSet(0),
			expectedLabels: podLabelSet(0),
			shouldAdd:      false,
		},
		{
			description:    "pod with an image that has a registry in the name, but name is under 63 characters",
			pod:            makePod(1),
			position:       1,
			existingLabels: make(map[string]string),
			expectedLabels: fullLabelSet(1),
			shouldAdd:      true,
		},
		{
			description:    "pod with an image that has a registry in the name, but name is longer than 63 characters",
			pod:            makePod(2),
			position:       2,
			existingLabels: make(map[string]string),
			expectedLabels: fullLabelSet(2),
			shouldAdd:      true,
		},
		{
			description:    "pod with an image that has a name that is longer than 63 characters",
			pod:            makePod(3),
			position:       3,
			existingLabels: make(map[string]string),
			expectedLabels: fullLabelSet(3),
			shouldAdd:      true,
		},
		{
			description:    "pod with an image that has a registry with a port in the name",
			pod:            makePod(4),
			position:       4,
			existingLabels: make(map[string]string),
			expectedLabels: fullLabelSet(4),
			shouldAdd:      true,
		},
	}

	for _, tc := range testcases {
		annotationObj := makePodAnnotationObj(tc.position)
		tc.pod.SetLabels(tc.existingLabels)
		result := createPA().addPodLabels(tc.pod, annotationObj, scannedImages)
		if result != tc.shouldAdd {
			t.Fatalf("[%s] expected %t, got %t", tc.description, tc.shouldAdd, result)
		}
		updated := tc.pod.GetLabels()
		for k, v := range tc.expectedLabels {
			if val, ok := updated[k]; !ok {
				t.Errorf("[%s] key %s doesn't exist in pod labels %v", tc.description, k, updated)
			} else if val != v {
				t.Errorf("[%s] key %s has wrong value in pod label.  Expected %s got %s", tc.description, k, tc.expectedLabels[k], updated[k])
			}
			if len(k) > 63 {
				t.Errorf("[%s] key %s is longer than 63 characters", tc.description, k)
			}
			if len(updated[k]) > 63 {
				t.Errorf("[%s] key %s has value %s, which is longer than 63 characters", tc.description, k, updated[k])
			}
		}
		newName := annotations.RemoveRegistryInfo(scannedImages[tc.position].Repository)
		if len(newName) > 63 {
			shortName := newName[0:63]
			if strings.Compare(shortName, updated["image0"]) != 0 {
				t.Errorf("[%s] truncated value %s is wrong, expected %s", tc.description, updated["image0"], shortName)
			}
		}
	}
}

func TestGetPodContainerMap(t *testing.T) {
	generator := func(obj interface{}, name string, count int) map[string]string {
		return map[string]string{fmt.Sprintf("key%d", count): fmt.Sprintf("%s%d", name, count)}
	}
	imageWithoutPrefix := v1.ContainerStatus{
		Name:    "notscanned",
		ImageID: "repository.com/notscanned@sha256:34545ngelkj235knegr",
	}

	imageWithPrefix := v1.ContainerStatus{
		Name:    "notscanned",
		ImageID: "docker-pullable://repository.com/notscanned@sha256:j2345msdf9235nb834",
	}

	testcases := []struct {
		description      string
		pod              *v1.Pod
		additionalImages []v1.ContainerStatus
		resultMap        map[string]string
	}{
		{
			description:      "all containers scanned",
			pod:              makePod(0),
			additionalImages: make([]v1.ContainerStatus, 0),
			resultMap:        map[string]string{"key0": scannedImages[0].Repository + "0"},
		},
		{
			description:      "one container scanned, one not scanned",
			pod:              makePod(0),
			additionalImages: []v1.ContainerStatus{imageWithPrefix},
			resultMap:        map[string]string{"key0": scannedImages[0].Repository + "0"},
		},
		{
			description:      "2 images without scans",
			pod:              &v1.Pod{},
			additionalImages: []v1.ContainerStatus{imageWithPrefix, imageWithoutPrefix},
			resultMap:        make(map[string]string),
		},
	}

	for _, tc := range testcases {
		for _, image := range tc.additionalImages {
			tc.pod.Status.ContainerStatuses = append(tc.pod.Status.ContainerStatuses, image)
		}
		new := createPA().getPodContainerMap(tc.pod, scannedImages, "hub version", "scan client version", generator)
		if !reflect.DeepEqual(new, tc.resultMap) {
			t.Errorf("[%s] container maps are different.  Expected %v got %v", tc.description, tc.resultMap, new)
		}
	}
}

func TestFindImageAnnotations(t *testing.T) {
	testcases := []struct {
		description string
		name        string
		sha         string
		result      *perceptorapi.ScannedImage
	}{
		{
			description: "finds name and sha in scanned images",
			name:        "image1",
			sha:         "ASDJ4FSF3FSFK3SF450",
			result:      &scannedImages[0],
		},
		{
			description: "correct name, wrong sha",
			name:        "image1",
			sha:         "asj23gadgk234",
			result:      nil,
		},
		{
			description: "correct sha, wrong name",
			name:        "notfound",
			sha:         "ASDJ4FSF3FSFK3SF450",
			result:      nil,
		},
		{
			description: "wrong name and sha",
			name:        "notfound",
			sha:         "asj23gadgk234",
			result:      nil,
		},
	}

	for _, tc := range testcases {
		result := createPA().findImageAnnotations(tc.name, tc.sha, scannedImages)
		if result != tc.result && !reflect.DeepEqual(*result, *tc.result) {
			t.Errorf("[%s] expected %v got %v: name %s, sha %s", tc.description, tc.result, result, tc.name, tc.sha)
		}
	}
}

func TestPodAnnotatorAnnotate(t *testing.T) {
	testcases := []struct {
		description string
		statusCode  int
		body        *perceptorapi.ScanResults
		shouldPass  bool
	}{
		{
			description: "successful GET with empty results",
			statusCode:  200,
			body:        &perceptorapi.ScanResults{},
			shouldPass:  true,
		},
		{
			description: "failed to annotate",
			statusCode:  401,
			body:        nil,
			shouldPass:  false,
		},
	}
	endpoint := "RESTEndpoint"
	for _, tc := range testcases {
		bytes, _ := json.Marshal(tc.body)
		handler := utils.FakeHandler{
			StatusCode:  tc.statusCode,
			RespondBody: string(bytes),
			T:           t,
		}
		server := httptest.NewServer(&handler)
		defer server.Close()

		annotator := createPA()
		annotator.scanResultsURL = fmt.Sprintf("%s/%s", server.URL, endpoint)
		err := annotator.annotate()
		if err != nil && tc.shouldPass {
			t.Fatalf("[%s] unexpected error: %v", tc.description, err)
		}
		if err == nil && !tc.shouldPass {
			t.Errorf("[%s] expected error but didn't receive one", tc.description)
		}
	}
}
