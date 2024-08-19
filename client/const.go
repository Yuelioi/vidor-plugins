package main

import (
	"fmt"
	"path/filepath"
	"runtime"

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

const (
	testUrl = "https://www.bilibili.com/video/BV1bA411R7BN"
	// testUrl      = "https://www.bilibili.com/video/BV1nr421M747/"
	testPagesUrl = "https://www.bilibili.com/video/av1605323758/"
	// testPages4KUrl   = "https://www.bilibili.com/video/av1605323758/"
	testSeasonsUrl   = "https://www.bilibili.com/video/av1206318658/"
	testSeasons4KUrl = "https://www.bilibili.com/video/av1454529712/"
)
