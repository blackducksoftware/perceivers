package utils

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// ArtifactoryCredentials stores the credentials for an artifactory repo
// and is same as common.RegistryAuth in perceptor-scanner repo
type ArtifactoryCredentials struct {
	URL      string
	User     string
	Password string
}

// GetResourceOfType takes in the specified URL with credentials and
// tries to decode returning json to specified interface
func GetResourceOfType(url string, cred *ArtifactoryCredentials, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("Error in creating get request %e at url %s", err, url)
	}
	req.SetBasicAuth(cred.User, cred.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// PingArtifactoryServer takes in the specified URL with username & password and checks weather
// it's a valid login for artifactory by pinging the server
func PingArtifactoryServer(url string, username string, password string) (*ArtifactoryCredentials, error) {
	url = fmt.Sprintf("%s/artifactory/api/system/ping", url)
	log.Infof(url)
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
	return &ArtifactoryCredentials{URL: url, User: username, Password: password}, nil
}
