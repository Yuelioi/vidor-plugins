package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {

	testUrl := "https://www.bilibili.com/video/BV1nr421M747/"

	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Server1: %v", err)
	}
	defer conn.Close()

	go func() {
		client := pb.NewDownloadServiceClient(conn)
		ctx := context.Background()
		ctx = metadata.AppendToOutgoingContext(ctx, "id", "123")

		response, err := client.ShowInfo(ctx, &pb.ShowInfoRequest{
			Url: testUrl,
		})
		if err != nil {
			log.Fatalf("Could not greet: %v", err)
		}

		log.Printf("Server1 response: %s", response)
	}()

	go func() {
		client := pb.NewDownloadServiceClient(conn)

		ctx := context.Background()
		ctx = metadata.AppendToOutgoingContext(ctx, "id", "456")
		response2, err := client.ShowInfo(ctx, &pb.ShowInfoRequest{
			Url: testUrl,
		})
		if err != nil {
			log.Fatalf("Could not greet: %v", err)
		}
		log.Printf("Server1 response: %s", response2)
	}()

	go func() {

		req := &pb.DownloadRequest{
			Id: "https://example.com/video.mp4",
		}
		client := pb.NewDownloadServiceClient(conn)

		stream, err := client.Download(context.Background(), req)
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
	}()

	time.Sleep(time.Second * 20)
}
