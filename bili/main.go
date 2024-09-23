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
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedDownloadServiceServer
	client     *Client
	grpcServer *grpc.Server
}

// 初始化
func (s *server) Init(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to connect\n")
	s.LoadSessdata(ctx)
	return &empty.Empty{}, nil
}

// 更新数据
func (s *server) Update(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to update\n")
	s.LoadSessdata(ctx)
	return &empty.Empty{}, nil
}

// 关闭
func (s *server) Shutdown(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	// 关闭程序
	go func() {
		s.grpcServer.GracefulStop()
	}()

	// 返回成功响应
	return &empty.Empty{}, nil
}

// 功能

func (s *server) GetInfo(ctx context.Context, sr *pb.InfoRequest) (*pb.InfoResponse, error) {
	return s.client.GetInfo(sr.Url)
}

func (s *server) Parse(ctx context.Context, pr *pb.TasksRequest) (*pb.TasksResponse, error) {
	return s.client.Parse(pr)
}

func (s *server) Download(dr *pb.TasksRequest, stream pb.DownloadService_DownloadServer) error {
	for _, task := range dr.Tasks {
		s.client.Download(task, stream)
	}
	return nil
}

// TODO
func (s *server) Resume(ctx context.Context, pr *pb.TaskRequest) (*pb.TaskResponse, error) {
	return nil, nil
}

func (s *server) Stop(ctx context.Context, sr *pb.TaskRequest) (*pb.TaskResponse, error) {
	id := sr.Id
	if stopChan, ok := s.client.stopChannels.Load(id); ok {
		close(stopChan.(chan struct{}))
		s.client.stopChannels.Delete(id)
		return nil, nil
	}

	return &pb.TaskResponse{
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

	grpcServer := grpc.NewServer()

	s := &server{
		client:     NewClient(),
		grpcServer: grpcServer,
	}

	pb.RegisterDownloadServiceServer(grpcServer, s)

	// 创建健康检查服务
	healthServer := health.NewServer()

	// 设置服务状态为 SERVING
	healthServer.SetServingStatus("health_check", grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册健康检查服务到 gRPC 服务器
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("Server1 listening on %d", actualPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
