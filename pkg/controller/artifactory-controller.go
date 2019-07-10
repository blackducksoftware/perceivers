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

// ImageMetadata gets the info about the image
type ImageMetadata struct {
	Repo         string `json:"repo"`
	Path         string `json:"path"`
	Created      string `json:"created"`
	CreatedBy    string `json:"createdBy"`
	LastModified string `json:"lastModified"`
	ModifiedBy   string `json:"modifiedBy"`
	LastUpdated  string `json:"lastUpdated"`
	DownloadURI  string `json:"downloadUri"`
	MimeType     string `json:"mimeType"`
	Size         string `json:"size"`
	Checksums    struct {
		Sha1   string `json:"sha1"`
		Md5    string `json:"md5"`
		Sha256 string `json:"sha256"`
	} `json:"checksums"`
	OriginalChecksums struct {
		Sha256 string `json:"sha256"`
	} `json:"originalChecksums"`
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
		imageMetadata := &ImageMetadata{}

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
					url = fmt.Sprintf("%s/artifactory/api/storage/%s/%s/%s/manifest.json", baseURL, repo.Key, image, tag)
					err = utils.GetResourceOfType(url, cred, imageMetadata)
					if err != nil {
						log.Errorf("Error in getting metadata: %e", err)
						break
					}

					url = fmt.Sprintf("%s/%s:%s", baseURL, image, tag)
					log.Infof("URL: %s", url)
					log.Infof("Tag: %s", tag)
					log.Infof("SHA: %s", imageMetadata.OriginalChecksums.Sha256)
					log.Infof("Priority: %d", 1)
					log.Infof("BlackDuckProjectName: %s", image)
					log.Infof("BlackDuckProjectVersion: %s", tag)

					sha, err := m.NewDockerImageSha(imageMetadata.OriginalChecksums.Sha256)
					if err != nil {
						log.Errorf("Error in docker SHA: %e", err)
					} else {

						// Remove Tag & HTTPS because image model doesn't require it
						url = fmt.Sprintf("%s/%s/%s", registry.URL, repo.Key, image)
						artImage := m.NewImage(url, tag, sha, 0, image, tag)
						ic.putImageOnScanQueue(artImage, cred)
					}
				}
			}
		}

		log.Infof("There were total %d images found in artifactory.", len(images.Repositories))

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
