package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
)

func (s *server) LoadConfig(ctx context.Context) error {
	return s.LoadSessdata(ctx)
}

func (s *server) LoadSessdata(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		sessdata := md.Get("plugin.sessdata")
		if len(sessdata) > 0 {
			s.client.BpiService.Client.SESSDATA = sessdata[0]
		}

		host := md.Get("host")
		if len(host) > 0 {
			fmt.Printf("host: %v\n", host[0])
		}

	}

	return fmt.Errorf("验证失败")
}
