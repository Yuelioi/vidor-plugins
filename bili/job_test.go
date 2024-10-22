package main

import (
	"fmt"
	"os"
	"path/filepath"
	"proto"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/joho/godotenv"
)

func LoadEnv() {

	_, filename, _, _ := runtime.Caller(0)
	env := filepath.Join(filepath.Dir(filename), "..", ".env")

	// Attempt to load the .env file
	err := godotenv.Load(env)
	if err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
	}
}

func TestDownload(t *testing.T) {
	LoadEnv()
	sessdata := os.Getenv("SESSDATA")
	c := NewConfig()
	c.sessdata = sessdata

	task := newTask("output", "url string", "sessionId string", "cover string")
	task.Segments = make([]*proto.Segment, 0)
	job, err := NewJob(nil, task, c)

	job.video = &Media{
		url:            "https://cn-sccd-ct-01-18.bilivideo.com/upgcxcode/99/88/196018899/196018899-1-30120.m4s?e=ig8euxZM2rNcNbdlhoNvNC8BqJIzNbfqXBvEqxTEto8BTrNvN0GvT90W5JZMkX_YN0MvXg8gNEV4NC8xNEV4N03eN0B5tZlqNxTEto8BTrNvNeZVuJ10Kj_g2UB02J0mN0B5tZlqNCNEto8BTrNvNC7MTX502C8f2jmMQJ6mqF2fka1mqx6gqj0eN0B599M=&uipk=5&nbs=1&deadline=1728637696&gen=playurlv2&os=bcache&oi=3728857225&trid=0000e5f24163e173435b901f1cee1f5200d0u&mid=4279370&platform=pc&og=hw&upsig=7424251f18ec4e677546dbd069b3fb10&uparams=e,uipk,nbs,deadline,gen,os,oi,trid,mid,platform,og&cdnid=62618&bvc=vod&nettype=0&orderid=0,3&buvid=DF94E3F4-25F3-70D5-57F0-DDDFB8604B8659351infoc&build=0&f=u_0_0&agrr=1&bw=1757868&logo=80000000",
		filepath:       "./temp.video.mp4",
		totalBytesRead: &atomic.Int64{},
	}
	job.audio = &Media{
		url:            "https://xy119x188x114x50xy.mcdn.bilivideo.cn:8082/v1/resource/196018899_nb3-1-30280.m4s?agrr=1&build=0&buvid=DF94E3F4-25F3-70D5-57F0-DDDFB8604B8659351infoc&bvc=vod&bw=35877&deadline=1728637696&e=ig8euxZM2rNcNbdlhoNvNC8BqJIzNbfqXBvEqxTEto8BTrNvN0GvT90W5JZMkX_YN0MvXg8gNEV4NC8xNEV4N03eN0B5tZlqNxTEto8BTrNvNeZVuJ10Kj_g2UB02J0mN0B5tZlqNCNEto8BTrNvNC7MTX502C8f2jmMQJ6mqF2fka1mqx6gqj0eN0B599M%3D&f=u_0_0&gen=playurlv2&logo=A0020000&mcdnid=50010821&mid=4279370&nbs=1&nettype=0&og=cos&oi=3728857225&orderid=0%2C3&os=mcdn&platform=pc&sign=7c6309&traceid=trwUFmsOBWlbQu_0_e_N&uipk=5&uparams=e%2Cuipk%2Cnbs%2Cdeadline%2Cgen%2Cos%2Coi%2Ctrid%2Cmid%2Cplatform%2Cog&upsig=0941a709306700b51650918e7b5e7a3d",
		filepath:       "./temp.audio.mp3",
		totalBytesRead: &atomic.Int64{},
	}
	if err != nil {
		t.Errorf("Failed to create job: %v", err)
		return
	}

	h := createHandlerChain(&VideoDownloader{}, &AudioDownloader{}, &Combiner{})
	err = h.Handle(job, jm)
	fmt.Printf("err: %v\n", err)
}
