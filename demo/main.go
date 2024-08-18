package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

func getPluginPort(pluginExe string) (string, error) {
	cmd := exec.Command(pluginExe)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start plugin: %v", err)
	}

	// 创建一个扫描器从管道中逐行读取输出
	scanner := bufio.NewScanner(stdout)
	var port string

	for scanner.Scan() {
		line := scanner.Text()
		// 检查输出中是否包含端口信息
		if strings.HasPrefix(line, "Port:") {
			port = strings.TrimSpace(strings.TrimPrefix(line, "Port:"))
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read stdout: %v", err)
	}

	if port == "" {
		return "", fmt.Errorf("port not found in plugin output")
	}

	return port, nil
}

func main() {
	port, _ := getPluginPort(`F:\go_projects\vidor-plugins\server\server.exe`)

	fmt.Printf("port: %v\n", port)
}
