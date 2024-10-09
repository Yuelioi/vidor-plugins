package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
)

// command.go
type Command interface {
	Execute(url string, chunkStart, chunkEnd int64, out *os.File, totalBytesRead *atomic.Int64, stopChan chan struct{}) error
}

type DownloadChunkCommand struct {
	client *Client
}

func NewDownloadChunkCommand(client *Client) *DownloadChunkCommand {
	return &DownloadChunkCommand{
		client: client,
	}
}

func (d *DownloadChunkCommand) Execute(url string, chunkStart, chunkEnd int64, out *os.File, totalBytesRead *atomic.Int64, stopChan chan struct{}) error {
	req := d.client.BpiService.Client.HTTPClient.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd)).
		SetHeader("Referer", "https://www.bilibili.com/").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: d.client.BpiService.Client.SESSDATA,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(url)
	if err != nil {
		return err
	}
	defer resp.RawBody().Close()

	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-stopChan:
			fmt.Println("Context canceled")
			return fmt.Errorf("download stopped for chunk %d-%d", chunkStart, chunkEnd)
		default:
			n, err := io.ReadFull(resp.RawBody(), buffer)
			if n > 0 {
				_, writeErr := out.WriteAt(buffer[:n], chunkStart)
				if writeErr != nil {
					return writeErr
				}
				chunkStart += int64(n)
				totalBytesRead.Add(int64(n))
			}

			if err != nil {
				if err == io.EOF {
					return nil // 读取完毕，正常退出
				}
				return err // 读取过程中出错，返回错误
			}
		}
	}
}
