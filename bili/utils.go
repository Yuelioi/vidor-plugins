package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// 提取aid和bvid
func extractAidBvid(link string) (aid int, bvid string) {
	aidRegex := regexp.MustCompile(`av(\d+)`)
	bvidRegex := regexp.MustCompile(`BV\w+`)

	aidMatches := aidRegex.FindStringSubmatch(link)
	if len(aidMatches) > 1 {
		aid, _ = strconv.Atoi(aidMatches[1])
	}

	bvidMatches := bvidRegex.FindStringSubmatch(link)
	if len(bvidMatches) > 0 {
		bvid = bvidMatches[0]
	}
	return
}

// 格式化文件名
func sanitizeFileName(input string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	sanitized := re.ReplaceAllString(input, "_")

	sanitized = strings.TrimSpace(sanitized)
	sanitized = strings.Trim(sanitized, ".")

	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}

// 获取时间戳
func timestamp() string {
	now := time.Now()
	layout := "20060102150405.000"
	formattedTime := now.Format(layout)
	return strings.Replace(formattedTime, ".", "-", 1)
}

// 下载封面
func downloadCover(url, filePath string) error {
	client := resty.New()

	response, err := client.R().
		Get(url)
	if err != nil {
		return err
	}

	if response.StatusCode() != 200 {
		return fmt.Errorf("failed to fetch the image: %s", response.Status())
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	return os.WriteFile(filePath, response.Body(), os.ModePerm)
}
