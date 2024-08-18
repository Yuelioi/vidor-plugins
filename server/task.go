package main

import "sync"

const (
	Unknown  int = iota // 未知状态
	Queue               // 工作中状态
	Working             // 工作中状态
	Stopped             // 停止状态
	Finished            // 完成状态
)

type Task struct {
	Id       string
	State    int
	StopChan chan struct{}
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{}
}

func NewTask(id string) *Task {
	return &Task{
		Id:    id,
		State: Queue,
	}
}

type TaskQueue struct {
	Queue []*Task
	mu    sync.Mutex
}

func (tq *TaskQueue) AddTask(task *Task) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.Queue = append(tq.Queue, task)
}

func (tq *TaskQueue) SearchTask(id string) *Task {
	for _, tk := range tq.Queue {
		if tk.Id == id {
			return tk
		}
	}
	return nil
}

func (tq *TaskQueue) RemoveTask(id string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	newQueue := tq.Queue[:0]
	for _, tk := range tq.Queue {
		if tk.Id != id {
			newQueue = append(newQueue, tk)
		}
	}
	tq.Queue = newQueue
}
