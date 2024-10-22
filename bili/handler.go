package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-resty/resty/v2"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type Handler interface {
	Handle(job *Job, jm *JobManager) error
	SetNext(next Handler) Handler
}

type BaseHandler struct {
	next Handler
}

func (bh *BaseHandler) Handle(job *Job, jm *JobManager) error {
	if bh.next != nil {
		return bh.next.Handle(job, jm)
	}
	return nil
}

func (bh *BaseHandler) SetNext(next Handler) Handler {
	bh.next = next
	return next
}

type CoverDownloader struct {
	BaseHandler
}

func (bh *CoverDownloader) Handle(j *Job, jm *JobManager) error {

	workDir := filepath.Dir(j.task.Filepath)
	pureTitle := sanitizeFileName(j.task.Title)

	suffix := filepath.Ext(j.task.Cover)
	coverPath := filepath.Join(workDir, pureTitle+suffix)
	if err := downloadCover(j.task.Cover, coverPath); err != nil {
		// 下载封面失败, 不会停止下载任务
		fmt.Printf("下载封面失败, err: %s", err.Error())
	}

	j.task.Cover = coverPath
	return bh.BaseHandler.Handle(j, jm)
}

// 需要在下载前执行
type JobRegister struct {
	BaseHandler
}

func (bh *JobRegister) Handle(j *Job, jm *JobManager) error {
	jm.AddJob(j)
	return bh.BaseHandler.Handle(j, jm)
}

type VideoDownloader struct {
	BaseHandler
}

func (bh *VideoDownloader) Handle(j *Job, jm *JobManager) error {
	req := resty.New().R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Referer", "https://www.bilibili.com/").
		SetHeader("Range", "bytes=0-").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: j.config.sessdata,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(j.video.url)
	if err != nil {
		return fmt.Errorf("下载视频失败1, err: %s", err.Error())
	}

	contentLength, err := strconv.ParseInt(resp.Header().Get("Content-Length"), 10, 64)
	if err != nil {
		return fmt.Errorf("下载视频失败2, err: %s", err.Error())
	}

	j.video.contentLength = contentLength

	go j.monitor(j.video)
	err = j.download(j.video)
	if err != nil {
		return fmt.Errorf("下载视频失败3, err: %s", err.Error())
	}
	return bh.BaseHandler.Handle(j, jm)
}

type AudioDownloader struct {
	BaseHandler
}

func (bh *AudioDownloader) Handle(j *Job, jm *JobManager) error {
	req := resty.New().R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Referer", "https://www.bilibili.com/").
		SetHeader("Range", "bytes=0-").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: j.config.sessdata,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(j.audio.url)
	if err != nil {
		return fmt.Errorf("下载音频失败1, err: %s", err.Error())
	}

	contentLength, err := strconv.ParseInt(resp.Header().Get("Content-Length"), 10, 64)
	if err != nil {
		return fmt.Errorf("下载视音频失败2, err: %s", err.Error())
	}

	j.audio.contentLength = contentLength

	go j.monitor(j.audio)
	err = j.download(j.audio)
	if err != nil {
		return fmt.Errorf("下载音频失败3, err: %s", err.Error())
	}
	return bh.BaseHandler.Handle(j, jm)
}

type Combiner struct {
	BaseHandler
}

func (bh *Combiner) Handle(j *Job, jm *JobManager) error {
	input := []*ffmpeg_go.Stream{ffmpeg_go.Input(j.video.filepath), ffmpeg_go.Input(j.audio.filepath)}
	out := ffmpeg_go.OutputContext(context.Background(), input, j.task.Filepath, ffmpeg_go.KwArgs{"c:v": "copy", "c:a": "aac"})

	_, err := os.Stat(j.config.ffmpeg)
	if err == nil {
		out = out.SetFfmpegPath(j.config.ffmpeg)
	}

	// err = out.OverWriteOutput().WithOutput().Run()

	cmd := out.OverWriteOutput().Compile()

	// TODO关闭cmd弹窗
	// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("合并失败, err: %s", err.Error())
	}
	return bh.BaseHandler.Handle(j, jm)
}

func createHandlerChain(handlers ...Handler) Handler {
	if len(handlers) == 0 {
		return nil
	}

	for i := 0; i < len(handlers)-1; i++ {
		handlers[i].SetNext(handlers[i+1])
	}

	return handlers[0]
}
