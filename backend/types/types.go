package types

import (
	"time"
)

// FileInfo 文件信息
type FileInfo struct {
	Path        string     `json:"path"`
	Name        string     `json:"name"`
	Size        int64      `json:"size"`
	ModTime     time.Time  `json:"modTime"`
	ContentHash string     `json:"contentHash"`
	Status      FileStatus `json:"status"`
}

// FileStatus 文件状态
type FileStatus string

const (
	StatusUnchanged FileStatus = "unchanged"
	StatusNew       FileStatus = "new"
	StatusModified  FileStatus = "modified"
	StatusMoved     FileStatus = "moved"
	StatusDeleted   FileStatus = "deleted"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeSmallFile TaskType = "small_file"
	TaskTypeLargeFile TaskType = "large_file"
)

// ProcessingTask 处理任务
type ProcessingTask struct {
	FileInfo *FileInfo `json:"fileInfo"`
	Data     []byte    `json:"data,omitempty"` // 小文件的内容数据
	Path     string    `json:"path,omitempty"` // 大文件的路径
	Type     TaskType  `json:"type"`           // 任务类型
}

// ProcessingResult 处理结果
type ProcessingResult struct {
	SmallFileTasks []*ProcessingTask `json:"smallFileTasks"`
	LargeFileTasks []*ProcessingTask `json:"largeFileTasks"`
	TotalSize      int64             `json:"totalSize"`
	FileCount      int               `json:"fileCount"`
}

// WorkerResult 工作协程结果
type WorkerResult struct {
	Task        *ProcessingTask `json:"task"`
	ContentHash string          `json:"contentHash"`
	IsDuplicate bool            `json:"isDuplicate"`
	Error       error           `json:"error,omitempty"`
}

// TaskUnit 任务单元，用于在模块间传递数据
type TaskUnit struct {
	FileInfo   FileInfo `json:"file_info"`
	Content    []byte   `json:"content,omitempty"` // 小文件的内容
	IsInMemory bool     `json:"is_in_memory"`      // 是否已在内存中
}

// Episode 备份集
type Episode struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	SeriesID      string    `json:"seriesId"`
	CreatedAt     time.Time `json:"createdAt"`
	Status        string    `json:"status"`
	PackagePath   string    `json:"packagePath"`
	FileCount     int       `json:"fileCount"`
	TotalSize     int64     `json:"totalSize"`
	EstimatedSize int64     `json:"estimatedSize"`
}

// Series 备份系列
type Series struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"createdAt"`
	Status    string     `json:"status"`
	Episodes  []*Episode `json:"episodes"`
	FileCount int        `json:"fileCount"`
	TotalSize int64      `json:"totalSize"`
}

// BackupConfig 备份配置
type BackupConfig struct {
	MaxPackageSize      int64 `json:"maxPackageSize"`      // 单个包大小上限 (字节)
	MaxTotalSize        int64 `json:"maxTotalSize"`        // 本次任务总量上限 (字节)
	CompressionLevel    int   `json:"compressionLevel"`    // 压缩级别 (1-9)
	EnableDeduplication bool  `json:"enableDeduplication"` // 是否启用去重
}

// TaskStatus 任务状态
type TaskStatus struct {
	IsRunning      bool    `json:"isRunning"`
	CurrentPhase   string  `json:"currentPhase"`
	Progress       float64 `json:"progress"`
	ProcessedFiles int     `json:"processedFiles"`
	TotalFiles     int     `json:"totalFiles"`
	ProcessedSize  int64   `json:"processedSize"`
	TotalSize      int64   `json:"totalSize"`
	Speed          float64 `json:"speed"`
	ElapsedTime    int64   `json:"elapsedTime"`
	EstimatedTime  int64   `json:"estimatedTime"`
}

// ResourceInfo 系统资源信息
type ResourceInfo struct {
	TotalMemory     int64   `json:"totalMemory"`
	AvailableMemory int64   `json:"availableMemory"`
	MemoryUsage     float64 `json:"memoryUsage"`
	Threshold       int64   `json:"threshold"`
}

// Manifest 清单文件结构
type Manifest struct {
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"createdAt"`
	SeriesID    string                 `json:"seriesId"`
	EpisodeID   string                 `json:"episodeId"`
	Files       map[string]*FileInfo   `json:"files"`
	Directories map[string]*DirInfo    `json:"directories"`
	Metadata    map[string]interface{} `json:"metadata"`
	HashToFile  map[string]string      `json:"hashToFile"` // 哈希值到文件路径的映射，用于去重
}

// DirInfo 目录信息
type DirInfo struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	ModTime   time.Time `json:"modTime"`
	FileCount int       `json:"fileCount"`
	TotalSize int64     `json:"totalSize"`
}

// SessionState 会话状态（用于断点续传）
type SessionState struct {
	SeriesID       string    `json:"series_id"`
	LastUpdate     time.Time `json:"last_update"`
	CurrentEpisode string    `json:"current_episode"`
	ProcessedFiles []string  `json:"processed_files"`
	PendingFiles   []string  `json:"pending_files"`
	Status         string    `json:"status"`
}

// TreeNode 用于向前端传递目录树结构
type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Status   FileStatus  `json:"status,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}
