package main

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func (s *server) LoadConfig(ctx context.Context) {
	s.LoadSessdata(ctx)
}

func (s *server) LoadSessdata(ctx context.Context) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		result := md.Get("plugin.sessdata")
		if len(result) > 0 {
			s.client.BpiService.Client.SESSDATA = result[0]
		}
	}
}
