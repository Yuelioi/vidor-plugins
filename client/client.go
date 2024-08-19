package main

import (
	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
