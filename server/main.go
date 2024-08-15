package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	pb "proto"

	"github.com/Yuelioi/bilibili/pkg/bpi"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedDownloadServiceServer
	service   *bpi.BpiService
	taskQueue *TaskQueue
	once      sync.Once
}

func LoadEnv() {

	_, filename, _, _ := runtime.Caller(0)
	env := filepath.Join(filepath.Dir(filename), "..", ".env")

	// Attempt to load the .env file
	err := godotenv.Load(env)
	if err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
	}
}

func (s *server) ShowInfo(ctx context.Context, sr *pb.ShowInfoRequest) (*pb.ShowInfoResponse, error) {
	s.once.Do(func() {
		// 创建 client
		s.service = bpi.New()

		// 创建任务队列
		s.taskQueue = NewTaskQueue()
	})

	// 添加任务队列
	s.taskQueue.AddTask(NewTask(sr.Url))

	// 注册 cookie 可以走metadata?
	LoadEnv()
	value := os.Getenv("SESSDATA")
	s.service.Client.SESSDATA = value
	video, _ := s.service.Video().Detail(0, "BV1nr421M747")
	fmt.Printf("video.Data: %v\n", video.Data)
	return &pb.ShowInfoResponse{}, nil
}

func (s *server) Download(dr *pb.DownloadRequest, stream pb.DownloadService_DownloadServer) error {
	fmt.Printf("Starting download for URL: %v\n", dr.Id)

	// 模拟下载总大小（单位：MB）
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

func main() {

	lis, err := net.Listen("tcp", "localhost:9001")
	if err != nil {
		log.Fatalf("Failed to listen on Unix socket: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDownloadServiceServer(s, &server{})

	log.Printf("Server1 listening on %d", 9001)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
