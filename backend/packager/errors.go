package packager

import "errors"

var (
	// ErrEpisodeNotFound Episode未找到
	ErrEpisodeNotFound = errors.New("Episode未找到")

	// ErrEpisodeAlreadyClosed Episode已关闭
	ErrEpisodeAlreadyClosed = errors.New("Episode已关闭")

	// ErrInvalidEpisodeSize 无效的Episode大小
	ErrInvalidEpisodeSize = errors.New("无效的Episode大小")

	// ErrOutputDirectoryNotFound 输出目录未找到
	ErrOutputDirectoryNotFound = errors.New("输出目录未找到")
)
