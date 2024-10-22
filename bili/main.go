package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	pb "proto"

	bv "github.com/Yuelioi/bilibili/pkg/endpoints/video"

	"github.com/Yuelioi/bilibili/pkg/bpi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedDownloadServiceServer
	tq         *JobManager
	service    *bpi.BpiService
	grpcServer *grpc.Server
	config     *Config
}

// 初始化
func (s *server) Init(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to connect\n")
	return &empty.Empty{}, nil
}

// 更新数据
func (s *server) Update(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	fmt.Print("someone try to update\n")
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		sessdata := md.Get("plugin.sessdata")
		if len(sessdata) > 0 {
			s.config.sessdata = sessdata[0]
			s.service.Client.SESSDATA = sessdata[0]
			fmt.Printf("sessdata: %v\n", sessdata[0])
		}

		ffmpeg := md.Get("system.ffmpeg")
		if len(ffmpeg) > 0 {
			fmt.Printf("ffmpeg: %v\n", ffmpeg[0])
			s.config.ffmpeg = ffmpeg[0]
		}

		tmps := md.Get("system.tmpdirs")
		if len(tmps) > 0 {
			fmt.Printf("tmps: %v\n", tmps[0])
			s.config.tmpDir = tmps[0]
		}
		return &empty.Empty{}, nil
	}
	return &empty.Empty{}, errors.New("主机未提供相应参数")
}

// 关闭
func (s *server) Shutdown(ctx context.Context, i *empty.Empty) (*empty.Empty, error) {
	// 关闭程序
	go func() {
		s.grpcServer.GracefulStop()
	}()

	// 返回成功响应
	return &empty.Empty{}, nil
}

func (s *server) GetInfo(ctx context.Context, sr *pb.InfoRequest) (*pb.InfoResponse, error) {
	aid, bvid := extractAidBvid(sr.Url)
	vInfo, err := s.service.Video().Info(aid, bvid)
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

	suffix := filepath.Ext(resp.Cover)
	tmpCoverPath := filepath.Join(s.config.tmpDir, "cover", timestamp()+suffix)
	err = downloadCover(resp.Cover, tmpCoverPath)
	if err != nil {
		return nil, err
	}

	resp.Cover = tmpCoverPath
	return resp, nil
}

func (s *server) Parse(ctx context.Context, pr *pb.TasksRequest) (*pb.TasksResponse, error) {
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

		segData, err := s.service.Video().Stream(avid, bvid, cid, 0)
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

func (s *server) Download(tr *pb.TaskRequest, stream pb.DownloadService_DownloadServer) error {
	start := time.Now()

	job, err := NewJob(stream, tr.Task, s.config)
	if err != nil {
		return err
	}

	chains := []Handler{&CoverDownloader{}, &JobRegister{}}

	if s.config.downloadVideo {
		chains = append(chains, &VideoDownloader{})
	}

	if s.config.downloadAudio {
		chains = append(chains, &AudioDownloader{})
	}

	if s.config.downloadVideo && s.config.downloadAudio {
		chains = append(chains, &Combiner{})
	}

	if !(s.config.downloadVideo && s.config.downloadAudio) {
		if s.config.downloadVideo {
			job.video.filepath = job.task.Filepath
		}

		if s.config.downloadAudio {
			pureTitle := sanitizeFileName(tr.Task.Title)
			job.audio.filepath = filepath.Join(tr.Task.WorkDir, pureTitle+".mp4")
		}
	}

	h := createHandlerChain(chains...)
	err = h.Handle(job, jm)
	if err != nil {
		return err
	}
	log.Printf("下载完成：%v", time.Since(start))
	return nil
}

// TODO
func (s *server) Resume(ctx context.Context, pr *pb.TaskRequest) (*pb.TaskResponse, error) {
	return nil, nil
}

func (s *server) Stop(ctx context.Context, sr *pb.TaskRequest) (*pb.TaskResponse, error) {
	id := sr.Id

	return &pb.TaskResponse{
		Id: sr.Id,
	}, fmt.Errorf("task with ID %s not found", id)
}

func main() {

	port := flag.Int("port", 9001, "Port number to listen on")
	flag.Parse()

	lis, err := net.Listen("tcp", "localhost:"+strconv.Itoa(*port))
	if err != nil {
		log.Fatalf("Failed to listen on TCP: %v", err)
	}

	actualPort := lis.Addr().(*net.TCPAddr).Port

	grpcServer := grpc.NewServer()

	s := &server{
		tq:         NewJobManager(),
		service:    bpi.New(),
		grpcServer: grpcServer,
		config:     NewConfig(),
	}

	pb.RegisterDownloadServiceServer(grpcServer, s)

	// 创建健康检查服务
	healthServer := health.NewServer()

	// 设置服务状态为 SERVING
	healthServer.SetServingStatus("health_check", grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册健康检查服务到 gRPC 服务器
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	fmt.Printf("Server Run At Port: %d  PID:%d\n", actualPort, os.Getpid())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
