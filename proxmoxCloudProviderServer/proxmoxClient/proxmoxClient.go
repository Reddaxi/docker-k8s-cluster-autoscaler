package proxmoxClient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"crypto/rand"
	"math/big"

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

func (d *ProxmoxClient) GetNodeToSpinUpVMs() (*proxmox.Node, error) {

	nodeStatusses, err := d.apiClient.Nodes(context.Background())
	if err != nil {
		log.Printf("Error fetching nodes: %v", err)
		return nil, err
	}

	for _, nodeStatus := range nodeStatusses {
		node, err := d.apiClient.Node(context.Background(), nodeStatus.Node)
		if err != nil {
			log.Printf("Error fetching node %s: %v", nodeStatus.Node, err)
			continue
		}

		// Return first available node, since I only have the one.
		return node, nil
	}

	return nil, fmt.Errorf("no nodes available")
}

func (d *ProxmoxClient) GetVMToClone() (*proxmox.VirtualMachine, error) {

	node, err := d.GetNodeToSpinUpVMs()
	if err != nil {
		log.Printf("Error fetching node: %v", err)
		return nil, err
	}

	vms, err := node.VirtualMachines(context.Background())
	if err != nil {
		log.Printf("Error fetching VMs for node %s: %v", node.Name, err)
		return nil, err
	}

	// Return first available VM, since I only have the one.
	for i := 0; i < len(vms); i++ {
		if vms[i].Template == true {
			return vms[i], nil
		}
	}

	return nil, fmt.Errorf("Unable to find template VM to clone from")
}

func (d *ProxmoxClient) ListVMIDs() []string {
	log.Printf("Fetching currently running VMs...")

	ids := []string{}
	node, err := d.GetNodeToSpinUpVMs()
	if err != nil {
		log.Printf("Error fetching node: %v", err)
		return ids
	}

	vms, err := node.VirtualMachines(context.Background())
	if err != nil {
		log.Printf("Error fetching VMs for node %s: %v", node.Name, err)
		return ids
	}

	for _, vm := range vms {
		// Only include running VMs
		log.Printf("status of VM %d is %s", vm.VMID, vm.Status)
		if vm.Status == "running" {
			ids = append(ids, fmt.Sprintf("%d", vm.VMID))
		}
	}
	return ids
}

func (d *ProxmoxClient) CreateContainer(suffix string) (string, error) {

	nameofNewContainer := fmt.Sprintf("%s-%s", suffix, "prxmx-4gb")

	node, err := d.GetNodeToSpinUpVMs()
	if err != nil {
		log.Printf("Error fetching node: %v", err)
		return "", err
	}

	// Try to generate a random VMID and retry if creation fails due to conflict
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Generate a random VMID in a safe range (e.g., 200-9999)
		randID, err := generateRandomVMID(200, 9999)
		if err != nil {
			return "", fmt.Errorf("failed to generate random VMID: %w", err)
		}
		vmidCandidate := randID

		// Use go-proxmox library to clone the VM
		vmToClone, err := d.GetVMToClone()
		if err != nil {
			return "", fmt.Errorf("failed to get VM to clone: %w", err)
		}

		cloneOptions := proxmox.VirtualMachineCloneOptions{
			NewID: randID,
			Name:  nameofNewContainer,
			Full:  1,
		}
		createdVMID, _, err := vmToClone.Clone(context.Background(), &cloneOptions)
		if err != nil {
			if isVMIDConflictError(err) && attempt < maxRetries-1 {
				continue
			}
			return "", fmt.Errorf("failed to clone VM: %w", err)
		}

		// Start the new VM
		createdVM, err := node.VirtualMachine(context.Background(), createdVMID)
		if err != nil {
			return "", fmt.Errorf("failed to start VM: %w", err)
		}

		_, err = createdVM.Start(context.Background())
		if err != nil {
			return "", fmt.Errorf("failed to start VM: %w", err)
		}

		return fmt.Sprintf("%v", vmidCandidate), nil
	}
	return "", fmt.Errorf("failed to create VM after %d attempts", maxRetries)

}

func generateRandomVMID(min, max int) (int, error) {
	// Use crypto/rand for secure random
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}
	return int(nBig.Int64()) + min, nil
}

// Helper to check if error is due to VMID conflict
func isVMIDConflictError(err error) bool {
	// This is a simple string check; adjust as needed for your API's error messages
	return err != nil && (contains(err.Error(), "already exists") || contains(err.Error(), "conflict"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr))))
}

func (d *ProxmoxClient) DeleteContainer(containerID string) error {
	return nil
}
