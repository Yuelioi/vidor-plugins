package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	pb "proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	Port    string
	Service pb.DownloadServiceClient
}

func NewClient() (*Client, error) {

	c := &Client{}

	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c.Service = pb.NewDownloadServiceClient(conn)

	return c, nil
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

	// availablePort, err := getAvailablePort()
	// exec.Command("server.exe", "--port", strconv.Itoa(availablePort))

	c, err := NewClient()

	if err != nil {
		return
	}
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value, "host", "test")
	_, err = c.Service.Init(ctx, nil)

	req := &pb.ParseRequest{
		Id: "https://example.com/video.mp4",
		Tasks: []*pb.Task{
			{
				Url:       testUrl,
				SessionId: "973268535",
			},
		},
	}

	resp, err := c.Service.ParseEpisodes(context.Background(), req)

	fmt.Print("解析后内容", resp.Tasks[0].Segments[0].Formats[0])

	reqDown := &pb.DownloadRequest{
		Tasks: resp.Tasks,
	}

	stream, err := c.Service.Download(context.Background(), reqDown)

	go func() {
		time.Sleep(time.Second * 10)
		_, err := c.Service.StopDownload(ctx, &pb.StopDownloadRequest{
			Id: resp.Tasks[0].Id,
		})

		fmt.Printf("err: %v\n", err)

	}()

	// 从流中接收下载进度
	for {
		progress, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error receiving progress: %v", err)
		}
		fmt.Printf("Download progress: %s - %s\n", progress.Id, progress.Speed)
	}

	fmt.Println("Download finished.")
}
