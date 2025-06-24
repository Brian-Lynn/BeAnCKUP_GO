package worker

import "errors"

var (
	// ErrPoolAlreadyRunning 工作池已在运行
	ErrPoolAlreadyRunning = errors.New("工作池已在运行")

	// ErrPoolNotRunning 工作池未运行
	ErrPoolNotRunning = errors.New("工作池未运行")

	// ErrQueueFull 任务队列已满
	ErrQueueFull = errors.New("任务队列已满")

	// ErrInvalidWorkerCount 无效的工作协程数量
	ErrInvalidWorkerCount = errors.New("无效的工作协程数量")
)
