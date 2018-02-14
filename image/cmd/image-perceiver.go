package main

import (
	"fmt"

	"github.com/blackducksoftware/perceivers/image/cmd/app"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("starting image-perceiver")

	config, err := app.GetImagePerceiverConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %v", err))
	}

	// Create the Image Perceiver
	perceiver, err := app.NewImagePerceiver(config)
	if err != nil {
		panic(fmt.Errorf("failed to create image-perceiver: %v", err))
	}

	// Run the perceiver
	stopCh := make(chan struct{})
	perceiver.Run(stopCh)
}
