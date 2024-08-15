package main

import (
	"context"
	"log"

	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Server1: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)

	response, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "World"})
	if err != nil {
		log.Fatalf("Could not greet: %v", err)
	}

	log.Printf("Server1 response: %s", response.GetMessage())
}
