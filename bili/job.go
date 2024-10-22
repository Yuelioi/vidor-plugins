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

	"github.com/go-resty/resty/v2"
)

const (
	bufferSize   = 1024 * 256      // 500kb buffer size
	chunkSize    = 5 * 1024 * 1024 // 5MB chunk size
	timeInterval = 1333            // 任务更新周期
)

var (
	jm   *JobManager
	once sync.Once
)

type JobManager struct {
	jobs map[string]*Job
}

type Media struct {
	mediaType      string        // 媒体类型  视频/音频
	url            string        // 下载链接
	filepath       string        // 临时储存路径
	contentLength  int64         // 长度(bytes)
	file           *os.File      // 文件
	totalBytesRead *atomic.Int64 // 已读
}

func NewJobManager() *JobManager {
	once.Do(func() {
		jm = &JobManager{
			jobs: make(map[string]*Job, 0),
		}
	})
	return jm
}

func (jm *JobManager) AddJob(job *Job) {
	jm.jobs[job.task.Id] = job
}

// 一整个下载任务
type Job struct {
	stopChan   chan struct{} //  停止通道
	finishChan chan struct{} // 完成通道
	config     *Config

	stream pb.DownloadService_DownloadServer
	video  *Media
	audio  *Media
	task   *pb.Task
}

func NewJob(stream pb.DownloadService_DownloadServer, task *pb.Task, config *Config) (*Job, error) {
	v := &pb.Format{}
	a := &pb.Format{}

	for _, seg := range task.Segments {
		if seg.MimeType == "video" {
			for _, fm := range seg.Formats {
				if fm.Selected {
					v = fm
				}
			}
		}

		if seg.MimeType == "audio" {
			for _, fm := range seg.Formats {
				if fm.Selected {
					a = fm
				}
			}
		}
	}

	downloadDir := filepath.Join(config.tmpDir, "downloading")

	pureTitle := sanitizeFileName(task.Title)
	vPath := filepath.Join(downloadDir, pureTitle+".video.tmp.mp4")
	aPath := filepath.Join(downloadDir, pureTitle+".audio.tmp.mp3")
	targetPath := filepath.Join(task.WorkDir, pureTitle+".mp4")
	task.Filepath = targetPath

	return &Job{
		stopChan:   make(chan struct{}),
		finishChan: make(chan struct{}),
		stream:     stream,
		config:     config,
		video: &Media{
			mediaType:      "视频",
			url:            v.Url,
			filepath:       vPath,
			file:           &os.File{},
			totalBytesRead: &atomic.Int64{},
		},
		audio: &Media{
			mediaType:      "音频",
			url:            a.Url,
			filepath:       aPath,
			file:           &os.File{},
			totalBytesRead: &atomic.Int64{},
		},
		task: task,
	}, nil
}

func autoSetBatchSize(contentLength int64) int64 {
	minBatchSize := int64(2)
	maxBatchSize := int64(5)

	batchSize := int64(math.Sqrt(float64(contentLength) / (1024 * 1024))) // 1MB chunks
	batchSize = int64(math.Max(float64(minBatchSize), float64(math.Min(float64(batchSize), float64(maxBatchSize)))))
	return batchSize
}

// 监控任务进度
func (j *Job) monitor(m *Media) {
	ticker := time.NewTicker(time.Duration(timeInterval) * time.Millisecond)
	notify := NewDownloadNotification(j.stream)

	var previousBytesRead int64

	for {

		select {
		case <-ticker.C:
			currentBytesRead := m.totalBytesRead.Load()
			bytesRead := currentBytesRead - previousBytesRead
			previousBytesRead = currentBytesRead

			progressMsg := &pb.Task{
				Status:  fmt.Sprintf("下载%s中", m.mediaType),
				Cover:   j.task.Cover,
				Speed:   bytesRead * 1000 / timeInterval,
				Percent: (currentBytesRead * 100 / m.contentLength),
			}

			fmt.Printf("progressMsg: %v\n", progressMsg)
			notify.OnUpdate(progressMsg)
			// 如果没关闭
		case <-j.stopChan:
			ticker.Stop()
		case <-j.finishChan:
			ticker.Stop()
		}
	}

}

func (j *Job) download(m *Media) error {
	batchSize := autoSetBatchSize(m.contentLength)
	chunkSize := m.contentLength / batchSize
	if chunkSize*batchSize < m.contentLength {
		chunkSize += 1
	}

	file, err := os.Create(m.filepath)
	if err != nil {
		return err
	}
	m.file = file
	defer m.file.Close()

	var wg sync.WaitGroup

	for i := int64(0); i < batchSize; i++ {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == batchSize-1 {
			end = m.contentLength - 1
		}

		wg.Add(1)
		go func(chunkStart, chunkEnd int64) {
			defer wg.Done()
			if err := j.downloadChunk(chunkStart, chunkEnd, m); err != nil {
				return
			}
		}(start, end)
	}
	wg.Wait()

	return nil
}

func (j *Job) downloadChunk(chunkStart, chunkEnd int64, m *Media) error {

	req := resty.New().R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd)).
		SetHeader("Referer", "https://www.bilibili.com/").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: j.config.sessdata,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(m.url)
	if err != nil {
		log.Println("请求失败:", err)
		return err
	}
	defer resp.RawBody().Close()

	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-j.stopChan:
			fmt.Println("Context canceled")
			return fmt.Errorf("download stopped for chunk %d-%d", chunkStart, chunkEnd)

		default:
			n, err := io.ReadFull(resp.RawBody(), buffer)
			if n > 0 {
				_, writeErr := m.file.WriteAt(buffer[:n], chunkStart)
				if writeErr != nil {
					log.Printf("写入文件失败：%v", writeErr)
					return writeErr
				}
				chunkStart += int64(n)
				m.totalBytesRead.Add(int64(n))
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
