package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	pb "proto"
	"sync"

	"github.com/kkdai/youtube/v2"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	YTBClient    *youtube.Client
	proxyUrl     string
	useProxy     bool
	stopChannels sync.Map
}

func NewClient() *Client {

	//
	proxyURL := ""
	useProxy := true

	proxyStr := proxyURL
	transport := &http.Transport{}
	baseClient := &http.Client{}

	if useProxy {
		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
			fmt.Println("Error parsing proxy URL:", err)
			return nil
		}

		transport.Proxy = http.ProxyURL(proxyURL)
	}
	baseClient.Transport = transport

	return &Client{
		YTBClient: &youtube.Client{
			HTTPClient:  baseClient,
			MaxRoutines: 12,
		},
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

	resp := &pb.InfoResponse{
		Tasks: make([]*pb.Task, 0),
	}

	if isPlaylist(url) {
		playlist, err := c.YTBClient.GetPlaylist(url)
		if err != nil {
			return nil, err
		}

		resp.Author = playlist.Author
		resp.Title = playlist.Title

		for _, entry := range playlist.Videos {
			resp.Tasks = append(resp.Tasks, newTask(
				entry.Title,
				"https://www.bilibili.com/video/av"+entry.ID,
				"",
			))
		}

	} else {
		video, err := c.YTBClient.GetVideo(url)
		if err != nil {
			return nil, err
		}

		resp.Author = video.Author
		resp.Title = video.Title
		resp.Tasks = append(resp.Tasks, newTask(
			video.Title,
			"https://www.bilibili.com/video/av"+video.ID,
			"",
		))

	}

	return resp, nil
}

// 解析
func (c *Client) Parse(pr *pb.ParseRequest) (*pb.ParseResponse, error) {
	fmt.Printf("ParseTasks: %v\n", pr.Tasks[0].Id)

	resp := &pb.ParseResponse{}

	for _, task := range pr.Tasks {

		video, err := c.YTBClient.GetVideo(task.Url)
		if err != nil {
			return nil, err
		}

		// 使用 proto.Clone 来进行深拷贝
		newTask := proto.Clone(task).(*pb.Task)

		// 清空旧的 segment
		newTask.Segments = make([]*pb.Segment, 0)

		segMap := map[string][]*pb.Format{}

		for _, format := range video.Formats {
			fm := &pb.Format{
				Id:       uuid.New().String(),
				Fid:      int64(format.ItagNo),
				MimeType: format.MimeType,
				Label:    format.QualityLabel,
				Url:      format.URL,
			}
			segMap[format.MimeType] = append(segMap[format.MimeType], fm)
		}

		for mimeType, formats := range segMap {
			seg := &pb.Segment{
				MimeType: mimeType,
				Formats:  formats,
			}
			newTask.Segments = append(newTask.Segments, seg)
		}

		resp.Tasks = append(resp.Tasks, newTask)
	}

	return resp, nil
}

func (c *Client) Download(segInfo *pb.Task, seg pb.DownloadService_DownloadServer) error {

	stopChan := make(chan struct{})
	c.stopChannels.Store(segInfo.Id, stopChan)

	return nil
}

func (c *Client) StopDownload(ctx context.Context, sr *pb.StopDownloadRequest) (*pb.StopDownloadResponse, error) {
	return nil, nil
}
