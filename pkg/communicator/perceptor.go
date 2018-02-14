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
		return fmt.Errorf("unable to serialize %v: %v", obj, err)
	}
	resp, err := http.Post(dest, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("unable to POST to %s: %v", dest, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("http POST request to %s failed with status code %d", dest, resp.StatusCode)
	}
	return nil
}

// SendPerceptorDeleteEvent sends a delete event to perceptor at the dest endpoint
func SendPerceptorDeleteEvent(dest string, name string) error {
	jsonBytes, err := json.Marshal(name)
	if err != nil {
		return fmt.Errorf("unable to serialize %s: %v", name, err)
	}
	req, err := http.NewRequest("DELETE", dest, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("unable to create DELETE request for %s: %v", dest, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to DELETE to %s: %v", dest, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("http DELETE request to %s failed with status code %d", dest, resp.StatusCode)
	}
	return nil
}
