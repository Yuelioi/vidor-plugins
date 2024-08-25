package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	pb "proto"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedDownloadServiceServer
	client *Client
}

type healthServer struct {
	healthpb.UnimplementedHealthServer
}

func (s *healthServer) Check(ctx context.Context, in *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{
		Status: healthpb.HealthCheckResponse_SERVING,
	}, nil
}

// 初始化
func (s *server) Init(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to connect\n")
	return &empty.Empty{}, nil
}

// 关闭
func (s *server) Shutdown(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {

	// 关闭程序
	os.Exit(0)

	return &empty.Empty{}, nil
}

func (s *server) GetInfo(ctx context.Context, sr *pb.InfoRequest) (*pb.InfoResponse, error) {
	return s.client.GetInfo(sr.Url)
}

func (s *server) Parse(ctx context.Context, pr *pb.ParseRequest) (*pb.ParseResponse, error) {
	return s.client.Parse(pr)
}

func (s *server) Download(dr *pb.DownloadRequest, stream pb.DownloadService_DownloadServer) error {

	for _, task := range dr.Tasks {
		s.client.Download(task, stream)
	}

	return nil

}

func (s *server) StopDownload(ctx context.Context, sr *pb.StopDownloadRequest) (*pb.StopDownloadResponse, error) {

	id := sr.Id
	if stopChan, ok := s.client.stopChannels.Load(id); ok {
		close(stopChan.(chan struct{}))
		s.client.stopChannels.Delete(id)
		return nil, nil
	}

	return &pb.StopDownloadResponse{
		Id: sr.Id,
	}, fmt.Errorf("task with ID %s not found", id)

}

func main() {

	port := flag.Int("port", 9001, "Port number to listen on")
	flag.Parse()

	os.WriteFile("log.txt", []byte(strconv.Itoa(*port)), 0644)

	lis, err := net.Listen("tcp", "localhost:"+strconv.Itoa(*port))
	if err != nil {
		log.Fatalf("Failed to listen on TCP: %v", err)
	}

	fmt.Printf("Port: %d\n", *port)
	fmt.Printf("PID: %d\n", os.Getpid())

	actualPort := lis.Addr().(*net.TCPAddr).Port

	fmt.Printf("Port: %d\n", actualPort)

	grpcServer := grpc.NewServer()

	s := &server{
		client: NewClient(),
	}

	healthpb.RegisterHealthServer(grpcServer, &healthServer{})
	pb.RegisterDownloadServiceServer(grpcServer, s)

	log.Printf("Server1 listening on %d", actualPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
