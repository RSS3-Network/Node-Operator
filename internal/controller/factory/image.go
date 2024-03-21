package factory

import (
	"context"
	"fmt"
	"github.com/google/go-github/v60/github"
	"go.uber.org/zap"
	"os"
)

// imageForHub gets the Operand image which is managed by this controller
// from the NODE_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForNode() (string, error) {
	var imageEnvVar = "NODE_IMAGE"
	image, found := os.LookupEnv(imageEnvVar)
	if found {
		return image, nil
	}
	zap.L().Debug("Unable to find %s environment variable with the image",
		zap.String("envVar", imageEnvVar))
	// Get the latest tag from the GitHub repository
	tag, err := getNewestTag()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("rss3/node:%s", tag), nil
}

func getNewestTag() (string, error) {
	owner := "rss3-network"
	repo := "node"
	client := github.NewClient(nil)
	ctx := context.Background()
	tags, _, err := client.Repositories.ListTags(ctx, owner, repo, nil)
	if err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found for %s/%s", owner, repo)
	}
	return *tags[0].Name, nil
}
