package main

import (
	"context"
	"log"
	"net"

	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	var data map[string]string

	metadata.NewIncomingContext(ctx, metadata.New(data))
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {

	lis, err := net.Listen("tcp", "localhost:9001")
	if err != nil {
		log.Fatalf("Failed to listen on Unix socket: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})

	log.Printf("Server1 listening on %d", 9001)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
