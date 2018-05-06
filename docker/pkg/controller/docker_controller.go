package controller

import (
	"time"

	"github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
)

const workerTimeout = 60 * time.Second

type Handler interface {
	Handle(*docker.APIEvents) error
}

type EventHandler struct {
	handlers      map[string][]Handler
	dockerClient  *docker.Client
	listener      chan *docker.APIEvents
	workers       chan *worker
	workerTimeout time.Duration
}

func NewEventHandler(bufferSize int, workerPoolSize int, dockerClient *docker.Client, handlers map[string][]Handler) (*EventHandler, error) {
	workers := make(chan *worker, workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		workers <- &worker{}
	}

	eventHandler := &EventHandler{
		handlers:      handlers,
		dockerClient:  dockerClient,
		listener:      make(chan *docker.APIEvents, bufferSize),
		workers:       workers,
		workerTimeout: workerTimeout,
	}

	return eventHandler, nil
}

func (e *EventHandler) Start() error {
	log.Info("Starting event router.")
	go e.handleEvents()
	if err := e.dockerClient.AddEventListener(e.listener); err != nil {
		return err
	}
	return nil
}

func (e *EventHandler) Stop() error {
	if e.listener == nil {
		return nil
	}
	if err := e.dockerClient.RemoveEventListener(e.listener); err != nil {
		return err
	}
	return nil
}

func (e *EventHandler) handleEvents() {
	for {
		event := <-e.listener
		timer := time.NewTimer(e.workerTimeout)
		gotWorker := false
		for !gotWorker {
			select {
			case w := <-e.workers:
				go w.processDockerEvents(event, e)
				gotWorker = true
			case <-timer.C:
				log.Infof("Timed out waiting for worker. Re-initializing wait.")
			}
		}
	}
}

type worker struct{}

func (w *worker) processDockerEvents(event *docker.APIEvents, e *EventHandler) {
	defer func() {
		e.workers <- w
	}()
	if event == nil {
		return
	}
	if handlers, ok := e.handlers[event.Status]; ok {
		log.Infof("Processing docker event: %#v", event)
		for _, handler := range handlers {
			if err := handler.Handle(event); err != nil {
				log.Errorf("Error processing event %#v. Error: %v", event, err)
			}
		}
	}
}
