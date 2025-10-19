package main

import (
	"context"
	"log"
	"time"

	"mbj-autoscaler/cluster-autoscaler/cloudprovider/externalgrpc/protos"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create client
	client := protos.NewCloudProviderClient(conn)

	// Test NodeGroups call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Testing NodeGroups call...")
	resp, err := client.NodeGroups(ctx, &protos.NodeGroupsRequest{})
	if err != nil {
		log.Fatalf("NodeGroups failed: %v", err)
	}

	log.Printf("Success! Got %d node groups:", len(resp.NodeGroups))
	for _, ng := range resp.NodeGroups {
		log.Printf("  - ID: %s, Min: %d, Max: %d, Debug: %s",
			ng.Id, ng.MinSize, ng.MaxSize, ng.Debug)
	}

	// Test NodeGroupTargetSize call
	log.Println("Testing NodeGroupTargetSize call...")
	sizeResp, err := client.NodeGroupTargetSize(ctx, &protos.NodeGroupTargetSizeRequest{
		Id: "test-group",
	})
	if err != nil {
		log.Fatalf("NodeGroupTargetSize failed: %v", err)
	}
	log.Printf("Target size for test-group: %d", sizeResp.TargetSize)

	// Test NodeGroupIncreaseSize call
	log.Println("Testing NodeGroupIncreaseSize call...")
	_, err = client.NodeGroupIncreaseSize(ctx, &protos.NodeGroupIncreaseSizeRequest{
		Id:    "test-group",
		Delta: 2,
	})
	if err != nil {
		log.Fatalf("NodeGroupIncreaseSize failed: %v", err)
	}
	log.Println("Successfully called NodeGroupIncreaseSize!")

	log.Println("All tests passed! 🎉")
}
