package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	pb "proto"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Yuelioi/bilibili/pkg/bpi"
	bv "github.com/Yuelioi/bilibili/pkg/endpoints/video"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	BpiService   *bpi.BpiService
	stopChannels sync.Map
}

func NewClient() *Client {
	return &Client{
		BpiService: bpi.New(),
	}
}

func newTask(title, url, sessionId string) *pb.Task {
	uid := uuid.New()
	task := &pb.Task{
		Id:        uid.String(),
		Url:       url,
		SessionId: sessionId,
		Title:     title,
	}

	for _, mimeType := range []string{"video", "audio"} {
		seg := &pb.Segment{}
		seg.MimeType = mimeType
		seg.Formats = []*pb.Format{
			{MimeType: mimeType, Label: "未解析", Code: "未解析"},
		}
		task.Segments = append(task.Segments, seg)

	}
	return task
}

// 获取列表基础信息
func (c *Client) GetInfo(url string) (*pb.InfoResponse, error) {
	fmt.Printf("GetInfo: %v\n", url)
	aid, bvid := extractAidBvid(url)
	Info, err := c.BpiService.Video().Info(aid, bvid)
	if err != nil {
		return nil, err
	}

	resp := &pb.InfoResponse{

		Title:  Info.Data.Title,
		Cover:  Info.Data.Pic,
		Author: Info.Data.Owner.Name,
		Tasks:  make([]*pb.Task, 0),
	}

	if Info.Data.IsSeasonDisplay {

		for _, episode := range Info.Data.UgcSeason.Sections[0].Episodes {
			resp.Tasks = append(resp.Tasks, newTask(
				episode.Title,
				"https://www.bilibili.com/video/av"+strconv.Itoa(episode.AID),
				strconv.Itoa(episode.CID),
			))
		}

	} else {
		for _, page := range Info.Data.Pages {
			resp.Tasks = append(resp.Tasks, newTask(
				page.Part,
				"https://www.bilibili.com/video/av"+strconv.Itoa(Info.Data.AID)+"?p="+strconv.Itoa(page.Page),
				strconv.Itoa(page.CID),
			))
		}
	}
	return resp, nil
}

// 解析
func (c *Client) Parse(pr *pb.TasksRequest) (*pb.TasksResponse, error) {
	fmt.Printf("ParseTasks: %v\n", pr.Tasks[0].Id)

	resp := &pb.TasksResponse{}

	for _, task := range pr.Tasks {
		avid, bvid := extractAidBvid(task.Url)
		cid, err := strconv.Atoi(task.SessionId)
		if err != nil {
			return nil, err
		}

		print(avid, bvid, cid)

		segData, err := c.BpiService.Video().Stream(avid, bvid, cid, 0)
		if err != nil {
			return nil, err
		}

		if segData.Code != 0 {
			return nil, errors.New("获取数据失败")
		}

		// 过滤掉充电视频
		if segData.Data.AcceptDescription[0] == "试看" {
			return nil, errors.New("没有观看权限")
		}

		// 使用 proto.Clone 来进行深拷贝
		newTask := proto.Clone(task).(*pb.Task)

		// 清空旧的 segment
		newTask.Segments = make([]*pb.Segment, 0)

		fmt.Printf("segData: %v\n", segData.Data)

		// 处理视频格式
		videoSeg := &pb.Segment{MimeType: "video"}
		for _, video := range segData.Data.Dash.Video {
			format := &pb.Format{
				Id:       uuid.New().String(),
				MimeType: "video",
				Label:    bv.VideoQualityMap[video.ID],
				Code:     bv.VideoCodecMap[video.Codecid],
				Url:      video.BaseURL,
			}
			videoSeg.Formats = append(videoSeg.Formats, format)
		}
		newTask.Segments = append(newTask.Segments, videoSeg)

		// 处理音频格式
		audioSeg := &pb.Segment{MimeType: "audio"}
		for _, audio := range segData.Data.Dash.Audio {
			format := &pb.Format{
				Id:       uuid.New().String(),
				MimeType: "audio",
				Label:    bv.AudioQualityMap[audio.ID],
				Url:      audio.BaseURL,
			}
			audioSeg.Formats = append(audioSeg.Formats, format)
		}
		newTask.Segments = append(newTask.Segments, audioSeg)

		resp.Tasks = append(resp.Tasks, newTask)
	}

	return resp, nil
}

func (c *Client) Download(segInfo *pb.Task, seg pb.DownloadService_DownloadServer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopChan := make(chan struct{})
	c.stopChannels.Store(segInfo.Id, stopChan)

	start := time.Now()

	req := c.BpiService.Client.HTTPClient.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Referer", "https://www.bilibili.com/").
		SetHeader("Range", "bytes=0-").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: c.BpiService.Client.SESSDATA,
		}).SetDoNotParseResponse(true)

	url1 := segInfo.Segments[0].Formats[0].Url
	// url2 := segInfo.Segments[0].Formats[0].Url
	resp, err := req.Get(url1)

	if err != nil {
		return err
	}
	// 从响应头中获取 Content-Range 的值
	contentLengthStr := resp.Header().Get("Content-Length")
	var contentLength int64
	if contentLengthStr != "" {
		contentLength, err = strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			return err
		}
	} else {
		return errors.New("Content-Length header is missing")
	}
	fmt.Printf("contentLength: %v\n", contentLength)

	if err := c.download(ctx, seg, segInfo.Id, url1, contentLength, "temp.mp4"); err != nil {
		log.Printf("下载失败：%v", err)
		return err
	}

	log.Printf("下载完成：%v", time.Since(start))

	return nil
}

func (c *Client) StopDownload(ctx context.Context, sr *pb.TaskRequest) (*pb.TaskResponse, error) {
	return nil, nil
}

const (
	bufferSize = 1024 * 256      // 500kb buffer size
	chunkSize  = 5 * 1024 * 1024 // 5MB chunk size
)

func autoSetBatchSize(contentLength int64) int64 {
	minBatchSize := int64(2)
	maxBatchSize := int64(5)

	batchSize := int64(math.Sqrt(float64(contentLength) / (1024 * 1024))) // 1MB chunks
	batchSize = int64(math.Max(float64(minBatchSize), float64(math.Min(float64(batchSize), float64(maxBatchSize)))))
	return batchSize
}
func (c *Client) download(ctx context.Context, seg pb.DownloadService_DownloadServer, id string, url string, contentLength int64, tempPath string) error {
	batchSize := autoSetBatchSize(contentLength)
	chunkSize := contentLength / batchSize
	if chunkSize*batchSize < contentLength {
		chunkSize += 1
	}

	out, err := os.Create(tempPath)
	if err != nil {
		log.Printf("无法创建文件：%v", err)
		return err
	}
	defer out.Close()

	var wg sync.WaitGroup
	var totalBytesRead atomic.Int64

	var finished bool

	timeInterval := 333

	ticker := time.NewTicker(time.Duration(timeInterval) * time.Millisecond)

	go func() {
		var previousBytesRead int64
		defer ticker.Stop()

		for range ticker.C {
			if !finished {
				currentBytesRead := totalBytesRead.Load()
				bytesRead := currentBytesRead - previousBytesRead
				previousBytesRead = currentBytesRead

				speed := float64(bytesRead) // Speed in B/s

				progressMsg := &pb.DownloadProgress{
					Id:         id,
					TotalBytes: 100,
					Speed:      fmt.Sprintf("%.2f MB/s", speed*1000/(1024*1024*float64(timeInterval))),
				}

				if err := seg.Send(progressMsg); err != nil {
					return
				}

			} else {
				ticker.Stop()
				return
			}

		}

	}()

	for i := int64(0); i < batchSize; i++ {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == batchSize-1 {
			end = contentLength - 1
		}

		wg.Add(1)
		go func(chunkStart, chunkEnd int64) {
			defer wg.Done()
			c.downloadChunk(ctx, id, url, chunkStart, chunkEnd, out, &totalBytesRead)
		}(start, end)
	}
	wg.Wait()
	finished = true

	return nil
}

func (c *Client) downloadChunk(ctx context.Context, id string, url string, chunkStart, chunkEnd int64, out *os.File, totalBytesRead *atomic.Int64) error {
	c.BpiService.Client.HTTPClient.SetTimeout(0)

	stopChan, ok := c.stopChannels.Load(id)

	if !ok {
		return nil
	}

	req := c.BpiService.Client.HTTPClient.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd)).
		SetHeader("Referer", "https://www.bilibili.com/").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: c.BpiService.Client.SESSDATA,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(url)
	if err != nil {
		log.Println("请求失败:", err)
		return err
	}
	defer resp.RawBody().Close()

	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-stopChan.(chan struct{}):
			fmt.Println("Context canceled")
			return fmt.Errorf("download stopped for chunk %d-%d", chunkStart, chunkEnd)

		case <-ctx.Done():
			fmt.Println("Context canceled")
			return ctx.Err()
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
