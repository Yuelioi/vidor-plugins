package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	pb "proto"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Yuelioi/bilibili/pkg/client"
)

const (
	bufferSize   = 1024 * 256      // 500kb buffer size
	chunkSize    = 5 * 1024 * 1024 // 5MB chunk size
	timeInterval = 1333            // 任务更新周期
)

type JobManager struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

var (
	jm   *JobManager
	once sync.Once
)

func NewJobManager() *JobManager {
	once.Do(func() {
		jm = &JobManager{
			mu:   sync.Mutex{},
			jobs: make(map[string]*Job, 0),
		}
	})
	return jm
}

func (tq *JobManager) AddJob(job *Job) error {
	tq.jobs[job.task.Id] = job

	return nil
}

type Job struct {
	stopChan      chan struct{}
	stream        pb.DownloadService_DownloadServer
	contentLength int64
	vTmp          string
	aTmp          string
	client        *client.Client
	task          *pb.Task
}

func NewJob(stream pb.DownloadService_DownloadServer, client *client.Client, task *pb.Task, tmpDir string) *Job {

	workDir := filepath.Dir(task.Filepath)
	downloadDir := filepath.Join(tmpDir, "downloading")

	pureTitle := sanitizeFileName(task.Title)
	vPath := filepath.Join(downloadDir, pureTitle+".video.tmp.mp4")
	aPath := filepath.Join(downloadDir, pureTitle+".audio.tmp.mp3")
	targetPath := filepath.Join(workDir, pureTitle+".mp4")
	task.Filepath = targetPath

	return &Job{
		stopChan:      make(chan struct{}),
		stream:        stream,
		contentLength: 0,
		vTmp:          vPath,
		aTmp:          aPath,
		client:        client,
		task:          task,
	}
}

func autoSetBatchSize(contentLength int64) int64 {
	minBatchSize := int64(2)
	maxBatchSize := int64(5)

	batchSize := int64(math.Sqrt(float64(contentLength) / (1024 * 1024))) // 1MB chunks
	batchSize = int64(math.Max(float64(minBatchSize), float64(math.Min(float64(batchSize), float64(maxBatchSize)))))
	return batchSize
}

func download(j *Job) error {
	batchSize := autoSetBatchSize(j.contentLength)
	chunkSize := j.contentLength / batchSize
	if chunkSize*batchSize < j.contentLength {
		chunkSize += 1
	}

	out, err := os.Create(j.task.Filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	var wg sync.WaitGroup
	var totalBytesRead atomic.Int64

	ticker := time.NewTicker(time.Duration(timeInterval) * time.Millisecond)
	notify := NewDownloadNotification(j.stream)

	go func() {
		defer ticker.Stop()
		var previousBytesRead int64

		for range ticker.C {
			currentBytesRead := totalBytesRead.Load()
			bytesRead := currentBytesRead - previousBytesRead
			previousBytesRead = currentBytesRead

			progressMsg := &pb.Task{
				Id:    j.task.Id,
				Cover: "bytes",
				Speed: bytesRead,
			}

			if err := j.stream.Send(progressMsg); err != nil {
				return

			}

			notify.OnUpdate(progressMsg)
		}
	}()

	for i := int64(0); i < batchSize; i++ {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == batchSize-1 {
			end = j.contentLength - 1
		}

		wg.Add(1)
		go func(chunkStart, chunkEnd int64) {
			defer wg.Done()
			j.downloadChunk(chunkStart, chunkEnd, &totalBytesRead)
		}(start, end)
	}
	wg.Wait()

	return nil
}

func (t *Job) downloadChunk(chunkStart, chunkEnd int64, totalBytesRead *atomic.Int64) error {
	req := t.client.HTTPClient.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd)).
		SetHeader("Referer", "https://www.bilibili.com/").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: t.client.SESSDATA,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(t.task.Url)
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
				_, writeErr := out.WriteAt(buffer[:n], chunkStart)
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
