package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
)

func (s *server) LoadConfig(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		sessdata := md.Get("plugin.sessdata")
		if len(sessdata) > 0 {
			s.client.BpiService.Client.SESSDATA = sessdata[0]
			fmt.Printf("sessdata: %v\n", sessdata[0])
		}

		ffmpeg := md.Get("system.ffmpeg")
		if len(ffmpeg) > 0 {
			fmt.Printf("ffmpeg: %v\n", ffmpeg[0])
			s.ffmpeg = ffmpeg[0]
		}

		tmps := md.Get("system.tmpdirs")
		if len(tmps) > 0 {
			fmt.Printf("tmps: %v\n", tmps[0])
			s.tmpDir = tmps[0]
		}

	}

	return fmt.Errorf("验证失败")
}
