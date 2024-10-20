package main

type Config struct {
	sessdata         string
	ffmpeg           string
	tmpDir           string
	downloadVideo    bool
	downloadAudio    bool
	downloadSubtitle bool
	download         bool
}

func NewConfig() *Config {
	return &Config{}
}
