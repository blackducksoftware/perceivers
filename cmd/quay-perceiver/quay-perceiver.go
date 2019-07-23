package main

import (
	"encoding/json"
	"fmt"

	"github.com/blackducksoftware/perceivers/pkg/metrics"

	"net/http"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("starting quay-perceiver")
	// configPath := os.Args[1]
	// log.Printf("Config path: %s", configPath)
	metrics.InitMetrics("quay_perceiver")

	// Create the Quay Perceiver
	// perceiver, err := app.NewArtifactoryPerceiver(configPath)
	// if err != nil {
	// 	panic(fmt.Errorf("failed to create image-perceiver: %v", err))
	// }

	// Run the perceiver
	// stopCh := make(chan struct{})
	// perceiver.Run(stopCh)

	http.HandleFunc("/webhook", webhook)

	fmt.Printf("Starting server for testing HTTP POST...\n")
	if err := http.ListenAndServe(":443", nil); err != nil {
		log.Fatal(err)
	}
}

type QuayRepo struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Namespace   string   `json:"namespace"`
	DockerURL   string   `json:"docker_url"`
	Homepage    string   `json:"homepage"`
	UpdatedTags []string `json:"updated_tags"`
}

func webhook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			log.Info("Error Parse prm")
			return
		}
		log.Info("Post Req succ")
		qr := &QuayRepo{}
		json.NewDecoder(r.Body).Decode(qr)

		log.Info(qr.Name)
		fmt.Fprintf(w, "Name = %s\n", qr.Name)
		fmt.Fprintf(w, "Repo = %s\n", qr.Repository)
		fmt.Fprintf(w, "Namespace = %s\n", qr.Namespace)
		fmt.Fprintf(w, "Docker URL = %s\n", qr.DockerURL)
		fmt.Fprintf(w, "Homepage = %s\n", qr.Homepage)
		fmt.Fprintf(w, "updated_tags = %s\n", qr.UpdatedTags)
	default:
		log.Info("Sorry, only POST requests are supported.")
	}
}
