package main

import pb "proto"

type Notification interface {
	OnUpdate(progress *pb.Task)
}

type DownloadNotification struct {
	stream pb.DownloadService_DownloadServer
}

func NewDownloadNotification(stream pb.DownloadService_DownloadServer) *DownloadNotification {
	return &DownloadNotification{
		stream: stream,
	}
}

func (d *DownloadNotification) OnUpdate(progress *pb.Task) {
	d.stream.Send(progress)
}
