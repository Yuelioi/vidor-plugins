package main

import (
	"errors"
	pb "proto"
	"strconv"

	"github.com/Yuelioi/bilibili/pkg/bpi"
	bv "github.com/Yuelioi/bilibili/pkg/endpoints/video"
	"github.com/google/uuid"
)

type Client struct {
	BpiService *bpi.BpiService
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
				{Id: int64(video.ID), MimeType: "video", Label: label, Code: code},
			}
			newStreamInfo.Streams = append(newStreamInfo.Streams, stream)
		}
		for _, audio := range streamData.Data.Dash.Audio {

			stream := &pb.Stream{}
			stream.MimeType = "audio"

			label := bv.AudioQualityMap[audio.ID]
			stream.Formats = []*pb.Format{
				{Id: int64(audio.ID), MimeType: "video", Label: label, Code: ""},
			}
			newStreamInfo.Streams = append(newStreamInfo.Streams, stream)
		}

		resp.StreamInfos = append(resp.StreamInfos, newStreamInfo)
	}

	return resp, nil
}
