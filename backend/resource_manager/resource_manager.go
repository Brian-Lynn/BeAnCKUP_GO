package resource_manager

import (
	"math"
	"runtime"

	"beanckup/backend/types"

	"github.com/shirou/gopsutil/v3/mem"
)

// ResourceManager 资源管理器接口
type ResourceManager interface {
	// 获取系统资源信息
	GetResourceInfo() types.ResourceInfo

	// 计算小文件处理阈值
	CalculateThreshold() int64

	// 刷新资源信息
	Refresh() error
}

// Manager 资源管理器
type Manager struct {
	lastResourceInfo *types.ResourceInfo
}

// NewManager 创建新的资源管理器
func NewManager() *Manager {
	return &Manager{}
}

// CalculateThreshold 计算动态阈值
func (m *Manager) CalculateThreshold() (int64, error) {
	// 获取系统内存信息
	vmstat, err := mem.VirtualMemory()
	if err != nil {
		return 64 * 1024 * 1024, err // 默认64MB
	}

	totalMemory := int64(vmstat.Total)
	availableMemory := int64(vmstat.Available)

	// 计算内存使用率
	memoryUsage := float64(totalMemory-availableMemory) / float64(totalMemory) * 100

	// 动态算法：最终阈值 = Max( Min(50%可用内存, 15%总内存, 4GB硬性上限), 64MB保底值 )
	threshold1 := int64(float64(availableMemory) * 0.5) // 50%可用内存
	threshold2 := int64(float64(totalMemory) * 0.15)    // 15%总内存
	threshold3 := int64(4 * 1024 * 1024 * 1024)         // 4GB硬性上限

	// 取最小值
	minThreshold := min(threshold1, threshold2, threshold3)

	// 与保底值比较，取最大值
	finalThreshold := max(minThreshold, 64*1024*1024) // 64MB保底值

	// 更新资源信息
	m.lastResourceInfo = &types.ResourceInfo{
		TotalMemory:     totalMemory,
		AvailableMemory: availableMemory,
		MemoryUsage:     memoryUsage,
		Threshold:       finalThreshold,
	}

	return finalThreshold, nil
}

// GetResourceInfo 获取资源信息
func (m *Manager) GetResourceInfo() *types.ResourceInfo {
	if m.lastResourceInfo == nil {
		// 如果没有缓存的信息，重新计算
		threshold, _ := m.CalculateThreshold()
		if m.lastResourceInfo == nil {
			// 如果还是nil，创建一个默认值
			m.lastResourceInfo = &types.ResourceInfo{
				TotalMemory:     16 * 1024 * 1024 * 1024, // 16GB
				AvailableMemory: 8 * 1024 * 1024 * 1024,  // 8GB
				MemoryUsage:     50.0,                    // 50%
				Threshold:       threshold,
			}
		}
	}
	return m.lastResourceInfo
}

// GetOptimalWorkerCount 获取最优工作协程数
func (m *Manager) GetOptimalWorkerCount() int {
	cpuCount := runtime.NumCPU()

	// 根据CPU核心数确定工作协程数
	// 通常设置为CPU核心数的1-2倍
	workerCount := cpuCount * 2

	// 限制最大工作协程数
	if workerCount > 16 {
		workerCount = 16
	}

	// 确保至少有2个工作协程
	if workerCount < 2 {
		workerCount = 2
	}

	return workerCount
}

// min 返回最小值
func min(a, b, c int64) int64 {
	return int64(math.Min(float64(a), math.Min(float64(b), float64(c))))
}

// max 返回最大值
func max(a, b int64) int64 {
	return int64(math.Max(float64(a), float64(b)))
}
