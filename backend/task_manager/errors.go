package task_manager

import "errors"

var (
	// ErrTaskAlreadyRunning 任务已在运行
	ErrTaskAlreadyRunning = errors.New("任务已在运行")

	// ErrTaskNotRunning 任务未运行
	ErrTaskNotRunning = errors.New("任务未运行")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("配置无效")

	// ErrWorkspaceNotFound 工作区未找到
	ErrWorkspaceNotFound = errors.New("工作区未找到")

	// ErrNoFilesToProcess 没有文件需要处理
	ErrNoFilesToProcess = errors.New("没有文件需要处理")
)
