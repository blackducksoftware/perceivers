package docker

import (
	"github.com/docker/docker/api/types/swarm"
	dockerClient "github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
)

type Docker struct {
	Client *dockerClient.Client
}

func (docker *Docker) ListServices() (swarmServices []swarm.Service, err error) {
	log.Printf("List Swarm services started")
	swarmService, err := docker.Client.ListServices(dockerClient.ListServicesOptions{})
	if err != nil {
		log.Printf("Swarm list services failed because %v \n", err)
		return swarmService, err
	}
	log.Printf("Swarm service count: %d \n", len(swarmService))
	return swarmService, nil
}

func (docker *Docker) GetServices(id string) (swarmServices *swarm.Service, err error) {
	swarmService, err := docker.Client.InspectService(id)
	log.Printf("Swarm service image name: %s \n", swarmService.Spec.TaskTemplate.ContainerSpec.Image)
	return swarmService, err
}

func (docker *Docker) GetSwarmServiceImage(swarmService swarm.Service) string {
	log.Printf("Swarm version: %v \n", swarmService.Spec.TaskTemplate.ContainerSpec.Image)
	return swarmService.Spec.TaskTemplate.ContainerSpec.Image
}

func (docker *Docker) UpdateServices(swarmService *swarm.Service, labels map[string]string) error {
	log.Printf("Update swarm service for image: %s \n", swarmService.Spec.TaskTemplate.ContainerSpec.Image)
	mergedLabels := []map[string]string{swarmService.Spec.TaskTemplate.ContainerSpec.Labels, labels}
	labels = mergeMaps(mergedLabels...)
	taskTemplate := swarmService.Spec.TaskTemplate
	taskTemplate.ContainerSpec.Labels = labels
	err := docker.Client.UpdateService(swarmService.ID, dockerClient.UpdateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			TaskTemplate: taskTemplate,
		},
	})
	return err
}

func mergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// func (docker *Docker) WatchEvents() {
// 	events := make(chan *dockerClient.APIEvents)
//
// 	docker_events := make(chan *dockerClient.APIEvents)
// 	go func() {
// 		/* step: add our channel as an event listener for docker events */
// 		if err := docker.client.AddEventListener(docker_events); err != nil {
// 			log.Fatalf("Unable to register docker events listener, error: %s", err)
// 			return
// 		}
// 		/* step: start the event loop and wait for docker events */
// 		log.Infof("Entering into the docker events loop")
// 		for {
// 			select {
// 			case event := <-docker_events:
// 				log.Infof("Received docker event status: %s, id: %s", event.Status, event.ID)
// 				switch event.Status {
// 				case DOCKER_START:
// 					log.Infof("Process creation")
// 					// go r.ProcessDockerCreation(event.ID, channel)
// 				case DOCKER_DESTROY:
// 					log.Infof("Process destroy")
// 					// go r.ProcessDockerDestroy(event.ID, channel)
// 				}
// 			}
// 		}
// 		log.Errorf("Exitting the docker events loop")
// 	}()
// }

func NewDocker() (cli *Docker, err error) {

	endpoint := "unix:///var/run/docker.sock"
	client, err := dockerClient.NewVersionedClient(endpoint, "1.24")

	return &Docker{
		Client: client,
	}, err
}
