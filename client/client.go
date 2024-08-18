package main

import (
	"log"

	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Port    string
	Service pb.DownloadServiceClient
}

func NewClient() *Client {

	c := &Client{}

	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Server1: %v", err)
	}

	c.Service = pb.NewDownloadServiceClient(conn)
	defer conn.Close()
	return c
}
