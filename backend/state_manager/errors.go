package state_manager

import "errors"

var (
	// ErrSessionNotFound 会话未找到
	ErrSessionNotFound = errors.New("会话未找到")

	// ErrInvalidSessionState 无效的会话状态
	ErrInvalidSessionState = errors.New("无效的会话状态")

	// ErrSessionCorrupted 会话状态已损坏
	ErrSessionCorrupted = errors.New("会话状态已损坏")
)
