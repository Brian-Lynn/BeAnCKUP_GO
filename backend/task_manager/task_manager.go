package task_manager

import (
	"beanckup/backend/indexer"
	"beanckup/backend/manifest_manager"
	"beanckup/backend/tree_builder"
	"beanckup/backend/types"
	"context"
	"fmt"
	"log"
)

// Manager 任务管理器，是所有业务逻辑的编排器
type Manager struct {
	indexer         *indexer.Manager
	manifestManager *manifest_manager.Manager
}

// NewManager 创建一个新的任务管理器
func NewManager() *Manager {
	log.Println("Task Manager initialized.")
	return &Manager{
		indexer:         indexer.NewManager(),
		manifestManager: manifest_manager.NewManager(),
	}
}

// StartBackupPreparation 接收备份参数，进行预处理
func (m *Manager) StartBackupPreparation(workspacePath string, maxPackageSizeGB, maxTotalSizeGB float64) (*types.BackupPreparationResult, error) {
	log.Printf("Task Manager: Starting backup preparation for %s", workspacePath)

	// 1. 扫描当前工作区的所有文件
	// 注意：这里的进度回调暂时为nil，因为这个重量级操作的整体进度应该由task_manager在更高层面控制和报告
	currentFiles, err := m.indexer.ScanWorkspace(workspacePath, nil)
	if err != nil {
		log.Printf("Task Manager: Failed to scan workspace: %v", err)
		return nil, fmt.Errorf("扫描工作区失败: %w", err)
	}

	// 2. 加载上一次的清单
	previousManifest, err := m.manifestManager.LoadLatestManifest(workspacePath)
	if err != nil {
		// 根据我们的设计，这个错误理论上不应该发生，但为了健壮性还是处理一下
		log.Printf("Task Manager: Failed to load previous manifest: %v", err)
		return nil, fmt.Errorf("加载旧备份记录失败: %w", err)
	}

	// 3. 对比新旧文件，找出所有变更
	changedFiles := m.indexer.QuickScan(currentFiles, previousManifest)
	log.Printf("Task Manager: Found %d changed files.", len(changedFiles))

	// 4. 根据变更预估分包
	episodes, changeInfo := m.estimateEpisodes(changedFiles, maxPackageSizeGB, maxTotalSizeGB)
	log.Printf("Task Manager: Estimated %d episodes. Changes: %d new, %d modified, %d deleted. Total size: %d bytes.", len(episodes), changeInfo.NewCount, changeInfo.ModifiedCount, changeInfo.DeletedCount, changeInfo.TotalSize)

	// 5. 根据变更构建UI文件树
	fileTree := tree_builder.BuildTreeFromChanges(changedFiles, workspacePath)
	log.Printf("Task Manager: Built file tree with %d root nodes.", len(fileTree))

	// 6. 将所有结果打包返回
	result := &types.BackupPreparationResult{
		Episodes:   episodes,
		FileTree:   fileTree,
		ChangeInfo: changeInfo,
	}

	log.Println("Task Manager: Backup preparation finished successfully.")
	return result, nil
}

// estimateEpisodes 根据变更文件和大小限制，预估需要生成的交付包
func (m *Manager) estimateEpisodes(changedFiles map[string]*types.FileInfo, maxPackageSizeGB, maxTotalSizeGB float64) ([]*types.Episode, struct {
	NewCount      int   `json:"newCount"`
	ModifiedCount int   `json:"modifiedCount"`
	DeletedCount  int   `json:"deletedCount"`
	TotalSize     int64 `json:"totalSize"`
}) {
	var filesToPack []*types.FileInfo
	var changeInfo struct {
		NewCount      int   `json:"newCount"`
		ModifiedCount int   `json:"modifiedCount"`
		DeletedCount  int   `json:"deletedCount"`
		TotalSize     int64 `json:"totalSize"`
	}

	for _, file := range changedFiles {
		switch file.Status {
		case types.StatusNew:
			changeInfo.NewCount++
			changeInfo.TotalSize += file.Size
			filesToPack = append(filesToPack, file)
		case types.StatusModified:
			changeInfo.ModifiedCount++
			changeInfo.TotalSize += file.Size
			filesToPack = append(filesToPack, file)
		case types.StatusDeleted:
			changeInfo.DeletedCount++
		}
	}

	if len(filesToPack) == 0 {
		return []*types.Episode{}, changeInfo
	}

	maxPackageSizeBytes := int64(maxPackageSizeGB * 1024 * 1024 * 1024)
	if maxPackageSizeBytes <= 0 {
		maxPackageSizeBytes = 2 * 1024 * 1024 * 1024 // 默认2GB
	}

	var episodes []*types.Episode
	var currentEpisodeFiles []*types.FileInfo
	var currentEpisodeSize int64
	episodeIndex := 1

	for _, file := range filesToPack {
		if currentEpisodeSize+file.Size > maxPackageSizeBytes && len(currentEpisodeFiles) > 0 {
			// 当前分集满了，创建它
			episodes = append(episodes, createEpisode(episodeIndex, currentEpisodeFiles, currentEpisodeSize))
			// 为下一个分集重置
			currentEpisodeFiles = nil
			currentEpisodeSize = 0
			episodeIndex++
		}
		currentEpisodeFiles = append(currentEpisodeFiles, file)
		currentEpisodeSize += file.Size
	}

	// 添加最后一个（或唯一一个）分集
	if len(currentEpisodeFiles) > 0 {
		episodes = append(episodes, createEpisode(episodeIndex, currentEpisodeFiles, currentEpisodeSize))
	}

	return episodes, changeInfo
}

// createEpisode 是一个辅助函数，用于创建一个新的分集对象
func createEpisode(index int, files []*types.FileInfo, size int64) *types.Episode {
	return &types.Episode{
		ID:            fmt.Sprintf("E%03d", index),
		Name:          fmt.Sprintf("Episode-%03d", index),
		Status:        "未交付",
		FileCount:     len(files),
		EstimatedSize: size,
	}
}

// StartBackupExecution 启动实际的备份流程
func (m *Manager) StartBackupExecution(workspacePath, deliveryPath string, maxPackageSizeGB, maxTotalSizeGB float64, password string, ctx context.Context) (interface{}, error) {
	log.Printf("Task Manager: Starting backup execution for %s to %s", workspacePath, deliveryPath)
	// TODO: 实现文件哈希、压缩、打包、生成新清单的逻辑
	// 1. 基于预处理结果，创建并发任务
	// 2. 使用 worker 池处理文件（哈希等）
	// 3. 使用 packager 打包成.7z文件
	// 4. 生成并保存最终的 manifest
	log.Println("Task Manager: Backup execution finished (stub).")
	return nil, nil
}
