package docker

import (
	"fmt"
	"testing"
)

func TestParseImageIDString(t *testing.T) {
	cli, _ := NewDocker()
	swarmServices, _ := cli.ListServices()
	for _, swarmService := range swarmServices {
		image := cli.GetSwarmServiceImage(swarmService)
		fmt.Printf("Image name: %s", image)
	}
}
