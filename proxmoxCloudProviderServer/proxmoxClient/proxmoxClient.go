package proxmoxClient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

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
		proxmox.WithAPIToken("ansible@pve!lol", "86c30426-2ead-497d-afb6-5f7fa0f9ac81"),
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
			ids = append(ids, fmt.Sprintf("proxmox-node-%d", vm.VMID))
		}
	}
	return ids
}

var MAX_CONCURRENT_SCALING_VMS = 1
var listOfCurrentlyScalingVMs = []int{}

func (d *ProxmoxClient) CreateNode(suffix string) (string, error) {

	if len(listOfCurrentlyScalingVMs) >= MAX_CONCURRENT_SCALING_VMS {
		log.Printf("Too many concurrent scaling VMs: %d/%d", len(listOfCurrentlyScalingVMs), MAX_CONCURRENT_SCALING_VMS)
		return "", fmt.Errorf("maximum concurrent scaling VMs reached, please wait. Currently scaling VM's: %v", listOfCurrentlyScalingVMs)
	}

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
		//await the task of cloning the VM to complete before proceeding

		listOfCurrentlyScalingVMs = append(listOfCurrentlyScalingVMs, createdVMID)

		if err != nil {
			if isVMIDConflictError(err) && attempt < maxRetries-1 {
				continue
			}
			return "", fmt.Errorf("failed to clone VM: %w", err)
		}

		// Start background process to start the VM with retry logic
		createdVM, err := node.VirtualMachine(context.Background(), createdVMID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch cloned VM: %w", err)
		}
		go startVMWithRetry(d, int(createdVM.VMID))
		// MUST match k8s node name format for autoscaler to recognize it,
		// which is currently "proxmox-node-<vmid>"
		return fmt.Sprintf("proxmox-node-%v", vmidCandidate), nil
	}
	return "", fmt.Errorf("failed to create VM after %d attempts", maxRetries)

}

func startVMWithRetry(d *ProxmoxClient, vmID int) {
	log.Printf("Attempting to start VM %d", vmID)

	node, err := d.GetNodeToSpinUpVMs()
	if err != nil {
		log.Printf("Error fetching node for VM %d: %v, retrying in 10s", vmID, err)
		time.Sleep(10 * time.Second)
		go startVMWithRetry(d, vmID)
		return
	}

	vm, err := node.VirtualMachine(context.Background(), vmID)
	if err != nil {
		log.Printf("Error fetching VM %d: %v, retrying in 10s", vmID, err)
		time.Sleep(10 * time.Second)
		go startVMWithRetry(d, vmID)
		return
	}

	if vm.Status != "stopped" {
		log.Printf("VM %d is no longer stopped (status: %s), skipping", vmID, vm.Status)
		return
	}

	task, err := vm.Start(context.Background())
	if err != nil {
		log.Printf("Error starting VM %d: %v, retrying in 10s", vmID, err)
		time.Sleep(10 * time.Second)
		go startVMWithRetry(d, vmID)
		return
	}
	waitTimeInSeconds := 30
	task.WaitFor(context.Background(), waitTimeInSeconds)
	if task.IsFailed {
		log.Printf("Task to start VM %d failed, retrying in 10s", vmID)
		time.Sleep(10 * time.Second)
		go startVMWithRetry(d, vmID)
		return
	}

	log.Printf("Successfully started VM %d", vmID)
}

func (d *ProxmoxClient) FinalizeNodeIfStillListedAsScalingUp(nodeIdAsString string) {
	log.Printf("Finalizing node with proxmox node id '%s' if still listed as scaling up", nodeIdAsString)

	// Convert nodeIdAsString to int
	nodeId, err := strconv.Atoi(nodeIdAsString)
	if err != nil {
		log.Printf("Error converting node ID '%s' to int: %v", nodeIdAsString, err)
		return
	}

	if slices.Contains(listOfCurrentlyScalingVMs, nodeId) {
		log.Printf("Node with proxmox node id '%s' is still listed as scaling up. Finalizing...", nodeIdAsString)
		listOfCurrentlyScalingVMs = slices.DeleteFunc(listOfCurrentlyScalingVMs, func(id int) bool {
			return id == nodeId
		})
		log.Printf("Node with proxmox node id '%s' finalized and removed from scaling list", nodeIdAsString)
	} else {
		log.Printf("Node with proxmox node id '%s' is not listed as scaling up, no action taken", nodeIdAsString)
	}
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

func (d *ProxmoxClient) DeleteNode(nodeID string) error {
	log.Printf("Shutting node with proxmox node id '%s' down", nodeID)

	// Convert nodeID to int
	nodeIdInt, err := strconv.Atoi(nodeID)
	if err != nil {
		log.Printf("Error converting node ID '%s' to int: %v", nodeID, err)
		return err
	}

	node, err := d.GetNodeToSpinUpVMs()
	if err != nil {
		log.Printf("Error fetching node: %v", err)
		return err
	}

	vm, err := node.VirtualMachine(context.Background(), nodeIdInt)
	if err != nil {
		log.Printf("Error fetching VM with ID '%d': %v", nodeIdInt, err)
		return err
	}

	_, err = vm.Shutdown(context.Background())

	go deleteVMWithRetry(d, int(vm.VMID))
	if err != nil {
		log.Printf("Error deleting VM with ID '%d': %v", nodeIdInt, err)
		return err
	}

	log.Printf("Successfully deleted VM with ID '%d'", nodeIdInt)
	return nil
}

func deleteVMWithRetry(d *ProxmoxClient, vmID int) {
	log.Printf("Attempting to delete VM %d", vmID)

	node, err := d.GetNodeToSpinUpVMs()
	if err != nil {
		log.Printf("Error fetching node for VM %d: %v, retrying in 10s", vmID, err)
		time.Sleep(10 * time.Second)
		go deleteVMWithRetry(d, vmID)
		return
	}

	vm, err := node.VirtualMachine(context.Background(), vmID)
	if err != nil {
		log.Printf("Error fetching VM %d: %v, retrying in 10s", vmID, err)
		time.Sleep(10 * time.Second)
		go deleteVMWithRetry(d, vmID)
		return
	}

	if vm.Status != "running" {
		log.Printf("VM %d is no longer running (status: %s), skipping", vmID, vm.Status)
		return
	}

	task, err := vm.Delete(context.Background())
	if err != nil {
		log.Printf("Error deleting VM %d: %v, retrying in 10s", vmID, err)
		time.Sleep(10 * time.Second)
		go deleteVMWithRetry(d, vmID)
		return
	}
	waitTimeInSeconds := 30
	task.WaitFor(context.Background(), waitTimeInSeconds)
	if task.IsFailed {
		log.Printf("Task to delete VM %d failed, retrying in 10s", vmID)
		time.Sleep(10 * time.Second)
		go deleteVMWithRetry(d, vmID)
		return
	}

	log.Printf("Successfully deleted VM %d", vmID)
}
