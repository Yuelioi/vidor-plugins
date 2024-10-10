package main

import (
	"net/http"
	"path/filepath"
	"strconv"
)

type Handler interface {
	Handle(job *Job, jm *JobManager) error
	SetNext(next Handler)
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

func (bh *CoverDownloader) Handle(job *Job, jm *JobManager) error {

	workDir := filepath.Dir(job.task.Filepath)
	pureTitle := sanitizeFileName(job.task.Title)

	suffix := filepath.Ext(job.task.Cover)
	coverPath := filepath.Join(workDir, pureTitle+suffix)
	if err := downloadCover(job.task.Cover, coverPath); err != nil {
		return err
	}

	job.task.Cover = coverPath
	return nil
}

// 需要在下载前执行
type JobRegister struct {
	BaseHandler
}

func (bh *JobRegister) Handle(job *Job, jm *JobManager) error {
	return jm.AddJob(job)
}

type VideoDownloader struct {
	BaseHandler
}

func (bh *VideoDownloader) Handle(job *Job) error {
	req := job.client.HTTPClient.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Referer", "https://www.bilibili.com/").
		SetHeader("Range", "bytes=0-").
		SetCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: job.client.SESSDATA,
		}).SetDoNotParseResponse(true)

	resp, err := req.Get(job.task.Url)
	if err != nil {
		return err
	}

	contentLength, err := strconv.ParseInt(resp.Header().Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	return download(job)

}
