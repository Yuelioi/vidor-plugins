package main

func TestMain() {
	proxy := "http://127.0.0.1:10809"
	t := New("", "", proxy)

	links := []string{}

	for _, link := range links {
		id, err := extractIDFromURL(link)
		if err != nil {
			println(id, err)
			continue
		}

		tweet, err := t.scraper.GetTweet(id)
		if err != nil {
			println(id, err)
			continue

		}

		t.Download(tweet)
	}
}
