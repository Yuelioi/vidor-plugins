package main

import (
	"context"
	"fmt"
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

	ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value, "host", "test")
	// ctx = metadata.AppendToOutgoingContext(ctx, "plugin.sessdata", value, "host", "test")
	_, err = c.Service.Init(ctx, nil)

	assert.NoError(t, err)

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
		testUrl,
		// testPagesUrl,
		// testSeasonsUrl,
	} {
		response, err := c.Service.GetInfo(ctx, &pb.InfoRequest{
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

	assert.NoError(t, err)

	start := time.Now()

	req := &pb.ParseRequest{
		Id: "https://example.com/video.mp4",
		Tasks: []*pb.Task{
			{
				Url:       testUrl,
				SessionId: "973268535",
			},
		},
	}

	resp, err := c.Service.Parse(context.Background(), req)
	assert.NoError(t, err)

	log.Printf("Server response:%s\n\n\n", resp)

	fmt.Printf("运行时间: %v\n", time.Since(start))

}
