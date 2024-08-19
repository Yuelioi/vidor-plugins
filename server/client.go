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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Yuelioi/bilibili/pkg/bpi"
	bv "github.com/Yuelioi/bilibili/pkg/endpoints/video"
	"github.com/google/uuid"
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

func newStreamInfo(title, url, sessionId string) *pb.StreamInfo {
	uid := uuid.New()
	info := &pb.StreamInfo{
		Id:        uid.String(),
		Url:       url,
		SessionId: sessionId,
		Title:     title,
	}

	for _, mimeType := range []string{"video", "audio"} {
		stream := &pb.Stream{}
		stream.MimeType = mimeType
		stream.Formats = []*pb.Format{
			{MimeType: mimeType, Label: "未解析", Code: "未解析"},
		}
		info.Streams = append(info.Streams, stream)

	}
	return info
}

func (c *Client) Info(url string) (*pb.ShowResponse, error) {

	aid, bvid := extractAidBvid(url)
	videoInfo, err := c.BpiService.Video().Info(aid, bvid)
	if err != nil {
		return nil, err
	}

	resp := &pb.ShowResponse{

		Title:       videoInfo.Data.Title,
		Cover:       videoInfo.Data.Pic,
		Author:      videoInfo.Data.Owner.Name,
		StreamInfos: make([]*pb.StreamInfo, 0),
	}

	if videoInfo.Data.IsSeasonDisplay {

		for _, episode := range videoInfo.Data.UgcSeason.Sections[0].Episodes {
			resp.StreamInfos = append(resp.StreamInfos, newStreamInfo(
				episode.Title,
				"https://www.bilibili.com/video/"+strconv.Itoa(episode.AID),
				strconv.Itoa(episode.CID),
			))
		}

	} else {
		for _, page := range videoInfo.Data.Pages {
			resp.StreamInfos = append(resp.StreamInfos, newStreamInfo(
				page.Part,
				"https://www.bilibili.com/video/"+strconv.Itoa(videoInfo.Data.AID)+"?p="+strconv.Itoa(page.Page),
				strconv.Itoa(page.CID),
			))
		}
	}
	return resp, nil
}

func (c *Client) Parse(pr *pb.ParseRequest) (*pb.ParseResponse, error) {
	resp := &pb.ParseResponse{}

	for _, info := range pr.StreamInfos {
		avid, bvid := extractAidBvid(info.Url)
		cid, err := strconv.Atoi(info.SessionId)
		if err != nil {
			return nil, err
		}
		streamData, err := c.BpiService.Video().Stream(avid, bvid, cid, 0)
		if err != nil {
			return nil, err
		}

		if streamData.Code != 0 {
			return nil, errors.New("获取数据失败")
		}

		newStreamInfo := &pb.StreamInfo{}

		for _, video := range streamData.Data.Dash.Video {

			stream := &pb.Stream{}
			stream.MimeType = "video"

			label := bv.VideoQualityMap[video.ID]
			code := bv.VideoCodecMap[video.Codecid]
			stream.Formats = []*pb.Format{
				{Id: int64(video.ID), MimeType: "video", Label: label, Code: code, Url: video.BaseURL},
			}
			newStreamInfo.Streams = append(newStreamInfo.Streams, stream)
		}
		for _, audio := range streamData.Data.Dash.Audio {

			stream := &pb.Stream{}
			stream.MimeType = "audio"

			label := bv.AudioQualityMap[audio.ID]
			stream.Formats = []*pb.Format{
				{Id: int64(audio.ID), MimeType: "video", Label: label, Code: "", Url: audio.BaseURL},
			}
			newStreamInfo.Streams = append(newStreamInfo.Streams, stream)
		}

		resp.StreamInfos = append(resp.StreamInfos, newStreamInfo)
	}

	return resp, nil
}

func (c *Client) Download(streamInfo *pb.StreamInfo, stream pb.DownloadService_DownloadServer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopChan := make(chan struct{})
	c.stopChannels.Store(streamInfo.Id, stopChan)

	start := time.Now()

	req := c.BpiService.Client.HTTPClient.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Referer", "https://www.bilibili.com/").
		SetHeader("Range", "bytes=0-0"). // 只请求第一个字节
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: c.BpiService.Client.SESSDATA,
		})

	url1 := streamInfo.Streams[0].Formats[0].Url
	// url2 := streamInfo.Streams[0].Formats[0].Url
	resp, err := req.Get(url1)

	if err != nil {
		return err
	}
	// 从响应头中获取 Content-Range 的值
	contentRange := resp.Header().Get("Content-Range")
	fmt.Printf("contentRange: %v\n", contentRange)

	parts := strings.Split(contentRange, "/")
	if len(parts) != 2 {
		log.Println("Content-Range 头格式不正确:", contentRange)
		return fmt.Errorf("无法解析视频大小")
	}

	contentLength, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		log.Println("解析 Content-Range 失败:", err)
		return err
	}

	if err := c.download(ctx, stream, streamInfo.Id, url1, contentLength, "temp.mp4"); err != nil {
		log.Printf("下载失败：%v", err)
		return err
	}

	log.Printf("下载完成：%v", time.Since(start))

	return nil
}

func (c *Client) StopDownload(ctx context.Context, sr *pb.StopDownloadRequest) (*pb.StopDownloadResponse, error) {
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
func (c *Client) download(ctx context.Context, stream pb.DownloadService_DownloadServer, id string, url string, contentLength int64, tempPath string) error {
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

				if err := stream.Send(progressMsg); err != nil {
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
