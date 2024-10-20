package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedDownloadServiceServer
	tq         *JobManager
	client     *Client
	grpcServer *grpc.Server
	config     *Config
}

// 初始化
func (s *server) Init(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to connect\n")
	return &empty.Empty{}, nil
}

// 更新数据
func (s *server) Update(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to update\n")
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		sessdata := md.Get("plugin.sessdata")
		if len(sessdata) > 0 {
			s.client.BpiService.Client.SESSDATA = sessdata[0]
			fmt.Printf("sessdata: %v\n", sessdata[0])
		}

		ffmpeg := md.Get("system.ffmpeg")
		if len(ffmpeg) > 0 {
			fmt.Printf("ffmpeg: %v\n", ffmpeg[0])
			s.config.ffmpeg = ffmpeg[0]
		}

		tmps := md.Get("system.tmpdirs")
		if len(tmps) > 0 {
			fmt.Printf("tmps: %v\n", tmps[0])
			s.config.tmpDir = tmps[0]
		}
		return &empty.Empty{}, nil
	}
	return &empty.Empty{}, errors.New("主机未提供相应参数")
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

func (s *server) GetInfo(ctx context.Context, sr *pb.InfoRequest) (*pb.InfoResponse, error) {
	ir, err := s.client.GetInfo(sr.Url)
	if err != nil {
		return nil, err
	}

	suffix := filepath.Ext(ir.Cover)
	tmpCoverPath := filepath.Join(s.config.tmpDir, "cover", timestamp()+suffix)
	err = downloadCover(ir.Cover, tmpCoverPath)
	if err != nil {
		return nil, err
	}

	ir.Cover = tmpCoverPath
	return ir, nil
}

func (s *server) Parse(ctx context.Context, pr *pb.TasksRequest) (*pb.TasksResponse, error) {
	return s.client.Parse(pr)
}

func (s *server) Download(tr *pb.TaskRequest, stream pb.DownloadService_DownloadServer) error {
	return s.client.Download(tr.Task, s.config, stream, s.tq)
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

	lis, err := net.Listen("tcp", "localhost:"+strconv.Itoa(*port))
	if err != nil {
		log.Fatalf("Failed to listen on TCP: %v", err)
	}

	actualPort := lis.Addr().(*net.TCPAddr).Port

	grpcServer := grpc.NewServer()

	s := &server{
		tq:         NewJobManager(),
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

	fmt.Printf("Server Run At Port: %d  PID:%d\n", actualPort, os.Getpid())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
