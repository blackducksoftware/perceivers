package main

import (
	"fmt"
	"os"

	"github.com/blackducksoftware/perceivers/cmd/quay-perceiver/app"
	"github.com/blackducksoftware/perceivers/pkg/metrics"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("starting quay-perceiver")
	configPath := os.Args[1]
	log.Printf("Config path: %s", configPath)
	metrics.InitMetrics("quay_perceiver")

	// Create the Quay Perceiver
	perceiver, err := app.NewQuayPerceiver(configPath)
	if err != nil {
		panic(fmt.Errorf("failed to create quay-perceiver: %v", err))
	}

	// Run the perceiver
	stopCh := make(chan struct{})
	perceiver.Run(stopCh)
}
