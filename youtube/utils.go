package main

import (
	"regexp"

	"github.com/kkdai/youtube/v2"
)

func isPlaylist(url string) bool {
	re := regexp.MustCompile(`list=([^&]+)`)
	match := re.FindStringSubmatch(url)
	return len(match) > 1
}

func getBestThumbnail(thumbnails youtube.Thumbnails) *youtube.Thumbnail {
	var bestThumbnail *youtube.Thumbnail

	var height = uint(0)

	for _, tn := range thumbnails {

		if tn.Height > height {
			height = tn.Height
			bestThumbnail = &tn
		}
	}

	return bestThumbnail
}
