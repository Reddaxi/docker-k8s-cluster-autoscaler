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
	instances := []*protos.Instance{}
	for _, ng := range resp.NodeGroups {
		log.Printf("  - ID: %s, Min: %d, Max: %d, Debug: %s",
			ng.Id, ng.MinSize, ng.MaxSize, ng.Debug)

		// Test NodeGroupNodes call
		log.Printf("Testing NodeGroupNodes call for node group '%s'...", ng.Id)
		nodesResp, err := client.NodeGroupNodes(ctx, &protos.NodeGroupNodesRequest{
			Id: ng.Id,
		})
		instances = append(instances, nodesResp.Instances...)
		if err != nil {
			log.Fatalf("NodeGroupNodes failed: %v", err)
		}
		log.Printf("  Success! Got %d nodes:", len(instances))
		for _, inst := range instances {
			log.Printf("    - Instance ID: %s, Status: %s", inst.Id, inst.Status)
		}
	}

	// Test NodeGroupTargetSize call
	log.Println("Testing NodeGroupTargetSize call...")
	sizeResp, err := client.NodeGroupTargetSize(ctx, &protos.NodeGroupTargetSizeRequest{
		Id: resp.NodeGroups[0].Id,
	})
	if err != nil {
		log.Fatalf("NodeGroupTargetSize failed: %v", err)
	}
	log.Printf("Target size for %s: %d", resp.NodeGroups[0].Id, sizeResp.TargetSize)

	// Test NodeGroupIncreaseSize call
	log.Println("Testing NodeGroupIncreaseSize call...")
	_, err = client.NodeGroupIncreaseSize(ctx, &protos.NodeGroupIncreaseSizeRequest{
		Id:    resp.NodeGroups[0].Id,
		Delta: 1,
	})
	if err != nil {
		log.Fatalf("NodeGroupIncreaseSize failed: %v", err)
	}
	log.Println("Successfully called NodeGroupIncreaseSize!")

	// Test NodeGroupDeleteNodes call
	nodesResp, err := client.NodeGroupNodes(ctx, &protos.NodeGroupNodesRequest{
		Id: resp.NodeGroups[0].Id,
	})
	if err != nil {
		log.Fatalf("NodeGroupNodes failed: %v", err)
	}
	instances = append(instances, nodesResp.Instances...)
	lastInstance := instances[len(instances)-1]
	log.Println("Testing NodeGroupDeleteNodes call...")
	_, err = client.NodeGroupDeleteNodes(ctx, &protos.NodeGroupDeleteNodesRequest{
		Id: resp.NodeGroups[0].Id,
		Nodes: []*protos.ExternalGrpcNode{
			{Name: lastInstance.Id},
		},
	})
	if err != nil {
		log.Printf("NodeGroupDeleteNodes expectedly failed (not implemented): %v", err)
	} else {
		log.Fatalf("NodeGroupDeleteNodes unexpectedly succeeded")
	}

	log.Println("All tests passed! 🎉")
}
