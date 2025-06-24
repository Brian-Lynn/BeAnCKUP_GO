package file_processor

import (
	"beanckup/backend/types"
	"io"
	"os"
	"path/filepath"
)

// Processor 文件处理器接口
type Processor interface {
	// 处理文件列表，根据大小进行分流
	ProcessFiles(changedFiles map[string]*types.FileInfo, threshold int64) (*types.ProcessingResult, error)
}

// Manager 文件处理器实现
type Manager struct{}

// NewManager 创建新的文件处理器
func NewManager() *Manager {
	return &Manager{}
}

// ProcessFiles 处理文件列表，根据大小进行分流
func (m *Manager) ProcessFiles(changedFiles map[string]*types.FileInfo, threshold int64) (*types.ProcessingResult, error) {
	result := &types.ProcessingResult{
		SmallFileTasks: make([]*types.ProcessingTask, 0),
		LargeFileTasks: make([]*types.ProcessingTask, 0),
	}

	// 遍历所有变更文件
	for filePath, fileInfo := range changedFiles {
		// 跳过删除的文件，它们不需要处理
		if fileInfo.Status == types.StatusDeleted {
			continue
		}

		// 根据文件大小决定处理策略
		if fileInfo.Size <= threshold {
			// 小文件：一次性读取到内存
			task, err := m.createSmallFileTask(filePath, fileInfo)
			if err != nil {
				// 如果读取失败，将其归类为大文件处理
				task = m.createLargeFileTask(filePath, fileInfo)
				result.LargeFileTasks = append(result.LargeFileTasks, task)
			} else {
				result.SmallFileTasks = append(result.SmallFileTasks, task)
			}
		} else {
			// 大文件：只保存路径，后续流式处理
			task := m.createLargeFileTask(filePath, fileInfo)
			result.LargeFileTasks = append(result.LargeFileTasks, task)
		}

		result.TotalSize += fileInfo.Size
		result.FileCount++
	}

	return result, nil
}

// createSmallFileTask 创建小文件任务
func (m *Manager) createSmallFileTask(filePath string, fileInfo *types.FileInfo) (*types.ProcessingTask, error) {
	// 读取文件内容
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取所有内容
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &types.ProcessingTask{
		FileInfo: fileInfo,
		Data:     content,
		Path:     "",
		Type:     types.TaskTypeSmallFile,
	}, nil
}

// createLargeFileTask 创建大文件任务
func (m *Manager) createLargeFileTask(filePath string, fileInfo *types.FileInfo) *types.ProcessingTask {
	return &types.ProcessingTask{
		FileInfo: fileInfo,
		Data:     nil,
		Path:     filePath,
		Type:     types.TaskTypeLargeFile,
	}
}

// GetFileExtension 获取文件扩展名
func (m *Manager) GetFileExtension(filePath string) string {
	return filepath.Ext(filePath)
}

// IsTextFile 判断是否为文本文件
func (m *Manager) IsTextFile(filePath string) bool {
	ext := m.GetFileExtension(filePath)
	textExtensions := map[string]bool{
		".txt":  true,
		".md":   true,
		".json": true,
		".xml":  true,
		".html": true,
		".css":  true,
		".js":   true,
		".py":   true,
		".go":   true,
		".java": true,
		".cpp":  true,
		".c":    true,
		".h":    true,
		".sql":  true,
		".sh":   true,
		".bat":  true,
		".ps1":  true,
		".yml":  true,
		".yaml": true,
		".toml": true,
		".ini":  true,
		".cfg":  true,
		".conf": true,
		".log":  true,
	}
	return textExtensions[ext]
}
