package mapper

import (
	"fmt"

	"github.com/blackducksoftware/perceivers/pkg/docker"

	perceptorapi "github.com/blackducksoftware/perceptor/pkg/api"

	imageapi "github.com/openshift/api/image/v1"
)

// NewPerceptorPodFromOSImage will convert an openshift image object to a
// perceptor image object
func NewPerceptorImageFromOSImage(image *imageapi.Image) (*perceptorapi.Image, error) {
	dockerRef := image.DockerImageReference
	name, sha, err := docker.ParseImageIDString(dockerRef)
	if err != nil {
		return nil, fmt.Errorf("unable to parse openshift imageID %s from image %s: %v", dockerRef, name, err)
	}

	return perceptorapi.NewImage(name, sha, dockerRef), nil
}
