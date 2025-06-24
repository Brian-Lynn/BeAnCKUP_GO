package manifest_manager

import "errors"

var (
	// ErrNoCurrentManifest 没有当前清单
	ErrNoCurrentManifest = errors.New("没有当前清单")

	// ErrManifestNotFound 清单未找到
	ErrManifestNotFound = errors.New("清单未找到")

	// ErrInvalidManifest 无效的清单
	ErrInvalidManifest = errors.New("无效的清单")

	// ErrManifestCorrupted 清单已损坏
	ErrManifestCorrupted = errors.New("清单已损坏")
)
