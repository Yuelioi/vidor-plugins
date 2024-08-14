package main

import (
	"context"
	"log"

	pb "proto"

	"google.golang.org/grpc"
)

const socketPath2 = "/tmp/server1.sock"

func main2() {
	conn, err := grpc.Dial("unix://"+socketPath, grpc.WithInsecure())
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
