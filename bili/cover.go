package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-resty/resty/v2"
)

func downloadCover(url, filePath string) error {
	client := resty.New()

	response, err := client.R().
		Get(url)
	if err != nil {
		return err
	}

	if response.StatusCode() != 200 {
		return fmt.Errorf("failed to fetch the image: %s", response.Status())
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	return os.WriteFile(filePath, response.Body(), os.ModePerm)
}
