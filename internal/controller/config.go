package controller

import (
	"fmt"
	"os"
)

// imageForHub gets the Operand image which is managed by this controller
// from the NODE_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForNode() (string, error) {
	var imageEnvVar = "NODE_IMAGE"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		return "", fmt.Errorf("Unable to find %s environment variable with the image", imageEnvVar)
	}
	return image, nil
}
