package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	pb "proto"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedDownloadServiceServer
	client    *Client
	taskQueue *TaskQueue
	once      sync.Once
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
	s.once.Do(func() {
		// 创建 client
		s.client = NewClient()

		// 创建任务队列
		s.taskQueue = NewTaskQueue()

		// 初始化插件配置
		s.LoadConfig(ctx)
	})
	return &empty.Empty{}, nil
}

func (s *server) Show(ctx context.Context, sr *pb.ShowRequest) (*pb.ShowResponse, error) {
	return s.client.Info(sr.Url)
}

func (s *server) Parse(ctx context.Context, pr *pb.ParseRequest) (*pb.ParseResponse, error) {
	return s.client.Parse(pr)
}

func (s *server) Download(dr *pb.DownloadRequest, stream pb.DownloadService_DownloadServer) error {
	// 添加任务队列
	s.taskQueue.AddTask(NewTask(dr.Id))

	// 模拟下载总大小（单位：MB）Download
	totalSize := 100
	chunkSize := 10

	for i := 0; i <= totalSize; i += chunkSize {
		// 模拟每次下载一个块
		time.Sleep(1 * time.Second)

		// 计算进度百分比
		progress := float32(i) / float32(totalSize) * 100

		// 创建 DownloadProgress 消息并发送给客户端
		progressMsg := &pb.DownloadProgress{
			Id:         "1",
			TotalBytes: 100,
		}

		// 将进度发送到客户端
		if err := stream.Send(progressMsg); err != nil {
			return fmt.Errorf("error sending progress: %v", err)
		}

		// 模拟下载完成
		if i == totalSize {
			progressMsg = &pb.DownloadProgress{
				Id:         "1",
				TotalBytes: 100,
				Speed:      fmt.Sprint(progress),
			}
			if err := stream.Send(progressMsg); err != nil {
				return fmt.Errorf("error sending final progress: %v", err)
			}
			break
		}
	}

	return nil
}

func (s *server) StopDownload(context.Context, *pb.StopDownloadRequest) (*pb.StopDownloadResponse, error) {
	return nil, nil
}

func getAvailablePort() (int, error) {
	// 监听 "localhost:0" 让系统分配一个可用端口
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find an available port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}

func main() {

	port := flag.Int("port", 9001, "Port number to listen on")
	flag.Parse()

	lis, err := net.Listen("tcp", "localhost:9001")
	if err != nil {
		log.Fatalf("Failed to listen on TCP: %v", err)
	}

	fmt.Printf("Port: %d\n", *port)
	fmt.Printf("PID: %d\n", os.Getpid())

	actualPort := lis.Addr().(*net.TCPAddr).Port

	fmt.Printf("Port: %d\n", actualPort)

	grpcServer := grpc.NewServer()

	healthpb.RegisterHealthServer(grpcServer, &healthServer{})
	pb.RegisterDownloadServiceServer(grpcServer, &server{})

	log.Printf("Server1 listening on %d", actualPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
