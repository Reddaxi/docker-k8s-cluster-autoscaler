package dockerCloudProviderServer

import (
	"context"
	"fmt"
	"log"
	"mbj-autoscaler/cluster-autoscaler/cloudprovider/externalgrpc/protos"
	dockerclient "mbj-autoscaler/dockerCloudProviderServer/dockerClient"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewServer() CloudProviderServer {
	// Implementation goes here
	log.Printf("Creating new CloudProverServer...")
	server := &CloudProviderServer{}
	dockerClient, err := dockerclient.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	server.dockerClient = dockerClient
	currentContainerIds := server.dockerClient.ListContainers()
	for _, id := range currentContainerIds {
		addDockerContainerAsNode(server, id, nodeGroups[0].Id)
	}

	return *server
}

// CloudProviderServer implements the gRPC CloudProvider service
type CloudProviderServer struct {
	protos.UnimplementedCloudProviderServer
	dockerClient *dockerclient.DockerClient
	instances    []*protos.Instance
}

var nodeGroups = []*protos.NodeGroup{
	{
		Id:      "docker-virtual-node-group",
		MinSize: 1,
		MaxSize: 3,
		Debug:   "This node group is managed by my custom docker cloud provider and allows for assigning containers running on an old work laptop as nodes. If it wasn't obvious before, this is for learning purposes.",
	},
}

// nodeGroupTargetSizes tracks the current target size for each node group
var nodeGroupTargetSizes = map[string]int32{
	"docker-virtual-node-group": 1, // Start with 1 node as default
}

func (CloudProviderServer) NodeGroups(context.Context, *protos.NodeGroupsRequest) (*protos.NodeGroupsResponse, error) {
	response := &protos.NodeGroupsResponse{
		NodeGroups: nodeGroups,
	}
	return response, nil
}
func (CloudProviderServer) NodeGroupForNode(c context.Context, r *protos.NodeGroupForNodeRequest) (*protos.NodeGroupForNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupForNode not implemented")
}
func (CloudProviderServer) PricingNodePrice(context.Context, *protos.PricingNodePriceRequest) (*protos.PricingNodePriceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PricingNodePrice not implemented")
}
func (CloudProviderServer) PricingPodPrice(context.Context, *protos.PricingPodPriceRequest) (*protos.PricingPodPriceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PricingPodPrice not implemented")
}
func (CloudProviderServer) GPULabel(context.Context, *protos.GPULabelRequest) (*protos.GPULabelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GPULabel not implemented")
}
func (CloudProviderServer) GetAvailableGPUTypes(context.Context, *protos.GetAvailableGPUTypesRequest) (*protos.GetAvailableGPUTypesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAvailableGPUTypes not implemented")
}
func (CloudProviderServer) Cleanup(context.Context, *protos.CleanupRequest) (*protos.CleanupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Cleanup not implemented")
}
func (CloudProviderServer) Refresh(context.Context, *protos.RefreshRequest) (*protos.RefreshResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Refresh not implemented")
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
	log.Printf("Increasing size for node group '%s' by %d", r.Id, r.Delta)

	for i := 0; i < int(r.Delta); i++ {
		containerDockerID, err := c.dockerClient.CreateContainer(fmt.Sprintf("%s-%d", r.Id, i))
		if err != nil {
			return nil, err
		}
		log.Printf("Created container '%s' for node group '%s'", containerDockerID, r.Id)
		addDockerContainerAsNode(c, containerDockerID, r.Id)
	}

	log.Printf("Finished creating nodes. There are now '%d' nodes running on this machine", len(c.instances))
	return &protos.NodeGroupIncreaseSizeResponse{}, nil
}

func (c *CloudProviderServer) NodeGroupDeleteNodes(context context.Context, r *protos.NodeGroupDeleteNodesRequest) (*protos.NodeGroupDeleteNodesResponse, error) {
	log.Printf("Deleting nodes '%d' from node group '%s'", len(r.Nodes), r.Id)

	for _, node := range r.Nodes {
		log.Printf("Deleting node with name '%s'", node.Name)
		err := c.dockerClient.DeleteContainer(node.Name)
		if err != nil {
			log.Printf("Failed to delete container '%s': %v", node.Name, err)
		}
		log.Printf("Deleted container '%s' for node group '%s'", node.Name, r.Id)
		removeNode(c, findNodeByContainerID(*c, node.Name))
	}

	log.Printf("Finished deleting nodes. There are now '%d' nodes running on this machine", len(c.instances))
	return &protos.NodeGroupDeleteNodesResponse{}, nil
}

func (CloudProviderServer) NodeGroupDecreaseTargetSize(context.Context, *protos.NodeGroupDecreaseTargetSizeRequest) (*protos.NodeGroupDecreaseTargetSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupDecreaseTargetSize not implemented")
}

func (c CloudProviderServer) NodeGroupNodes(context context.Context, r *protos.NodeGroupNodesRequest) (*protos.NodeGroupNodesResponse, error) {
	// ## Testing ##
	// if r.Id != nodeGroups[0].Id {
	// 	return nil, status.Errorf(codes.NotFound, "node group not found")
	// }
	localContainerIDs := c.dockerClient.ListContainers()
	nodesInNodeGroup := []*protos.Instance{}
	for _, containerID := range localContainerIDs {
		instance := findNodeByContainerID(c, containerID)
		if instance != nil {
			nodesInNodeGroup = append(nodesInNodeGroup, instance)
		}
	}

	// ## Testing ##
	// return &protos.NodeGroupNodesResponse{
	// 	Instances: c.instances,
	// }, nil
	return &protos.NodeGroupNodesResponse{
		Instances: nodesInNodeGroup,
	}, nil
}
func (CloudProviderServer) NodeGroupTemplateNodeInfo(context.Context, *protos.NodeGroupTemplateNodeInfoRequest) (*protos.NodeGroupTemplateNodeInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupTemplateNodeInfo not implemented")
}
func (CloudProviderServer) NodeGroupGetOptions(context.Context, *protos.NodeGroupAutoscalingOptionsRequest) (*protos.NodeGroupAutoscalingOptionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupGetOptions not implemented")
}

func addDockerContainerAsNode(c *CloudProviderServer, containerDockerID string, nodeGroupID string) {
	c.instances = append(c.instances, &protos.Instance{
		Id:     containerDockerID,
		Status: &protos.InstanceStatus{InstanceState: protos.InstanceStatus_instanceRunning},
	})
	log.Printf("Added container '%s' as node to node group '%s'. There are now a total of '%d' nodes.", containerDockerID, nodeGroupID, len(c.instances))
}

func removeNode(c *CloudProviderServer, removedNode *protos.Instance) {
	// Simply remove the last node for now
	if len(c.instances) == 0 {
		log.Printf("No nodes to remove.")
		return
	}
	// Remove the node with the specificied id
	for i, instance := range c.instances {
		if instance.Id == removedNode.Id {
			// Remove the instance from the slice
			c.instances = append(c.instances[:i], c.instances[i+1:]...)
			break
		}
	}
	log.Printf("Removed node with ID '%s'. There are now a total of '%d' nodes.", removedNode.Id, len(c.instances))
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

func findNodeByContainerID(c CloudProviderServer, containerID string) *protos.Instance {
	for _, i := range c.instances {
		if i.Id == containerID {
			return i
		}
	}
	return nil
}
