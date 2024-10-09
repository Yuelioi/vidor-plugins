package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	pb "proto"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
)

type TaskQueue struct {
	mu    sync.Mutex
	tasks map[string]*Task
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		mu:    sync.Mutex{},
		tasks: make(map[string]*Task, 0),
	}
}

func (tq *TaskQueue) AddTask(id, url, tempPath string, contentLength int64, stream pb.DownloadService_DownloadServer) (*Task, error) {
	out, err := os.Create(tempPath)
	if err != nil {
		log.Printf("无法创建文件：%v", err)
		return nil, err
	}

	return &Task{
		id:             id,
		url:            url,
		stopChan:       make(chan struct{}),
		stream:         stream,
		contentLength:  contentLength,
		totalBytesRead: &atomic.Int64{},
		out:            out,
		req:            &resty.Request{},
	}, nil
}

type Task struct {
	id             string
	url            string
	stopChan       chan struct{}
	stream         pb.DownloadService_DownloadServer
	contentLength  int64
	totalBytesRead *atomic.Int64

	out *os.File
	req *resty.Request
}

func autoSetBatchSize(contentLength int64) int64 {
	minBatchSize := int64(2)
	maxBatchSize := int64(5)

	batchSize := int64(math.Sqrt(float64(contentLength) / (1024 * 1024))) // 1MB chunks
	batchSize = int64(math.Max(float64(minBatchSize), float64(math.Min(float64(batchSize), float64(maxBatchSize)))))
	return batchSize
}

func (t *Task) download() error {
	batchSize := autoSetBatchSize(t.contentLength)
	chunkSize := t.contentLength / batchSize
	if chunkSize*batchSize < t.contentLength {
		chunkSize += 1
	}

	defer t.out.Close()

	var wg sync.WaitGroup
	var totalBytesRead atomic.Int64

	subject := NewDownloadSubject()
	observer := NewDownloadObserver(t.stream)
	subject.RegisterObserver(observer)

	ticker := time.NewTicker(time.Duration(timeInterval) * time.Millisecond)

	go func() {
		defer ticker.Stop()
		var previousBytesRead int64

		for range ticker.C {
			currentBytesRead := totalBytesRead.Load()
			bytesRead := currentBytesRead - previousBytesRead
			previousBytesRead = currentBytesRead

			progressMsg := &pb.DownloadProgress{
				Id:         t.id,
				TotalBytes: 100,
				Speed:      bytesRead,
			}

			subject.NotifyObservers(progressMsg)

			if err := t.stream.Send(progressMsg); err != nil {
				return

			}
		}
	}()

	for i := int64(0); i < batchSize; i++ {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == batchSize-1 {
			end = t.contentLength - 1
		}

		wg.Add(1)
		go func(chunkStart, chunkEnd int64) {
			defer wg.Done()
			t.downloadChunk(chunkStart, chunkEnd, &totalBytesRead)
		}(start, end)
	}
	wg.Wait()

	return nil
}

func (t *Task) downloadChunk(chunkStart, chunkEnd int64, totalBytesRead *atomic.Int64) error {

	resp, err := t.req.SetHeader("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd)).Get(t.url)
	if err != nil {
		log.Println("请求失败:", err)
		return err
	}
	defer resp.RawBody().Close()

	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-t.stopChan:
			fmt.Println("Context canceled")
			return fmt.Errorf("download stopped for chunk %d-%d", chunkStart, chunkEnd)

		default:
			n, err := io.ReadFull(resp.RawBody(), buffer)
			if n > 0 {
				_, writeErr := t.out.WriteAt(buffer[:n], chunkStart)
				if writeErr != nil {
					log.Printf("写入文件失败：%v", writeErr)
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
