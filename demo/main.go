package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"net/http"
	"net/url"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"

	"github.com/go-resty/resty/v2"
	"github.com/kkdai/youtube/v2"
)

type Media struct {
	mediaType      string        // 媒体类型  视频/音频
	url            string        // 链接
	filepath       string        // 临时储存路径
	contentLength  int64         // 长度(bytes)
	file           *os.File      // 文件
	totalBytesRead *atomic.Int64 // 已读
	finishChan     chan struct{} // 完成通道
}

func getProxyClient() (*http.Client, error) {
	proxyUrl, err := url.Parse("http://127.0.0.1:10809")
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl), // 这里需要提供你的代理URL
		// 可以根据需要调整其他参数
	}

	// 创建一个HTTP客户端，并将上面创建的Transport与之关联
	httpClient := &http.Client{
		Transport: transport,
	}
	return httpClient, nil
}

func parse(url string) {
	// 创建一个新的Transport并设置代理
	httpClient, err := getProxyClient()
	if err != nil {
		fmt.Printf("err1: %v\n", err)
		return
	}

	// 使用带有代理的HTTP客户端初始化你的YouTube客户端
	c := youtube.Client{
		HTTPClient: httpClient,
	}

	v, err := c.GetVideo(url)
	if err != nil {
		fmt.Printf("err2: %v\n", err)
		return
	}

	for _, f := range v.Formats {
		fmt.Printf("f.MimeType: %v Quality %v QualityLabel %v\n", f.MimeType, f.Quality, f.QualityLabel)
	}

	videos := FilterFormats(v.Formats, "video")
	audios := FilterFormats(v.Formats, "audio")

	bv := GetBestHighFormat(videos)
	ba := GetBestHighFormat(audios)

	pureTitle := sanitizeFileName(v.Title)

	tmpV := pureTitle + ".tmp.mp4"
	tmpA := pureTitle + ".tmp.mp3"
	outV := pureTitle + ".mp4"

	mV := &Media{
		mediaType:      "video",
		url:            bv.URL,
		filepath:       tmpV,
		totalBytesRead: &atomic.Int64{},
		finishChan:     make(chan struct{}),
	}

	mA := &Media{
		mediaType:      "audio",
		url:            ba.URL,
		filepath:       tmpA,
		totalBytesRead: &atomic.Int64{},
		finishChan:     make(chan struct{}),
	}

	err = getContentLength(mV)
	if err != nil {
		fmt.Printf("err3: %v\n", err)
		return
	}
	err = getContentLength(mA)
	if err != nil {
		fmt.Printf("err4: %v\n", err)
		return
	}

	download(mV)
	download(mA)

	CombineAV("", tmpA, tmpV, outV)
}

func main() {
	videoLinks := []string{
		"https://www.youtube.com/watch?v=vkOhsJpclxY",
		"https://www.youtube.com/watch?v=SNIE80rJRiM",
		"https://www.youtube.com/watch?v=jS_A1V_9OYI",
		"https://www.youtube.com/watch?v=UbKqqjDy38E",
		"https://www.youtube.com/watch?v=EaabtWuycmk",
		"https://www.youtube.com/watch?v=Zscr1k_A36E",
		"https://www.youtube.com/watch?v=Ttp3QbQFgtc",
		"https://www.youtube.com/watch?v=sP5NpARVX0M",
		"https://www.youtube.com/watch?v=fBKxZh5fejI",
		"https://www.youtube.com/watch?v=DCfk7tc_KqE",
	}

	for _, url := range videoLinks {
		parse(url)
	}
}

// 合并音频与视频
func CombineAV(ffmpegPath string, input_v, input_a, output_v string) (err error) {

	input := []*ffmpeg_go.Stream{ffmpeg_go.Input(input_v), ffmpeg_go.Input(input_a)}
	out := ffmpeg_go.OutputContext(context.Background(), input, output_v, ffmpeg_go.KwArgs{"c:v": "copy", "c:a": "aac"})

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

const bufferSize = 512 * 1024

func getContentLength(m *Media) error {

	httpClient, err := getProxyClient()
	if err != nil {
		fmt.Printf("err1: %v\n", err)
		return nil
	}

	// 创建一个新的resty客户端，并设置代理
	client := resty.NewWithClient(httpClient)

	req := client.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Range", "bytes=0-").SetDoNotParseResponse(true)

	resp, err := req.Get(m.url)
	if err != nil {
		return fmt.Errorf("下载音频失败1, err: %s", err.Error())
	}

	contentLength, err := strconv.ParseInt(resp.Header().Get("Content-Length"), 10, 64)
	if err != nil {
		return fmt.Errorf("下载视音频失败2, err: %s", err.Error())
	}

	m.contentLength = contentLength
	return nil
}

func download(m *Media) error {
	batchSize := autoSetBatchSize(m.contentLength)
	chunkSize := m.contentLength / batchSize
	if chunkSize*batchSize < m.contentLength {
		chunkSize += 1
	}

	file, err := os.Create(m.filepath)
	if err != nil {
		return err
	}
	m.file = file
	defer m.file.Close()

	var wg sync.WaitGroup

	for i := int64(0); i < batchSize; i++ {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == batchSize-1 {
			end = m.contentLength - 1
		}

		wg.Add(1)
		go func(chunkStart, chunkEnd int64) {
			defer wg.Done()
			if err := downloadChunk(chunkStart, chunkEnd, m); err != nil {
				return
			}
		}(start, end)
	}
	wg.Wait()

	fmt.Printf("%s下载完毕\n", m.mediaType)

	return nil
}

func autoSetBatchSize(contentLength int64) int64 {
	minBatchSize := int64(2)
	maxBatchSize := int64(5)

	batchSize := int64(math.Sqrt(float64(contentLength) / (1024 * 1024))) // 1MB chunks
	batchSize = int64(math.Max(float64(minBatchSize), float64(math.Min(float64(batchSize), float64(maxBatchSize)))))
	return batchSize
}

func downloadChunk(chunkStart, chunkEnd int64, m *Media) error {

	httpClient, err := getProxyClient()
	if err != nil {
		fmt.Printf("err1: %v\n", err)
		return nil
	}

	// 创建一个新的resty客户端，并设置代理
	client := resty.NewWithClient(httpClient)

	req := client.R().
		SetHeader("Accept-Ranges", "bytes").
		SetHeader("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd)).
		SetDoNotParseResponse(true)

	resp, err := req.Get(m.url)
	if err != nil {
		log.Println("请求失败:", err)
		return err
	}
	defer resp.RawBody().Close()

	buffer := make([]byte, bufferSize)

	for {

		n, err := io.ReadFull(resp.RawBody(), buffer)
		if n > 0 {
			_, writeErr := m.file.WriteAt(buffer[:n], chunkStart)
			if writeErr != nil {
				log.Printf("写入文件失败：%v", writeErr)
				return writeErr
			}
			chunkStart += int64(n)
			m.totalBytesRead.Add(int64(n))
		}

		if err != nil {
			if err == io.EOF {
				return nil // 读取完毕，正常退出
			}

			return err // 读取过程中出错，返回错误
		}

	}
}

func sanitizeFileName(input string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	sanitized := re.ReplaceAllString(input, "_")

	sanitized = strings.TrimSpace(sanitized)
	sanitized = strings.Trim(sanitized, ".")

	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}

// FilterFormats filters a list of YouTube formats based on a specific kind (e.g. "video/mp4")
func FilterFormats(formats youtube.FormatList, kind string) []youtube.Format {
	var filteredFormats []youtube.Format
	for _, format := range formats {
		if strings.Contains(format.MimeType, kind) {
			filteredFormats = append(filteredFormats, format)
		}
	}
	return filteredFormats
}

// GetBestHighFormat returns the format with the highest bitrate from a list of formats
func GetBestHighFormat(formats []youtube.Format) youtube.Format {
	var bestFormat youtube.Format
	for _, format := range formats {
		if format.Bitrate > bestFormat.Bitrate {
			bestFormat = format
		}
	}
	return bestFormat
}

func DownloadVideoOrAudioByVideoData(client *youtube.Client, video *youtube.Video, folder string, index int) {
	title := video.Title

	if index != 0 {
		title = strconv.Itoa(index) + "_" + title
	}

	formats_highest_v := GetBestHighFormat(FilterFormats(video.Formats, "video"))
	formats_highest_a := GetBestHighFormat(FilterFormats(video.Formats, "audio"))

	DownloadStream(client, video, &formats_highest_v, folder+"/"+title+".mp4")
	DownloadStream(client, video, &formats_highest_a, folder+"/"+title+".flc")
}

func DownloadStream(client *youtube.Client, video *youtube.Video, format *youtube.Format, filepath string) {
	stream, _, err := client.GetStream(video, format)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	file, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		panic(err)
	}
}
