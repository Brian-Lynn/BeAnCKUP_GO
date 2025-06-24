package task_manager

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"beanckup/backend/indexer"
	"beanckup/backend/manifest_manager"
	"beanckup/backend/packager"
	"beanckup/backend/resource_manager"
	"beanckup/backend/types"
	"beanckup/backend/worker"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Manager 任务管理器
type Manager struct {
	indexer         *indexer.Manager
	manifestManager *manifest_manager.Manager
	resourceManager *resource_manager.Manager
	workerManager   *worker.Manager
	packager        *packager.Manager
}

// NewManager 创建新的任务管理器
func NewManager(
	indexer *indexer.Manager,
	manifestManager *manifest_manager.Manager,
	resourceManager *resource_manager.Manager,
	workerManager *worker.Manager,
	packager *packager.Manager,
) *Manager {
	return &Manager{
		indexer:         indexer,
		manifestManager: manifestManager,
		resourceManager: resourceManager,
		workerManager:   workerManager,
		packager:        packager,
	}
}

// StartBackupPreparation 开始备份准备
func (m *Manager) StartBackupPreparation(workspacePath string, maxPackageSizeGB, maxTotalSizeGB float64) ([]*types.Episode, error) {
	// 1. 加载最新的清单文件
	previousManifest, err := m.manifestManager.LoadLatestManifest(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("加载清单失败: %w", err)
	}

	// 2. 扫描当前工作区
	currentFiles, err := m.indexer.ScanWorkspace(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("扫描工作区失败: %w", err)
	}

	// 3. 快速扫描：对比元数据，找出嫌疑人
	suspectFiles := m.indexer.QuickScan(currentFiles, previousManifest)

	// 4. 预估交付包
	episodes, err := m.estimateEpisodes(suspectFiles, maxPackageSizeGB, maxTotalSizeGB)
	if err != nil {
		return nil, fmt.Errorf("预估交付包失败: %w", err)
	}

	return episodes, nil
}

// StartBackupExecution 开始备份执行
func (m *Manager) StartBackupExecution(workspacePath, deliveryPath string, maxPackageSizeGB, maxTotalSizeGB float64, password string, ctx context.Context) (*BackupResult, error) {
	// 1. 加载最新的清单文件
	previousManifest, err := m.manifestManager.LoadLatestManifest(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("加载清单失败: %w", err)
	}

	// 2. 扫描当前工作区
	currentFiles, err := m.indexer.ScanWorkspace(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("扫描工作区失败: %w", err)
	}

	// 3. 快速扫描：对比元数据，找出嫌疑人
	suspectFiles := m.indexer.QuickScan(currentFiles, previousManifest)

	// 4. 并发哈希计算和去重
	workerCount := m.resourceManager.GetOptimalWorkerCount()
	workerResult, err := m.workerManager.StartWorkerPool(suspectFiles, workerCount, previousManifest)
	if err != nil {
		return nil, fmt.Errorf("工作池处理失败: %w", err)
	}

	// 5. 分包计算
	fileGroups, err := m.groupFilesForPackaging(workerResult.FilesToPack, maxPackageSizeGB, maxTotalSizeGB)
	if err != nil {
		return nil, fmt.Errorf("分包计算失败: %w", err)
	}

	// 6. 创建新清单
	newManifest, err := m.createNewManifest(workspacePath, currentFiles, previousManifest)
	if err != nil {
		return nil, fmt.Errorf("创建新清单失败: %w", err)
	}

	// 7. 顺序打包
	episodes, err := m.packageFiles(fileGroups, deliveryPath, workspacePath, password, ctx, newManifest)
	if err != nil {
		return nil, fmt.Errorf("文件打包失败: %w", err)
	}

	// 8. 保存清单到工作区和交付路径
	err = m.manifestManager.SaveManifest(workspacePath, deliveryPath, newManifest)
	if err != nil {
		return nil, fmt.Errorf("保存清单失败: %w", err)
	}

	return &BackupResult{
		Episodes: episodes,
		Manifest: newManifest,
	}, nil
}

// BackupResult 备份结果
type BackupResult struct {
	Episodes []*types.Episode `json:"episodes"`
	Manifest *types.Manifest  `json:"manifest"`
}

// groupFilesForPackaging 将文件分组用于打包
func (m *Manager) groupFilesForPackaging(filesToPack []*types.FileInfo, maxPackageSizeGB, maxTotalSizeGB float64) ([][]*types.FileInfo, error) {
	if len(filesToPack) == 0 {
		return [][]*types.FileInfo{}, nil
	}

	// 转换GB为字节
	maxPackageSize := int64(maxPackageSizeGB * 1024 * 1024 * 1024)
	maxTotalSize := int64(maxTotalSizeGB * 1024 * 1024 * 1024)

	// 计算总大小
	totalSize := int64(0)
	for _, file := range filesToPack {
		totalSize += file.Size
	}

	// 检查是否超过总量限制
	if totalSize > maxTotalSize {
		return nil, fmt.Errorf("总大小 %.2f GB 超过限制 %.2f GB", float64(totalSize)/1024/1024/1024, maxTotalSizeGB)
	}

	// 分组文件
	var groups [][]*types.FileInfo
	var currentGroup []*types.FileInfo
	var currentGroupSize int64

	for _, file := range filesToPack {
		// 如果当前组加上这个文件会超过限制，创建新组
		if currentGroupSize+file.Size > maxPackageSize && len(currentGroup) > 0 {
			groups = append(groups, currentGroup)
			currentGroup = []*types.FileInfo{}
			currentGroupSize = 0
		}

		currentGroup = append(currentGroup, file)
		currentGroupSize += file.Size
	}

	// 添加最后一组
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups, nil
}

// packageFiles 打包文件
func (m *Manager) packageFiles(fileGroups [][]*types.FileInfo, deliveryPath, workspacePath string, password string, ctx context.Context, manifest *types.Manifest) ([]*types.Episode, error) {
	episodes := make([]*types.Episode, 0, len(fileGroups))
	currentTime := time.Now()

	// 获取工作区名称
	workspaceName := filepath.Base(workspacePath)
	// 生成系列ID（年份+序号）
	seriesID := fmt.Sprintf("S%04d", currentTime.Year())

	for i, group := range fileGroups {
		// 计算组大小
		groupSize := int64(0)
		for _, file := range group {
			groupSize += file.Size
		}

		// 创建Episode
		episode := &types.Episode{
			ID:            fmt.Sprintf("E%03d", i+1),
			Name:          fmt.Sprintf("Episode-%03d", i+1),
			SeriesID:      seriesID,
			CreatedAt:     currentTime,
			Status:        "打包中",
			PackagePath:   "",
			FileCount:     len(group),
			TotalSize:     groupSize,
			EstimatedSize: groupSize,
		}

		// 生成包文件名：工作区名-Sxxx-Exxx-时间戳.7z
		packageFileName := fmt.Sprintf("%s-%s-%s-%s.7z",
			workspaceName,
			seriesID,
			episode.ID,
			currentTime.Format("20060102-150405"))
		packagePath := filepath.Join(deliveryPath, packageFileName)
		episode.PackagePath = packagePath

		// 打包文件 - 使用新的packager接口
		err := m.packager.CreateArchiveWith7zr(group, packagePath, workspacePath, password)
		if err != nil {
			episode.Status = "失败"
			// 推送状态更新事件
			runtime.EventsEmit(ctx, "episode-status-update", map[string]interface{}{
				"episodeId": episode.ID,
				"status":    "失败",
			})
			return nil, fmt.Errorf("打包Episode %s失败: %w", episode.ID, err)
		}

		// 更新状态
		episode.Status = "已完成"
		// 推送状态更新事件
		runtime.EventsEmit(ctx, "episode-status-update", map[string]interface{}{
			"episodeId": episode.ID,
			"status":    "已完成",
		})

		// 获取实际包大小
		if fileInfo, err := os.Stat(packagePath); err == nil {
			episode.TotalSize = fileInfo.Size()
		}

		episodes = append(episodes, episode)
	}

	return episodes, nil
}

// createNewManifest 创建新的清单
func (m *Manager) createNewManifest(workspacePath string, currentFiles map[string]*types.FileInfo, previousManifest *types.Manifest) (*types.Manifest, error) {
	// 创建新的清单
	newManifest := &types.Manifest{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		SeriesID:    fmt.Sprintf("S%04d", time.Now().Year()),
		EpisodeID:   "",
		Files:       make(map[string]*types.FileInfo),
		Directories: make(map[string]*types.DirInfo),
		Metadata:    make(map[string]interface{}),
		HashToFile:  make(map[string]string),
	}

	// 复制之前的HashToFile映射
	if previousManifest != nil && previousManifest.HashToFile != nil {
		for hash, filePath := range previousManifest.HashToFile {
			newManifest.HashToFile[hash] = filePath
		}
	}

	// 添加当前文件
	for filePath, fileInfo := range currentFiles {
		newManifest.Files[filePath] = fileInfo
		// 如果有哈希值，添加到HashToFile映射
		if fileInfo.ContentHash != "" {
			newManifest.HashToFile[fileInfo.ContentHash] = filePath
		}
	}

	return newManifest, nil
}

// estimateEpisodes 预估交付包
func (m *Manager) estimateEpisodes(suspectFiles map[string]*types.FileInfo, maxPackageSizeGB, maxTotalSizeGB float64) ([]*types.Episode, error) {
	if len(suspectFiles) == 0 {
		return []*types.Episode{}, nil
	}

	// 转换GB为字节
	maxPackageSize := int64(maxPackageSizeGB * 1024 * 1024 * 1024)
	maxTotalSize := int64(maxTotalSizeGB * 1024 * 1024 * 1024)

	// 计算总大小
	totalSize := int64(0)
	for _, file := range suspectFiles {
		if file.Status != types.StatusDeleted {
			totalSize += file.Size
		}
	}

	// 检查是否超过总量限制
	if totalSize > maxTotalSize {
		return nil, fmt.Errorf("总大小 %.2f GB 超过限制 %.2f GB", float64(totalSize)/1024/1024/1024, maxTotalSizeGB)
	}

	// 预估包数量
	estimatedPackageCount := int(math.Ceil(float64(totalSize) / float64(maxPackageSize)))
	if estimatedPackageCount == 0 {
		estimatedPackageCount = 1
	}

	// 生成包列表
	episodes := make([]*types.Episode, 0, estimatedPackageCount)
	currentTime := time.Now()

	// 将map转换为slice
	var files []*types.FileInfo
	for _, file := range suspectFiles {
		if file.Status != types.StatusDeleted {
			files = append(files, file)
		}
	}

	for i := 0; i < estimatedPackageCount; i++ {
		// 计算当前包的大小
		startIndex := i * len(files) / estimatedPackageCount
		endIndex := (i + 1) * len(files) / estimatedPackageCount
		if endIndex > len(files) {
			endIndex = len(files)
		}

		packageSize := int64(0)
		fileCount := 0
		for j := startIndex; j < endIndex; j++ {
			packageSize += files[j].Size
			fileCount++
		}

		// 创建Episode
		episode := &types.Episode{
			ID:            fmt.Sprintf("E%03d", i+1),
			Name:          fmt.Sprintf("Episode-%03d", i+1),
			SeriesID:      fmt.Sprintf("S%04d", currentTime.Year()),
			CreatedAt:     currentTime,
			Status:        "未交付",
			PackagePath:   "",
			FileCount:     fileCount,
			TotalSize:     packageSize,
			EstimatedSize: packageSize,
		}

		episodes = append(episodes, episode)
	}

	return episodes, nil
}

// GetTaskStatus 获取任务状态
func (m *Manager) GetTaskStatus() *types.TaskStatus {
	return &types.TaskStatus{
		IsRunning:      false,
		CurrentPhase:   "就绪",
		Progress:       0,
		ProcessedFiles: 0,
		TotalFiles:     0,
		ProcessedSize:  0,
		TotalSize:      0,
		Speed:          0,
		ElapsedTime:    0,
		EstimatedTime:  0,
	}
}
