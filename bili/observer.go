package main

import pb "proto"

type Observer interface {
	OnProgress(progress *pb.DownloadProgress)
}

type DownloadObserver struct {
	stream pb.DownloadService_DownloadServer
}

func NewDownloadObserver(stream pb.DownloadService_DownloadServer) *DownloadObserver {
	return &DownloadObserver{
		stream: stream,
	}
}

func (d *DownloadObserver) OnProgress(progress *pb.DownloadProgress) {
	d.stream.Send(progress)
}

// subject.go
type Subject interface {
	RegisterObserver(observer Observer)
	RemoveObserver(observer Observer)
	NotifyObservers(progress *pb.DownloadProgress)
}

type DownloadSubject struct {
	observers []Observer
}

func NewDownloadSubject() *DownloadSubject {
	return &DownloadSubject{
		observers: make([]Observer, 0),
	}
}

func (d *DownloadSubject) RegisterObserver(observer Observer) {
	d.observers = append(d.observers, observer)
}

func (d *DownloadSubject) RemoveObserver(observer Observer) {
	for i, o := range d.observers {
		if o == observer {
			d.observers = append(d.observers[:i], d.observers[i+1:]...)
			return
		}
	}
}

func (d *DownloadSubject) NotifyObservers(progress *pb.DownloadProgress) {
	for _, observer := range d.observers {
		observer.OnProgress(progress)
	}
}
