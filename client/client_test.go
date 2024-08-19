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

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func TestInit(t *testing.T) {
	c, err := NewClient()
	assert.NoError(t, err, err)
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")

	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err = c.Service.Init(ctx, nil)

	if err != nil {
		return
	}

}
func TestHealth(t *testing.T) {
	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()
	client := healthpb.NewHealthClient(conn)
	res, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	assert.NoError(t, err, err)

	fmt.Printf("Health check status: %s\n", res.Status)

}

func TestShow(t *testing.T) {
	c, err := NewClient()
	assert.NoError(t, err, err)

	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")

	start := time.Now()
	t.Log("开始测试: \n")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err = c.Service.Init(ctx, nil)

	assert.NoError(t, err, err)

	for idx, url := range []string{
		// testUrl,
		testPagesUrl,
		// testSeasonsUrl,
	} {
		response, err := c.Service.Show(ctx, &pb.ShowRequest{
			Url: url,
		})
		assert.NoError(t, err, err)

		t.Logf("Server response:%d %s\n\n\n", idx, response)
	}
	t.Logf("运行时间: %v\n", time.Since(start))
}

func TestParse(t *testing.T) {
	c, err := NewClient()
	assert.NoError(t, err, err)
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err = c.Service.Init(ctx, nil)

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
	c, err := NewClient()
	assert.NoError(t, err, err)
	ctx := context.Background()
	LoadEnv()
	value := os.Getenv("SESSDATA")
	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value)
	_, err = c.Service.Init(ctx, nil)

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