package main

import (
	"mbj-autoscaler/cluster-autoscaler/cloudprovider/externalgrpc/protos"
	"mbj-autoscaler/dockerCloudProviderServer"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

func main() {
	// TODO: Verify that Docker is running and accessible

	// Step 1: Create a gRPC client using mbjgrpcserver
	grpcServer := grpc.NewServer()

	// Step 2: Create a Docker gRPC server and register it with the gRPC client
	dockerCloudProviderServer := dockerCloudProviderServer.NewServer()
	protos.RegisterCloudProviderServer(grpcServer, dockerCloudProviderServer)

	// Step 3: Start listening
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		klog.Fatalf("Failed to listen on %s: %v", "127.0.0.1:50051", err)
	}

	// Step 4: Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		klog.Info("Received shutdown signal, stopping server...")
		grpcServer.GracefulStop()
	}()

	// Step 5: Start serving
	klog.Infof("Docker Cloud Provider gRPC server listening on %s", "127.0.0.1:50051")
	if err := grpcServer.Serve(lis); err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
}
