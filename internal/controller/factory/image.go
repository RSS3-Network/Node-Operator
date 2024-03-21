package factory

import (
	"context"
	"fmt"
	"github.com/google/go-github/v60/github"
	"google.golang.org/appengine/log"
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
	ctx := context.Background()
	log.Debugf(ctx, "Unable to find %s environment variable with the image", imageEnvVar)
	// Get the latest tag from the GitHub repository
	tag, err := getNewestTag(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("rss3/node:%s", tag), nil
}

func getNewestTag(ctx context.Context) (string, error) {
	owner := "rss3-network"
	repo := "node"
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	return *release.TagName, nil
}
