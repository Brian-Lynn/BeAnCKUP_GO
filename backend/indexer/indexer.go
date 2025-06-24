package indexer

import (
	"beanckup/backend/types"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Indexer 索引器接口
type Indexer interface {
	// 扫描工作区，找出需要处理的文件
	ScanWorkspace(workspacePath string) (map[string]*types.FileInfo, error)

	// 快速扫描：对比元数据，找出嫌疑人
	QuickScan(currentFiles map[string]*types.FileInfo, previousManifest *types.Manifest) map[string]*types.FileInfo

	// 获取扫描进度
	GetScanProgress() float64

	// 获取扫描状态
	GetScanStatus() string
}

// Manager 索引器，负责所有与文件发现、状态对比相关的任务
type Manager struct {
	// 可以在这里添加一些配置，例如要忽略的目录模式等
}

// NewManager 创建一个新的索引器实例
func NewManager() *Manager {
	return &Manager{}
}

// ProgressCallback 是一个回调函数类型，用于在扫描过程中报告进度
type ProgressCallback func(processedCount int, totalCount int)

// ScanWorkspace 扫描指定路径下的所有文件，并返回它们的信息
// 这个实现是健壮的，可以处理符号链接并提供进度报告
func (m *Manager) ScanWorkspace(workspacePath string, callback ProgressCallback) (map[string]*types.FileInfo, error) {
	log.Printf("Indexer: Starting to scan workspace: %s", workspacePath)
	files := make(map[string]*types.FileInfo)

	// 第一步：先遍历一次，统计文件总数，用于计算进度
	var totalFiles int
	filepath.WalkDir(workspacePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// 忽略 .beanckup 目录和符号链接
		if strings.Contains(path, ".beanckup") || d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			totalFiles++
		}
		return nil
	})
	log.Printf("Indexer: Found %d total files to process.", totalFiles)

	// 第二步：正式遍历，收集文件信息
	var processedFiles int
	err := filepath.WalkDir(workspacePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Indexer: Error accessing path %s: %v", path, err)
			return err
		}

		// 忽略 .beanckup 目录和符号链接
		if strings.Contains(path, ".beanckup") || d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				log.Printf("Indexer: Skipping directory: %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil // 是目录，直接返回
		}

		info, err := d.Info()
		if err != nil {
			log.Printf("Indexer: Could not get FileInfo for %s: %v", path, err)
			return nil // 跳过无法获取信息的文件
		}

		// 创建 FileInfo 对象
		fileInfo := &types.FileInfo{
			Path:    path,
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Status:  types.StatusUnchanged, // 默认状态
		}

		files[path] = fileInfo
		processedFiles++

		// 每处理100个文件报告一次进度，避免过于频繁的回调
		if callback != nil && processedFiles%100 == 0 {
			callback(processedFiles, totalFiles)
		}

		return nil
	})

	if err != nil {
		log.Printf("Indexer: A critical error occurred during scanning: %v", err)
		return nil, err
	}

	// 确保最后一次进度被报告
	if callback != nil {
		callback(processedFiles, totalFiles)
	}

	log.Printf("Indexer: Finished scanning. Processed %d files.", processedFiles)
	return files, nil
}

// QuickScan 快速扫描：对比元数据，找出嫌疑人
func (i *Manager) QuickScan(currentFiles map[string]*types.FileInfo, previousManifest *types.Manifest) map[string]*types.FileInfo {
	suspects := make(map[string]*types.FileInfo)

	// 如果是首次扫描（没有之前的清单），所有文件都是新增的
	if previousManifest == nil || len(previousManifest.Files) == 0 {
		for filePath, currentFile := range currentFiles {
			currentFile.Status = types.StatusNew
			suspects[filePath] = currentFile
		}
		return suspects
	}

	// 遍历当前文件，与上次清单对比
	for filePath, currentFile := range currentFiles {
		// 在清单中查找该文件
		previousFile, exists := previousManifest.Files[filePath]

		if !exists {
			// 文件不存在于上次清单中，标记为新增
			currentFile.Status = types.StatusNew
			suspects[filePath] = currentFile
		} else {
			// 文件存在，检查元数据是否有变更
			if i.hasMetadataChanged(currentFile, previousFile) {
				currentFile.Status = types.StatusModified
				suspects[filePath] = currentFile
			}
			// 如果元数据完全匹配，保持StatusUnchanged，不加入嫌疑人列表
		}
	}

	// 检查删除的文件（可选实现）
	// 遍历上次清单，查找在当前文件中不存在的文件
	for filePath, previousFile := range previousManifest.Files {
		if _, exists := currentFiles[filePath]; !exists {
			// 文件在当前扫描中不存在，标记为删除
			deletedFile := &types.FileInfo{
				Path:        filePath,
				Name:        previousFile.Name,
				Size:        previousFile.Size,
				ModTime:     previousFile.ModTime,
				ContentHash: previousFile.ContentHash,
				Status:      types.StatusDeleted,
			}
			suspects[filePath] = deletedFile
		}
	}

	return suspects
}

// hasMetadataChanged 检查文件元数据是否有变更
func (i *Manager) hasMetadataChanged(current, previous *types.FileInfo) bool {
	// 比较文件大小
	if current.Size != previous.Size {
		return true
	}

	// 比较修改时间
	if !current.ModTime.Equal(previous.ModTime) {
		return true
	}

	// 比较文件名（虽然路径相同，但文件名可能不同）
	if current.Name != previous.Name {
		return true
	}

	return false
}

// CompareWithManifest 对比当前文件和上次清单，找出变更（保持向后兼容）
func (i *Manager) CompareWithManifest(currentFiles map[string]*types.FileInfo, previousManifest *types.Manifest) map[string]*types.FileInfo {
	// 使用新的快速扫描方法
	return i.QuickScan(currentFiles, previousManifest)
}

// GetScanProgress 获取扫描进度 (兼容旧版，后续会废弃)
func (m *Manager) GetScanProgress() float64 {
	return 0
}

// GetScanStatus 获取扫描状态 (兼容旧版，后续会废弃)
func (m *Manager) GetScanStatus() string {
	return " "
}
