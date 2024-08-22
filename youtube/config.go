package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
)

func (s *server) LoadConfig(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		proxy := md.Get("vidor.proxy")
		if len(proxy) > 0 {
			s.client.proxyUrl = proxy[0]
		}
		useProxy := md.Get("vidor.useproxy")
		if len(useProxy) > 0 {
			s.client.useProxy = useProxy[0] == "true"
		}
	}
	return fmt.Errorf("验证失败")
}
