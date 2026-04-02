package proxmoxCloudProviderServer

import (
	"context"
	"fmt"
	"log"
	"mbj-autoscaler/cluster-autoscaler/cloudprovider/externalgrpc/protos"
	proxmoxclient "mbj-autoscaler/proxmoxCloudProviderServer/proxmoxClient"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	v11 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewServer() *CloudProviderServer {
	// Implementation goes here
	log.Printf("Creating new CloudProverServer...")
	server := &CloudProviderServer{}
	proxmoxClient, err := proxmoxclient.NewProxmoxClient()
	if err != nil {
		log.Fatalf("Failed to create Proxmox client: %v", err)
	}
	server.proxmoxClient = proxmoxClient

	// Prinmt out target sizes every 1 minute for debugging purposes
	go func() {
		for {
			log.Printf("Current target sizes: %+v", nodeGroupTargetSizes)
			time.Sleep(1 * time.Minute)
		}
	}()
	return server
}

// CloudProviderServer implements the gRPC CloudProvider service
type CloudProviderServer struct {
	protos.UnimplementedCloudProviderServer
	proxmoxClient *proxmoxclient.ProxmoxClient
}

var raspberryPiNodeGroupName = "raspberry-pi-node-group"
var proxmoxNodeGroupName = "proxmox-virtual-node-group"

var nodeGroups = []*protos.NodeGroup{
	{
		Id:      raspberryPiNodeGroupName,
		MinSize: 4,
		MaxSize: 4,
		Debug:   "This node group represents my physical Raspberry Pi nodes. It cannot scale.",
	},
	{
		Id:      proxmoxNodeGroupName,
		MinSize: 0,
		MaxSize: 3,
		Debug:   "This node group is managed by my custom proxmox cloud provider and allows for assigning containers running on an old work laptop as nodes. If it wasn't obvious before, this is for learning purposes.",
	},
}

// nodeGroupTargetSizes tracks the current target size for each node group
var nodeGroupTargetSizes = map[string]int32{
	raspberryPiNodeGroupName: 4, // Fixed-size physical node group
	proxmoxNodeGroupName:     0, // Start at 0; autoscaler will increase when pods are unschedulable
}

var externalGRPCNodeCache = map[string]*protos.ExternalGrpcNode{}

func (CloudProviderServer) NodeGroups(context.Context, *protos.NodeGroupsRequest) (*protos.NodeGroupsResponse, error) {
	response := &protos.NodeGroupsResponse{
		NodeGroups: nodeGroups,
	}
	return response, nil
}
func (cp CloudProviderServer) NodeGroupForNode(c context.Context, r *protos.NodeGroupForNodeRequest) (*protos.NodeGroupForNodeResponse, error) {

	var nodeGroup *protos.NodeGroup = nil // Default to raspberry pi node group
	var node = r.Node
	// Use cached node if we have one
	cachedNode, nodeExistsInCache := externalGRPCNodeCache[r.Node.ProviderID]
	if nodeExistsInCache {
		node = cachedNode
	}

	// Check node has label
	if value, ok := node.Labels["daxi/node-group"]; ok {
		nodeGroup = findNodeGroupByID(value)
	}

	// Check node has label
	if value, ok := node.Labels["daxi/proxmox-node-id"]; ok {
		cp.proxmoxClient.FinalizeNodeIfStillListedAsScalingUp(value)
		externalGRPCNodeCache[r.Node.ProviderID] = node
	}

	if nodeGroup == nil {
		nodeGroup = findNodeGroupByID(nodeGroups[0].Id)
	}

	return &protos.NodeGroupForNodeResponse{
		NodeGroup: nodeGroup,
	}, nil
}
func (CloudProviderServer) PricingNodePrice(context.Context, *protos.PricingNodePriceRequest) (*protos.PricingNodePriceResponse, error) {
	return &protos.PricingNodePriceResponse{
		Price: 0.0, // Since this is a custom implementation, we can return a dummy price or calculate based on some logic
	}, nil
}
func (CloudProviderServer) PricingPodPrice(context.Context, *protos.PricingPodPriceRequest) (*protos.PricingPodPriceResponse, error) {
	return &protos.PricingPodPriceResponse{
		Price: 0.0, // Since this is a custom implementation, we can return a dummy price or calculate based on some logic
	}, nil
}
func (CloudProviderServer) GPULabel(context.Context, *protos.GPULabelRequest) (*protos.GPULabelResponse, error) {
	return &protos.GPULabelResponse{}, nil
}
func (CloudProviderServer) GetAvailableGPUTypes(context.Context, *protos.GetAvailableGPUTypesRequest) (*protos.GetAvailableGPUTypesResponse, error) {
	return &protos.GetAvailableGPUTypesResponse{
		GpuTypes: map[string]*anypb.Any{},
	}, nil
}
func (CloudProviderServer) Cleanup(context.Context, *protos.CleanupRequest) (*protos.CleanupResponse, error) {
	log.Printf("Cleanup was called. It did nothing")
	return &protos.CleanupResponse{}, nil
}
func (c *CloudProviderServer) Refresh(context.Context, *protos.RefreshRequest) (*protos.RefreshResponse, error) {
	log.Printf("Refresh was called.")

	// Verify that the amount of running vms is the same as the target size for the proxmox node group. If there are more running vms than the target size, it means some VMs were created outside of the autoscaler and we should delete them to prevent resource waste. If there are fewer running vms than the target size, it means some VMs were deleted outside of the autoscaler and we should increase the target size to trigger the autoscaler to create new VMs.
	runningVMIDs := c.proxmoxClient.ListVMIDs()
	currentTargetSize := nodeGroupTargetSizes[proxmoxNodeGroupName]
	if int(currentTargetSize) == len(runningVMIDs) {
		log.Printf("Current target size matches amount of running VMs. No action needed. Target size: %d, Running VMs: %d", currentTargetSize, len(runningVMIDs))
	}
	if int32(len(runningVMIDs)) > currentTargetSize {
		log.Printf("There are %d running VMs but the target size is only %d. Deleting orphaned VMs to prevent resource waste.", len(runningVMIDs), currentTargetSize)
		for _, vmID := range runningVMIDs {
			c.proxmoxClient.DeleteNode(vmID)
		}
	} else if int32(len(runningVMIDs)) < currentTargetSize {
		log.Printf("There are only %d running VMs but the target size is %d. Updating target size to trigger autoscaler to create new VMs.", len(runningVMIDs), currentTargetSize)
		nodeGroupTargetSizes[proxmoxNodeGroupName] = int32(len(runningVMIDs))
	}
	return &protos.RefreshResponse{}, nil
}

func (CloudProviderServer) NodeGroupTargetSize(c context.Context, r *protos.NodeGroupTargetSizeRequest) (*protos.NodeGroupTargetSizeResponse, error) {
	nodeGroup := findNodeGroupByID(r.Id)
	if nodeGroup == nil {
		return nil, status.Errorf(codes.NotFound, "node group not found")
	}

	targetSize, exists := findTargetSizeByID(r.Id)
	if !exists {
		// Default to minimum size if not explicitly set
		targetSize = nodeGroup.MinSize
		nodeGroupTargetSizes[r.Id] = targetSize
	}

	return &protos.NodeGroupTargetSizeResponse{
		TargetSize: targetSize,
	}, nil
}

func (c *CloudProviderServer) NodeGroupIncreaseSize(context context.Context, r *protos.NodeGroupIncreaseSizeRequest) (*protos.NodeGroupIncreaseSizeResponse, error) {
	nodeGroup := findNodeGroupByID(r.Id)
	log.Printf("++ Increasing ++ target size for node group '%s' by %d. It's current target size is '%d'. It's min and max are '%d' and '%d'", r.Id, r.Delta, nodeGroupTargetSizes[r.Id], nodeGroup.MinSize, nodeGroup.MaxSize)
	if nodeGroup == nil {
		return nil, status.Errorf(codes.NotFound, "node group not found")
	}

	if nodeGroupTargetSizes[r.Id]+r.Delta > nodeGroup.MaxSize {
		log.Printf("Cannot increase size for node group '%s' beyond its maximum size of %d. Current target size is %d. Requested increase is %d.", r.Id, nodeGroup.MaxSize, nodeGroupTargetSizes[r.Id], r.Delta)
		return nil, fmt.Errorf("cannot increase size for node group '%s' beyond its maximum size of %d", r.Id, nodeGroup.MaxSize)
	}

	for i := 0; i < int(r.Delta); i++ {
		_, err := c.proxmoxClient.CreateNode(fmt.Sprintf("%s-%d", r.Id, i))
		if err != nil {
			return nil, err
		}
	}

	nodeGroupTargetSizes[r.Id] += r.Delta
	log.Printf("++ Increased ++ target size for node group '%s' by %d. New target size is %d.", r.Id, r.Delta, nodeGroupTargetSizes[r.Id])
	return &protos.NodeGroupIncreaseSizeResponse{}, nil
}

func (c *CloudProviderServer) NodeGroupDeleteNodes(context context.Context, r *protos.NodeGroupDeleteNodesRequest) (*protos.NodeGroupDeleteNodesResponse, error) {
	log.Printf("Deleting nodes '%d' from node group '%s'", len(r.Nodes), r.Id)

	for _, node := range r.Nodes {

		var nodeToDelete = node
		// Use cached node if we have one
		cachedNode, nodeExistsInCache := externalGRPCNodeCache[node.ProviderID]
		if nodeExistsInCache {
			nodeToDelete = cachedNode
		}

		// Check node has label
		if value, ok := nodeToDelete.Labels["daxi/proxmox-node-id"]; ok {
			log.Printf("Node '%s' has label 'daxi/proxmox-node-id' with value '%s'", nodeToDelete.Name, value)
			c.proxmoxClient.DeleteNode(value)
		} else {
			log.Printf("Node '%s' does not have label 'daxi/proxmox-node-id'. Cannot determine which proxmox VM to delete.", nodeToDelete.Name)
		}
	}

	nodeGroupTargetSizes[r.Id] -= int32(len(r.Nodes))
	return &protos.NodeGroupDeleteNodesResponse{}, nil
}

func (CloudProviderServer) NodeGroupDecreaseTargetSize(c context.Context, r *protos.NodeGroupDecreaseTargetSizeRequest) (*protos.NodeGroupDecreaseTargetSizeResponse, error) {
	nodeGroup := findNodeGroupByID(r.Id)
	log.Printf("-- Decreasing -- target size for node group '%s' by %d. It's current target size is '%d'. It's min and max are '%d' and '%d'", r.Id, r.Delta, nodeGroupTargetSizes[r.Id], nodeGroup.MinSize, nodeGroup.MaxSize)
	newValue := nodeGroupTargetSizes[r.Id] - r.Delta
	if newValue < nodeGroup.MinSize {
		log.Printf("Cannot decrease target size for node group '%s' below its minimum size of %d. Current target size is %d. Requested decrease is %d.", r.Id, nodeGroup.MinSize, nodeGroupTargetSizes[r.Id], r.Delta)
		return nil, fmt.Errorf("cannot decrease target size for node group '%s' below its minimum size of %d", r.Id, nodeGroup.MinSize)
	}

	nodeGroupTargetSizes[r.Id] = newValue
	log.Printf("-- Decreased -- target size for node group '%s' by %d. New target size is %d.", r.Id, r.Delta, nodeGroupTargetSizes[r.Id])
	return &protos.NodeGroupDecreaseTargetSizeResponse{}, nil
}

func (c CloudProviderServer) NodeGroupNodes(context context.Context, r *protos.NodeGroupNodesRequest) (*protos.NodeGroupNodesResponse, error) {
	if r.Id == nodeGroups[0].Id {
		return &protos.NodeGroupNodesResponse{
			Instances: []*protos.Instance{},
		}, nil
	}

	localContainerIDs := c.proxmoxClient.ListVMIDs()
	nodesInNodeGroup := []*protos.Instance{}
	for _, containerID := range localContainerIDs {

		instance := &protos.Instance{
			Id:     containerID,
			Status: &protos.InstanceStatus{InstanceState: protos.InstanceStatus_instanceRunning},
		}

		if instance != nil {
			nodesInNodeGroup = append(nodesInNodeGroup, instance)
		}
	}

	return &protos.NodeGroupNodesResponse{
		Instances: nodesInNodeGroup,
	}, nil
}
func (CloudProviderServer) NodeGroupTemplateNodeInfo(c context.Context, r *protos.NodeGroupTemplateNodeInfoRequest) (*protos.NodeGroupTemplateNodeInfoResponse, error) {
	var cpu, err = resource.ParseQuantity("1")
	if err != nil {
		log.Printf("Failed to parse CPU quantity: %v", err)
		return nil, err
	}

	capacity := v11.ResourceList{
		v11.ResourceCPU:              cpu,
		v11.ResourceMemory:           resource.MustParse("4Gi"),
		v11.ResourceEphemeralStorage: resource.MustParse("10Gi"),
		v11.ResourcePods:             resource.MustParse("110"),
	}

	var nodeGroup = findNodeGroupByID(r.Id)
	var cpuArchitecture = "amd64"
	if r.Id == raspberryPiNodeGroupName {
		cpuArchitecture = "arm64"
	}

	node := &v11.Node{
		ObjectMeta: v1.ObjectMeta{
			Name: nodeGroup.Id,
			Labels: map[string]string{
				"kubernetes.io/arch": cpuArchitecture,
			},
		},
		Status: v11.NodeStatus{
			Capacity:    capacity,
			Allocatable: capacity,
			Conditions: []v11.NodeCondition{
				{
					Type:   v11.NodeReady,
					Status: v11.ConditionTrue,
				},
			},
		},
	}

	nodeBytes, err := node.Marshal()
	if err != nil {
		log.Printf("Failed to marshal node: %v", err)
		return nil, err
	}

	return &protos.NodeGroupTemplateNodeInfoResponse{
		NodeBytes: nodeBytes,
	}, nil
}
func (CloudProviderServer) NodeGroupGetOptions(context.Context, *protos.NodeGroupAutoscalingOptionsRequest) (*protos.NodeGroupAutoscalingOptionsResponse, error) {

	// oneSecondInNanoseconds := 1
	// oneMinute := 60 * oneSecondInNanoseconds
	// fiveMinutes := 5 * oneMinute
	tenSeconds := 10

	return &protos.NodeGroupAutoscalingOptionsResponse{
		NodeGroupAutoscalingOptions: &protos.NodeGroupAutoscalingOptions{
			ScaleDownUtilizationThreshold:    0.5,
			ScaleDownUnneededDuration:        &duration.Duration{Seconds: int64(tenSeconds)}, // Time in seconds before considering a node unneeded
			ScaleDownUnreadyDuration:         &duration.Duration{Seconds: int64(tenSeconds)}, // Time in seconds before considering a node unready
			ScaleDownGpuUtilizationThreshold: 0.5,                                            // Example GPU utilization threshold for scaling down
		},
	}, nil
}

func findTargetSizeByID(id string) (int32, bool) {
	size, exists := nodeGroupTargetSizes[id]
	return size, exists
}

func findNodeGroupByID(id string) *protos.NodeGroup {
	for _, ng := range nodeGroups {
		if ng.Id == id {
			return ng
		}
	}
	return nil
}
