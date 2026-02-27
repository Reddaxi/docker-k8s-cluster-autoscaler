package proxmoxClient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxClient struct {
	apiClient *proxmox.Client
}

func NewProxmoxClient() (*ProxmoxClient, error) {

	insecureHTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := proxmox.NewClient("https://192.168.0.123:8006/api2/json",
		proxmox.WithAPIToken("ansible@pve!lol", "36455984-a3ae-4714-9be0-3f35f6d0b091"),
		proxmox.WithHTTPClient(&insecureHTTPClient),
	)

	version, err := client.Version(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(version.Release) // 7.4
	return &ProxmoxClient{apiClient: client}, nil
}

func (d *ProxmoxClient) ListContainers() []string {
	log.Printf("Fetching currently running containers...")

	ids := []string{}
	return ids
}

func (d *ProxmoxClient) CreateContainer(suffix string) (string, error) {
	// Implementation goes here

	return "101", nil
}

func (d *ProxmoxClient) DeleteContainer(containerID string) error {
	return nil
}
