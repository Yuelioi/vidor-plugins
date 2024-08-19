package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	pb "proto"

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
func main() {

	c, err := NewClient()

	if err != nil {
		return
	}
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err = c.Service.Init(ctx, nil)

	req := &pb.ParseRequest{
		Id: "https://example.com/video.mp4",
		StreamInfos: []*pb.StreamInfo{
			{
				Url:       testUrl,
				SessionId: "973268535",
			},
		},
	}

	resp, err := c.Service.Parse(context.Background(), req)

	fmt.Print("解析后内容", resp.StreamInfos[0].Streams[0].Formats[0])

	reqDown := &pb.DownloadRequest{
		StreamInfos: resp.StreamInfos,
	}

	stream, err := c.Service.Download(context.Background(), reqDown)

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
