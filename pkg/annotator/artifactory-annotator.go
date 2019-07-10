package annotator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blackducksoftware/perceivers/pkg/communicator"
	"github.com/blackducksoftware/perceivers/pkg/metrics"
	"github.com/blackducksoftware/perceivers/pkg/utils"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

// BlackDuck Annotation names
const (
	bdPolicy = "blackduck.policyViolations"
	bdVuln   = "blackduck.vulnerabilities"
	bdSt     = "blackduck.overallStatus"
	bdComp   = "blackduck.componentsURL"
)

// ReposBySha collects URIs for given SHA256
type ReposBySha struct {
	Results []struct {
		URI string `json:"uri"`
	} `json:"results"`
}

// ArtifactoryAnnotator handles annotating artifactory images with vulnerability and policy issues
type ArtifactoryAnnotator struct {
	scanResultsURL string
	registryAuths  []*utils.ArtifactoryCredentials
}

// NewArtifactoryAnnotator creates a new ArtifactoryAnnotator object
func NewArtifactoryAnnotator(perceptorURL string, registryAuths []*utils.ArtifactoryCredentials) *ArtifactoryAnnotator {
	return &ArtifactoryAnnotator{
		scanResultsURL: fmt.Sprintf("%s/%s", perceptorURL, perceptorapi.ScanResultsPath),
		registryAuths:  registryAuths,
	}
}

// Run starts a controller that will annotate images
func (ia *ArtifactoryAnnotator) Run(interval time.Duration, stopCh <-chan struct{}) {
	log.Infof("starting image annotator controller")

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		time.Sleep(interval)

		err := ia.annotate()
		if err != nil {
			log.Errorf("failed to annotate images: %v", err)
		}
	}
}

func (ia *ArtifactoryAnnotator) annotate() error {
	// Get all the scan results from the Perceptor
	log.Infof("attempting to GET %s for image annotation", ia.scanResultsURL)
	scanResults, err := ia.getScanResults()
	if err != nil {
		metrics.RecordError("image_annotator", "error getting scan results")
		return fmt.Errorf("error getting scan results: %v", err)
	}

	// Process the scan results and apply annotations/labels to images
	log.Infof("GET to %s succeeded, about to update annotations on all images", ia.scanResultsURL)
	ia.addAnnotationsToImages(*scanResults)
	return nil
}

func (ia *ArtifactoryAnnotator) getScanResults() (*perceptorapi.ScanResults, error) {
	var results perceptorapi.ScanResults

	bytes, err := communicator.GetPerceptorScanResults(ia.scanResultsURL)
	if err != nil {
		metrics.RecordError("image_annotator", "unable to get scan results")
		return nil, fmt.Errorf("unable to get scan results: %v", err)
	}

	err = json.Unmarshal(bytes, &results)
	if err != nil {
		metrics.RecordError("image_annotator", "unable to Unmarshal ScanResults")
		return nil, fmt.Errorf("unable to Unmarshal ScanResults from url %s: %v", ia.scanResultsURL, err)
	}

	return &results, nil
}

func (ia *ArtifactoryAnnotator) addAnnotationsToImages(results perceptorapi.ScanResults) {
	log.Infof("Total Private Registries: %d", len(ia.registryAuths))
	for _, registry := range ia.registryAuths {

		log.Infof("Total images in Artifactory with URL %s: %d", registry.URL, len(results.Images))
		for _, image := range results.Images {

			baseURL := fmt.Sprintf("https://%s", registry.URL)
			cred, err := utils.PingArtifactoryServer(baseURL, registry.User, registry.Password)

			if err != nil {
				log.Warnf("Annotator: URL %s either not a valid Artifactory repository or incorrect credentials: %e", baseURL, err)
				break
			}

			repos := &ReposBySha{}
			// Look for SHA
			url := fmt.Sprintf("%s/artifactory/api/search/checksum?sha256=%s", baseURL, image.Sha)
			err = utils.GetResourceOfType(url, cred, repos)
			if err != nil {
				log.Errorf("Error in getting docker repo: %e", err)
				break
			}

			log.Infof("Total Repos for image %s: %d", image.Repository, len(repos.Results))
			for _, repo := range repos.Results {
				uri := strings.Replace(repo.URI, "/manifest.json", "", -1)
				ia.AnnotateImage(uri, &image, cred)
			}

		}
	}
}

// AnnotateImage takes the specific Artifactory URL and applies the properties/annotations given by BD
func (ia *ArtifactoryAnnotator) AnnotateImage(uri string, im *perceptorapi.ScannedImage, cred *utils.ArtifactoryCredentials) {
	log.Infof("Annotating image %s with URI %s", im.Repository, uri)
	url := fmt.Sprintf("%s?properties=%s=%s;%s=%d;%s=%d;%s=%s;", uri, bdSt, im.OverallStatus, bdVuln, im.Vulnerabilities, bdPolicy, im.PolicyViolations, bdComp, im.ComponentsURL)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		log.Errorf("Error in creating put request %e", err)
	}
	req.SetBasicAuth(cred.User, cred.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Error in sending request %e", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		log.Errorf("Server is supposed to return status code %d given status code %d", http.StatusNoContent, resp.StatusCode)
	} else {
		log.Infof("Properties successfully added/updated for %s:%s", im.Repository, im.Tag)
	}

}
