package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	pb "proto"
	"strconv"
	"sync"
	"time"

	"github.com/Yuelioi/bilibili/pkg/bpi"
	bv "github.com/Yuelioi/bilibili/pkg/endpoints/video"
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

func newTask(title, url, sessionId, cover string) *pb.Task {
	task := &pb.Task{
		Url:       url,
		SessionId: sessionId,
		Title:     title,
		Cover:     cover,
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
	vInfo, err := c.BpiService.Video().Info(aid, bvid)
	if err != nil {
		return nil, err
	}

	resp := &pb.InfoResponse{
		Title:     vInfo.Data.Title,
		Cover:     vInfo.Data.Pic,
		Author:    vInfo.Data.Owner.Name,
		Tasks:     make([]*pb.Task, 0),
		NeedParse: true,
	}

	if vInfo.Data.IsSeasonDisplay {

		for _, episode := range vInfo.Data.UgcSeason.Sections[0].Episodes {
			resp.Tasks = append(resp.Tasks, newTask(
				episode.Title,
				"https://www.bilibili.com/video/av"+strconv.Itoa(episode.AID),
				strconv.Itoa(episode.CID),
				episode.Arc.Pic,
			))
		}
	} else {
		for _, page := range vInfo.Data.Pages {
			resp.Tasks = append(resp.Tasks, newTask(
				page.Part,
				"https://www.bilibili.com/video/av"+strconv.Itoa(vInfo.Data.AID)+"?p="+strconv.Itoa(page.Page),
				strconv.Itoa(page.CID),
				page.FirstFrame,
			))
		}
	}
	return resp, nil
}

// 解析
func (c *Client) Parse(pr *pb.TasksRequest) (*pb.TasksResponse, error) {
	resp := &pb.TasksResponse{}

	for _, task := range pr.Tasks {
		avid, bvid := extractAidBvid(task.Url)
		cid, err := strconv.Atoi(task.SessionId)
		if err != nil {
			return nil, err
		}

		if !task.Selected {
			resp.Tasks = append(resp.Tasks, task)
			continue
		}

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

		// 处理视频格式
		videoSeg := &pb.Segment{MimeType: "video"}
		for _, video := range segData.Data.Dash.Video {
			format := &pb.Format{
				MimeType: "video",
				Label:    bv.VideoQualityMap[video.ID] + " | " + bv.VideoCodecMap[video.Codecid],
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

func (c *Client) Download(task *pb.Task, tmpDir, ffmpeg string, stream pb.DownloadService_DownloadServer, tq *JobManager) error {
	start := time.Now()

	downloadDir := filepath.Join(tmpDir, "downloading")

	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		err := os.MkdirAll(downloadDir, os.ModePerm)
		if err != nil {
			fmt.Printf("创建目录失败: %v\n", err)
			return err
		}
	} else if err != nil {
		fmt.Printf("检查目录时发生错误: %v\n", err)
		return nil
	}

	job, err := NewJob(stream, c.BpiService.Client.SESSDATA, task, tmpDir, ffmpeg)
	if err != nil {
		return err
	}

	h := createHandlerChain(&CoverDownloader{}, &JobRegister{}, &VideoDownloader{}, &AudioDownloader{}, &Combiner{})
	h.Handle(job, jm)
	log.Printf("下载完成：%v", time.Since(start))

	return nil
}

func (c *Client) StopDownload(ctx context.Context, sr *pb.TaskRequest) (*pb.TaskResponse, error) {
	return nil, nil
}
