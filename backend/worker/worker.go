package worker

import (
	"beanckup/backend/types"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
)

// WorkerPool 工作池接口
type WorkerPool interface {
	// 启动工作池
	Start(workerCount int) error

	// 停止工作池
	Stop() error

	// 提交任务
	SubmitTask(task types.TaskUnit) error

	// 获取工作池状态
	GetStatus() PoolStatus
}

// PoolStatus 工作池状态
type PoolStatus struct {
	IsRunning      bool    `json:"is_running"`
	WorkerCount    int     `json:"worker_count"`
	ActiveWorkers  int     `json:"active_workers"`
	QueueSize      int     `json:"queue_size"`
	ProcessedTasks int     `json:"processed_tasks"`
	TotalTasks     int     `json:"total_tasks"`
	Progress       float64 `json:"progress"`
}

// Worker 工作协程接口
type Worker interface {
	// 启动工作池，专注哈希计算
	StartWorkerPool(suspectFiles map[string]*types.FileInfo, numWorkers int, previousManifest *types.Manifest) (*WorkerResult, error)
}

// Manager 工作协程管理器
type Manager struct {
	mu sync.Mutex
}

// TaskResult 任务结果
type TaskResult struct {
	TaskUnit    types.TaskUnit
	SHA256      string
	IsDuplicate bool
	Error       error
}

// NewManager 创建新的工作池
func NewManager() *Manager {
	return &Manager{}
}

// WorkerResult 工作池结果
type WorkerResult struct {
	FilesToPack    []*types.FileInfo // 需要物理备份的文件
	MetadataUpdate []*types.FileInfo // 仅需更新元数据的文件
	TotalProcessed int               // 总处理文件数
	TotalSize      int64             // 总大小
}

// StartWorkerPool 启动工作池，专注哈希计算
func (m *Manager) StartWorkerPool(suspectFiles map[string]*types.FileInfo, numWorkers int, previousManifest *types.Manifest) (*WorkerResult, error) {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU() * 2
		if numWorkers > 16 {
			numWorkers = 16
		}
	}

	if len(suspectFiles) == 0 {
		return &WorkerResult{
			FilesToPack:    []*types.FileInfo{},
			MetadataUpdate: []*types.FileInfo{},
			TotalProcessed: 0,
			TotalSize:      0,
		}, nil
	}

	// 将map转换为slice
	var allFiles []*types.FileInfo
	for _, file := range suspectFiles {
		// 跳过删除的文件，它们不需要计算哈希
		if file.Status != types.StatusDeleted {
			allFiles = append(allFiles, file)
		}
	}

	// 创建任务通道
	taskChan := make(chan *types.FileInfo, len(allFiles))
	resultChan := make(chan *HashResult, len(allFiles))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go m.worker(taskChan, resultChan, &wg)
	}

	// 发送任务到通道
	go func() {
		defer close(taskChan)
		for _, file := range allFiles {
			taskChan <- file
		}
	}()

	// 等待所有工作协程完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	result := &WorkerResult{
		FilesToPack:    make([]*types.FileInfo, 0),
		MetadataUpdate: make([]*types.FileInfo, 0),
	}

	for hashResult := range resultChan {
		if hashResult.Error != nil {
			// 记录错误但继续处理
			continue
		}

		// 更新文件的哈希值
		hashResult.File.ContentHash = hashResult.ContentHash

		// 检查是否为重复文件
		if previousManifest != nil && previousManifest.HashToFile != nil {
			if _, exists := previousManifest.HashToFile[hashResult.ContentHash]; exists {
				// 哈希已存在，仅需更新元数据
				result.MetadataUpdate = append(result.MetadataUpdate, hashResult.File)
			} else {
				// 哈希不存在，需要物理备份
				result.FilesToPack = append(result.FilesToPack, hashResult.File)
			}
		} else {
			// 没有之前的清单，所有文件都需要备份
			result.FilesToPack = append(result.FilesToPack, hashResult.File)
		}

		result.TotalProcessed++
		result.TotalSize += hashResult.File.Size
	}

	return result, nil
}

// HashResult 哈希计算结果
type HashResult struct {
	File        *types.FileInfo
	ContentHash string
	Error       error
}

// worker 工作协程函数
func (m *Manager) worker(taskChan <-chan *types.FileInfo, resultChan chan<- *HashResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range taskChan {
		result := &HashResult{
			File: file,
		}

		// 计算文件哈希
		contentHash, err := m.calculateFileHash(file.Path)
		if err != nil {
			result.Error = fmt.Errorf("计算哈希失败: %w", err)
			resultChan <- result
			continue
		}

		result.ContentHash = contentHash
		resultChan <- result
	}
}

// calculateFileHash 计算文件哈希
func (m *Manager) calculateFileHash(filePath string) (string, error) {
	hash := sha256.New()

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 流式读取并计算哈希
	buffer := make([]byte, 64*1024) // 64KB缓冲区
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("读取文件失败: %w", err)
		}
	}

	// 返回十六进制哈希值
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetOptimalWorkerCount 获取最优工作协程数
func (m *Manager) GetOptimalWorkerCount() int {
	cpuCount := runtime.NumCPU()
	workerCount := cpuCount * 2

	if workerCount > 16 {
		workerCount = 16
	}

	if workerCount < 2 {
		workerCount = 2
	}

	return workerCount
}
