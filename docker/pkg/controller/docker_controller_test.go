package controller

import (
	"fmt"
	"testing"

	dockerClient "github.com/blackducksoftware/perceivers/docker/pkg/docker"
	docker "github.com/fsouza/go-dockerclient"
)

type Handlers struct {
	handledEvents chan *docker.APIEvents
	handlerFunc   func(event *docker.APIEvents) error
}

func (th *Handlers) Handle(event *docker.APIEvents) error {
	return th.handlerFunc(event)
}

func TestEventHandler(t *testing.T) {
	handledEvents := make(chan *docker.APIEvents, 10)
	hFn := func(event *docker.APIEvents) error {
		handledEvents <- event
		return nil
	}

	handler := &Handlers{
		handlerFunc: hFn,
	}
	handlers := map[string][]Handler{"create": {handler}}

	dockerClient, err := dockerClient.NewDocker()

	if err != nil {
		fmt.Errorf("Unable to initiate the Docker client because %v", err)
	}

	eventHandler := NewEventHandler(10, 10, dockerClient, handlers, "")
	eventHandler.Run()
}
