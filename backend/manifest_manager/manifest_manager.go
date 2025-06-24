package manifest_manager

import (
	"beanckup/backend/types"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manager 清单管理器
type Manager struct{}

// NewManager 创建新的清单管理器
func NewManager() *Manager {
	return &Manager{}
}

// getManifestPath 返回一个工作区内清单文件的唯一、绝对路径
func getManifestPath(workspacePath string) string {
	return filepath.Join(workspacePath, ".beanckup", "manifest.json")
}

// LoadLatestManifest 加载最新的清单文件
func (m *Manager) LoadLatestManifest(workspacePath string) (*types.Manifest, error) {
	manifestPath := getManifestPath(workspacePath)

	// 检查文件是否存在
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// 文件不存在，是首次备份。返回一个全新的、空的清单对象，不要报错。
		return &types.Manifest{
			Version:    "1.0",
			CreatedAt:  time.Now(),
			Files:      make(map[string]*types.FileInfo),
			HashToFile: make(map[string]string),
		}, nil
	}

	// 文件存在，读取并解析
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("读取清单文件失败: %w", err)
	}

	var manifest types.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		// 如果JSON解析失败，可能文件已损坏。返回一个空的清单并记录错误，让用户可以重新开始。
		fmt.Fprintf(os.Stderr, "警告: 清单文件 %s 已损坏，将作为首次备份处理。错误: %v\n", manifestPath, err)
		return &types.Manifest{
			Version:    "1.0",
			CreatedAt:  time.Now(),
			Files:      make(map[string]*types.FileInfo),
			HashToFile: make(map[string]string),
		}, nil
	}

	return &manifest, nil
}

// SaveManifest 将最终清单保存到工作区内的.beanckup目录和交付路径下
func (m *Manager) SaveManifest(workspacePath, deliveryPath string, manifest *types.Manifest) error {
	// 确保manifest对象不为空
	if manifest == nil {
		return fmt.Errorf("不能保存空的清单")
	}

	// 更新时间戳
	manifest.CreatedAt = time.Now()

	// 序列化清单为JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化清单失败: %w", err)
	}

	// 1. 保存到工作区的.beanckup目录 (这是唯一的真相来源)
	beanckupDir := filepath.Join(workspacePath, ".beanckup")
	if err := os.MkdirAll(beanckupDir, 0755); err != nil {
		return fmt.Errorf("创建.beanckup目录失败: %w", err)
	}
	workspaceManifestPath := getManifestPath(workspacePath)
	if err := os.WriteFile(workspaceManifestPath, data, 0644); err != nil {
		return fmt.Errorf("写入工作区清单失败: %w", err)
	}

	// 2. 为了便携性，复制一份到交付路径
	if deliveryPath != "" {
		deliveryManifestPath := filepath.Join(deliveryPath, fmt.Sprintf("%s-manifest.json", manifest.SeriesID))
		if err := os.WriteFile(deliveryManifestPath, data, 0644); err != nil {
			// 即使这里失败，也不应算作致命错误，只打印警告
			fmt.Fprintf(os.Stderr, "警告: 复制清单到交付路径失败: %v\n", err)
		}
	}

	return nil
}
