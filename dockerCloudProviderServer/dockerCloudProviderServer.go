package dockerCloudProviderServer

import (
	"context"
	"mbj-autoscaler/cluster-autoscaler/cloudprovider/externalgrpc/protos"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewServer() CloudProviderServer {
	// Implementation goes here
	server := &CloudProviderServer{}
	return *server
}

// CloudProviderServer implements the gRPC CloudProvider service
type CloudProviderServer struct {
	protos.UnimplementedCloudProviderServer
}

func (CloudProviderServer) NodeGroups(context.Context, *protos.NodeGroupsRequest) (*protos.NodeGroupsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroups not implementeda")
}
func (CloudProviderServer) NodeGroupForNode(context.Context, *protos.NodeGroupForNodeRequest) (*protos.NodeGroupForNodeResponse, error) {
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
func (CloudProviderServer) NodeGroupTargetSize(context.Context, *protos.NodeGroupTargetSizeRequest) (*protos.NodeGroupTargetSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupTargetSize not implemented")
}
func (CloudProviderServer) NodeGroupIncreaseSize(context.Context, *protos.NodeGroupIncreaseSizeRequest) (*protos.NodeGroupIncreaseSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupIncreaseSize not implemented")
}
func (CloudProviderServer) NodeGroupDeleteNodes(context.Context, *protos.NodeGroupDeleteNodesRequest) (*protos.NodeGroupDeleteNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupDeleteNodes not implemented")
}
func (CloudProviderServer) NodeGroupDecreaseTargetSize(context.Context, *protos.NodeGroupDecreaseTargetSizeRequest) (*protos.NodeGroupDecreaseTargetSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupDecreaseTargetSize not implemented")
}
func (CloudProviderServer) NodeGroupNodes(context.Context, *protos.NodeGroupNodesRequest) (*protos.NodeGroupNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupNodes not implemented")
}
func (CloudProviderServer) NodeGroupTemplateNodeInfo(context.Context, *protos.NodeGroupTemplateNodeInfoRequest) (*protos.NodeGroupTemplateNodeInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupTemplateNodeInfo not implemented")
}
func (CloudProviderServer) NodeGroupGetOptions(context.Context, *protos.NodeGroupAutoscalingOptionsRequest) (*protos.NodeGroupAutoscalingOptionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupGetOptions not implemented")
}
