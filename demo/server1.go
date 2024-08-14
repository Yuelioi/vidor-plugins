package main

import (
	"context"
	"log"
	"net"
	"os"

	pb "proto"

	"google.golang.org/grpc"
)

const socketPath = "/tmp/server1.sock"

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	// 删除旧的 Unix Socket 文件
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatalf("Failed to remove Unix socket file: %v", err)
	}

	// 创建 Unix Socket 监听器
	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on Unix socket: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})

	log.Printf("Server1 listening on %s", socketPath)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
