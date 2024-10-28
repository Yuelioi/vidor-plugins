package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	resty "github.com/go-resty/resty/v2"
	twitterscraper "github.com/masa-finance/masa-twitter-scraper"
)

type Twitter struct {
	scraper  *twitterscraper.Scraper
	username string
	password string
}

func New(username, password string, proxy string) *Twitter {
	scraper := twitterscraper.New()
	err := scraper.SetProxy(proxy)
	if err != nil {
		panic(err)
	}

	t := &Twitter{
		scraper:  scraper,
		username: username,
		password: password,
	}

	t.login()

	return t

}

func (t *Twitter) login() {
	f, _ := os.Open("cookies.json")
	// deserialize from JSON
	var cookies []*http.Cookie
	json.NewDecoder(f).Decode(&cookies)
	// load cookies
	t.scraper.SetCookies(cookies)

	if !t.scraper.IsLoggedIn() {
		err := t.scraper.Login(t.username, t.password)
		if err != nil {
			panic(err)
		}

		cookies := t.scraper.GetCookies()
		// serialize to JSON
		js, _ := json.Marshal(cookies)
		// save to file
		f, _ := os.Create("cookies.json")
		f.Write(js)
	}
}

func (t *Twitter) Download(tweet *twitterscraper.Tweet) error {
	for _, v := range tweet.Videos {
		println("开始下载", v.ID)
		err := download(v.ID, v.Preview, ".jpg")
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return err
		}
		err = download(v.ID, v.URL, ".mp4")
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return err
		}
	}
	return nil
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

func download(id string, url string, ext string) error {

	httpClient, err := getProxyClient()
	if err != nil {
		return err
	}
	client := resty.NewWithClient(httpClient)

	resp, err := client.R().Get(url)
	if err != nil {
		return err
	}
	defer resp.RawBody().Close()

	f, err := os.Create(id + ext)
	if err != nil {
		return err
	}

	_, err = f.Write(resp.Body())

	return err
}

func extractIDFromURL(url string) (string, error) {
	// 定义一个正则表达式，用于匹配超过8位的数字
	re := regexp.MustCompile(`\b\d{9,}\b`)
	// 查找所有匹配项
	matches := re.FindAllString(url, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no ID found in the URL: %s", url)
	}
	// 假设我们只关心第一个匹配项
	id := matches[0]
	return id, nil
}
