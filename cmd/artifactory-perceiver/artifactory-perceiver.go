package main

import (
	"fmt"
	"os"

	"github.com/blackducksoftware/perceivers/cmd/artifactory-perceiver/app"
	"github.com/blackducksoftware/perceivers/pkg/metrics"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("starting artifactory-perceiver")
	configPath := os.Args[1]
	log.Printf("Config path: %s", configPath)
	metrics.InitMetrics("artifactory_perceiver")

	// Create the Artifactory Perceiver
	perceiver, err := app.NewArtifactoryPerceiver(configPath)
	if err != nil {
		panic(fmt.Errorf("failed to create image-perceiver: %v", err))
	}

	// Run the perceiver
	stopCh := make(chan struct{})
	perceiver.Run(stopCh)
}
