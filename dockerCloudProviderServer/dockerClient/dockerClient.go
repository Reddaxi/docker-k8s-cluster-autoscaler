package dockerclient

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	apiClient *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return &DockerClient{apiClient: apiClient}, nil
}

func (d *DockerClient) ListContainers() []string {
	// Implementation goes here
	containers, err := d.apiClient.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}

	ids := []string{}
	for _, container := range containers {
		fmt.Println(container.ID)
		ids = append(ids, container.ID)
	}
	return ids
}
