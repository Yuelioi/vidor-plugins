package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	pb "proto"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func TestInit(t *testing.T) {
	c := NewClient()
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")

	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err := c.Service.Init(ctx, nil)

	if err != nil {
		return
	}

}

func TestShow(t *testing.T) {
	c := NewClient()
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")

	start := time.Now()

	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err := c.Service.Init(ctx, nil)

	if err != nil {
		return
	}

	for idx, url := range []string{
		// testUrl,
		testPagesUrl,
		// testSeasonsUrl,
	} {
		response, err := c.Service.Show(ctx, &pb.ShowRequest{
			Url: url,
		})
		if err != nil {
			log.Fatalf("Could not greet: %v", err)
		}

		log.Printf("Server response:%d %s\n\n\n", idx, response)
	}
	fmt.Printf("运行时间: %v\n", time.Since(start))
}

func TestParse(t *testing.T) {
	c := NewClient()
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err := c.Service.Init(ctx, nil)

	if err != nil {
		return
	}

	start := time.Now()

	req := &pb.ParseRequest{
		Id: "https://example.com/video.mp4",
		StreamInfos: []*pb.StreamInfo{
			{
				Url:       "https://www.bilibili.com/video/av1956391733/",
				SessionId: "1641484630",
			},
		},
	}

	resp, err := c.Service.Parse(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to start download: %v", err)
	}
	log.Printf("Server response:%s\n\n\n", resp)

	fmt.Printf("运行时间: %v\n", time.Since(start))

}
func TestDownload(t *testing.T) {
	c := NewClient()
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err := c.Service.Init(ctx, nil)

	if err != nil {
		return
	}

	req := &pb.DownloadRequest{
		Id: "https://example.com/video.mp4",
	}

	stream, err := c.Service.Download(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to start download: %v", err)
	}

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
