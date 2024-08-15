package main

import (
	"github.com/Yuelioi/bilibili/pkg/bpi"
)

type Client struct {
	BpiService *bpi.BpiService
}

func NewClient() *Client {
	return &Client{
		BpiService: bpi.New(),
	}
}
