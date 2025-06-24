package indexer

import (
	"beanckup/backend/types"
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

// Manager 索引器实现
type Manager struct {
	scanProgress float64
	scanStatus   string
	scannedFiles int
	totalFiles   int
}

// NewManager 创建新的索引器
func NewManager() *Manager {
	return &Manager{
		scanProgress: 0.0,
		scanStatus:   "就绪",
	}
}

// ScanWorkspace 扫描工作区，返回当前所有文件的元数据
func (i *Manager) ScanWorkspace(workspacePath string) (map[string]*types.FileInfo, error) {
	currentFiles := make(map[string]*types.FileInfo)

	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略 .beanckup 目录及其内容
		if strings.Contains(path, ".beanckup") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 只处理文件，跳过目录
		if info.IsDir() {
			return nil
		}

		// 忽略标准交付包文件（工作区名-S开头的.7z文件）
		fileName := info.Name()
		if strings.HasSuffix(fileName, ".7z") && strings.Contains(fileName, "-S") {
			// 检查是否符合标准交付包命名格式：工作区名-SYYYY-EXXX-YYYYMMDD-HHMMSS.7z
			parts := strings.Split(fileName, "-")
			if len(parts) >= 4 {
				// 检查是否有S开头的部分和E开头的部分
				hasS := false
				hasE := false
				for _, part := range parts {
					if strings.HasPrefix(part, "S") && len(part) == 5 {
						hasS = true
					}
					if strings.HasPrefix(part, "E") && len(part) == 4 {
						hasE = true
					}
				}
				if hasS && hasE {
					return nil // 跳过标准交付包
				}
			}
		}

		// 创建文件信息
		fileInfo := &types.FileInfo{
			Path:        path,
			Name:        info.Name(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			ContentHash: "",                    // 暂时为空，后续计算
			Status:      types.StatusUnchanged, // 默认为未变更
		}

		// 使用绝对路径作为key
		currentFiles[path] = fileInfo

		return nil
	})

	return currentFiles, err
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

// GetScanProgress 获取扫描进度
func (m *Manager) GetScanProgress() float64 {
	return m.scanProgress
}

// GetScanStatus 获取扫描状态
func (m *Manager) GetScanStatus() string {
	return m.scanStatus
}
