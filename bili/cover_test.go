package main

import "testing"

func TestDownloadCover(t *testing.T) {
	s := &server{}
	testUrl := "https://i2.hdslb.com/bfs/archive/91b60beaa3a6a4443e4c2c683fd858ca79cb5003.png"
	s.downloadCover(testUrl, "./test.jpg")
}
