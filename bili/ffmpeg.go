package main

import (
	"context"
	"os"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

// 合并音频与视频
func CombineAV(ctx context.Context, ffmpegPath string, input_v, input_a, output_v string) (err error) {

	input := []*ffmpeg_go.Stream{ffmpeg_go.Input(input_v), ffmpeg_go.Input(input_a)}
	out := ffmpeg_go.OutputContext(ctx, input, output_v, ffmpeg_go.KwArgs{"c:v": "copy", "c:a": "aac"})

	_, err = os.Stat(ffmpegPath)
	if err == nil {
		out = out.SetFfmpegPath(ffmpegPath)
	}

	// err = out.OverWriteOutput().WithOutput().Run()

	cmd := out.OverWriteOutput().Compile()

	// TODO关闭cmd弹窗
	// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Run()
	return err
}
