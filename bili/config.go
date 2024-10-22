package main

type Config struct {
	sessdata         string
	ffmpeg           string
	tmpDir           string
	downloadVideo    bool
	downloadAudio    bool
	downloadSubtitle bool
	x                [][]int
}

func NewConfig() *Config {
	return &Config{}
}
