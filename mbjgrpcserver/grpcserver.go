package mbjgrpcserver

import "google.golang.org/grpc"

func NewServer() ServiceRegistrar {
	// Implementation goes here
	server := &ServiceRegistrar{}
	return *server
}

// ServiceRegistrar implements the gRPC CloudProvider service
type ServiceRegistrar struct {
	grpc.ServiceRegistrar
}

// Should somehow register the grpc server as a service? Maybe a literal linux service? I don't know.
func (r ServiceRegistrar) RegisterService(desc *grpc.ServiceDesc, impl any) {
	r.ServiceRegistrar.RegisterService(desc, impl)
}
