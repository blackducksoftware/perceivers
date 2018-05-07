package controller

import (
	"fmt"
	"testing"

	"github.com/fsouza/go-dockerclient"
)

type testHandler struct {
	handledEvents chan *docker.APIEvents
	t             *testing.T
	handlerFunc   func(event *docker.APIEvents) error
}

func (th *testHandler) Handle(event *docker.APIEvents) error {
	return th.handlerFunc(event)
}

func TestEventHandler(t *testing.T) {
	handledEvents := make(chan *docker.APIEvents, 10)
	hFn := func(event *docker.APIEvents) error {
		handledEvents <- event
		return nil
	}

	handler := &testHandler{
		handlerFunc: hFn,
	}
	handlers := map[string][]Handler{"create": {handler}}

	endpoint := "unix:///var/run/docker.sock"
	dockerClient, err := docker.NewVersionedClient(endpoint, "1.24")
	if err != nil {
		fmt.Printf("Unable to get the Docker client because %v", err)
	}

	//dockerClient, _ := docker.NewClientFromEnv()
	router, _ := NewEventHandler(10, 10, dockerClient, handlers)

	defer router.Stop()
	router.Start()
}
