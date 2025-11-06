package dockerclient

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

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
	log.Printf("Fetching currently running containers...")
	containers, err := d.apiClient.ContainerList(context.Background(), container.ListOptions{All: false}) // Only list running containers
	log.Printf("Found '%d' running containers", len(containers))
	if err != nil {
		panic(err)
	}

	ids := []string{}
	for _, container := range containers {
		log.Printf("Found container with ID '%s'", container.ID)
		ids = append(ids, container.ID)
	}
	return ids
}

func (d *DockerClient) CreateContainer(suffix string) (string, error) {
	// Implementation goes here
	resp, err := d.apiClient.ContainerCreate(context.Background(), &container.Config{
		Image: "busybox",
		Cmd:   []string{"sleep", "60"},
	}, nil, nil, nil, "node-"+suffix+randomGUID())
	if err != nil {
		log.Panic(err)
		return "", err
	}
	d.apiClient.ContainerStart(context.Background(), resp.ID, container.StartOptions{})

	log.Printf("Started docker container with ID '%s'", resp.ID)
	return resp.ID, nil
}

func (d *DockerClient) DeleteContainer(containerID string) error {
	log.Printf("Stopping and removing container with ID '%s'", containerID)
	if err := d.apiClient.ContainerStop(context.Background(), containerID, container.StopOptions{}); err != nil {
		return err
	}
	return d.apiClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true, RemoveVolumes: true, RemoveLinks: true})
}

func randomGUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}
