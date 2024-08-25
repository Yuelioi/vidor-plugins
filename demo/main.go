package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	pb "proto"
	"strconv"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Plugin struct {
	service pb.DownloadServiceClient
}

func getAvailablePort() (int, error) {
	// 监听 "localhost:0" 让系统分配一个可用端口
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find an available port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}

func (p *Plugin) start() {
	port, err := getAvailablePort()
	if err != nil {
		return
	}
	// port := 9001

	// 获取命令
	cmd := exec.Command(`F:\go_projects\vidor-plugins\demo\bilibili.exe`, "--port", strconv.Itoa(port))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// 启动进程
	err = cmd.Start()
	if err != nil {
		return
	}

	// 获取 exe 运行的 PID
	pid := cmd.Process.Pid
	fmt.Printf("p: %v\n", pid)

	conn, err := grpc.NewClient("localhost:"+strconv.Itoa(port), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return
	}

	server := pb.NewDownloadServiceClient(conn)
	p.service = server
}

func main() {
	p := &Plugin{}
	p.start()

	// conn, err := grpc.NewClient("localhost:"+strconv.Itoa(3446), grpc.WithTransportCredentials(insecure.NewCredentials()))
	// server := pb.NewDownloadServiceClient(conn)

	// Use the server, ensuring it's not nil
	_, err := p.service.Init(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	response, err := p.service.GetInfo(context.Background(), &pb.InfoRequest{
		Url: "https://www.bilibili.com/video/BV1bA411R7BN",
	})

	fmt.Printf("response: %v\n", response)

	p.service.Shutdown(context.Background(), nil)

	println("加载成功")

	// Keep the application running or proceed with further logic
	// to ensure that the server is actively used.
}
