package app

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blackducksoftware/perceivers/pkg/annotator"
	"github.com/blackducksoftware/perceivers/pkg/controller"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// ArtifactoryPerceiver handles watching and annotating Images
type ArtifactoryPerceiver struct {
	controller         *controller.ArtifactoryController
	annotator          *annotator.ArtifactoryAnnotator
	annotationInterval time.Duration
	dumpInterval       time.Duration
	metricsURL         string
}

// NewArtifactoryPerceiver creates a new ImagePerceiver object
func NewArtifactoryPerceiver(configPath string) (*ArtifactoryPerceiver, error) {
	config, err := GetConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	// Configure prometheus for metrics
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())
	http.Handle("/metrics", prometheus.Handler())

	perceptorURL := fmt.Sprintf("http://%s:%d", config.Perceptor.Host, config.Perceptor.Port)
	ap := ArtifactoryPerceiver{
		controller:         controller.NewArtifactoryController(perceptorURL, config.PrivateDockerRegistries),
		annotator:          annotator.NewArtifactoryAnnotator(perceptorURL, config.PrivateDockerRegistries),
		annotationInterval: time.Second * time.Duration(config.Perceiver.AnnotationIntervalSeconds),
		dumpInterval:       time.Minute * time.Duration(config.Perceiver.DumpIntervalMinutes),
		metricsURL:         fmt.Sprintf(":%d", config.Perceiver.Port),
	}
	return &ap, nil
}

// Run starts the ArtifactoryPerceiver watching and annotating Images
func (ap *ArtifactoryPerceiver) Run(stopCh <-chan struct{}) {
	log.Infof("starting artifactory controllers")
	go ap.controller.Run(ap.dumpInterval, stopCh)
	go ap.annotator.Run(ap.annotationInterval, stopCh)

	log.Infof("starting prometheus on %s", ap.metricsURL)
	http.ListenAndServe(ap.metricsURL, nil)

	<-stopCh
}
