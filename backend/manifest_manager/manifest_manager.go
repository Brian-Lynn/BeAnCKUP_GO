package manifest_manager

import (
	"beanckup/backend/types"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	manifestDir  = ".beanckup"
	manifestFile = "manifest.json"
)

// Manager 负责清单文件的读取和写入
type Manager struct{}

// NewManager 创建一个新的清单管理器
func NewManager() *Manager {
	return &Manager{}
}

// getManifestPath 返回清单文件的标准绝对路径
func (m *Manager) getManifestPath(workspacePath string) string {
	return filepath.Join(workspacePath, manifestDir, manifestFile)
}

// LoadLatestManifest 从工作区加载最新的清单文件
// 如果清单不存在或损坏，则返回一个新的空清单，不返回错误
func (m *Manager) LoadLatestManifest(workspacePath string) (*types.Manifest, error) {
	manifestPath := m.getManifestPath(workspacePath)
	log.Printf("ManifestManager: Attempting to load manifest from %s", manifestPath)

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		log.Println("ManifestManager: Manifest file not found. Creating a new empty manifest.")
		// 文件不存在，是首次备份，返回一个空的清单对象
		return &types.Manifest{
			Version:    "1.0",
			CreatedAt:  time.Now(),
			Files:      make(map[string]*types.FileInfo),
			Dirs:       make(map[string]*types.DirInfo),
			HashToFile: make(map[string]string),
		}, nil
	}

	// 文件存在，读取并解析
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Printf("ManifestManager: Error reading manifest file: %v. Returning a new empty manifest.", err)
		// 读取失败也返回新清单，保证程序健灸性
		return &types.Manifest{
			Version:    "1.0",
			CreatedAt:  time.Now(),
			Files:      make(map[string]*types.FileInfo),
			Dirs:       make(map[string]*types.DirInfo),
			HashToFile: make(map[string]string),
		}, nil
	}

	var manifest types.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		log.Printf("ManifestManager: Error unmarshalling manifest JSON: %v. Returning a new empty manifest.", err)
		// 解析失败也返回新清单
		return &types.Manifest{
			Version:    "1.0",
			CreatedAt:  time.Now(),
			Files:      make(map[string]*types.FileInfo),
			Dirs:       make(map[string]*types.DirInfo),
			HashToFile: make(map[string]string),
		}, nil
	}

	// 为了后续处理方便，确保map不是nil
	if manifest.Files == nil {
		manifest.Files = make(map[string]*types.FileInfo)
	}
	if manifest.Dirs == nil {
		manifest.Dirs = make(map[string]*types.DirInfo)
	}
	if manifest.HashToFile == nil {
		manifest.HashToFile = make(map[string]string)
	}

	log.Printf("ManifestManager: Successfully loaded manifest created at %s", manifest.CreatedAt)
	return &manifest, nil
}

// SaveManifest 将清单文件保存到工作区和交付路径
func (m *Manager) SaveManifest(workspacePath, deliveryPath string, manifest *types.Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		log.Printf("ManifestManager: Error marshalling manifest to JSON: %v", err)
		return err
	}

	// 1. 保存到工作区
	workspaceManifestPath := m.getManifestPath(workspacePath)
	// 确保 .beanckup 目录存在
	if err := os.MkdirAll(filepath.Dir(workspaceManifestPath), 0755); err != nil {
		log.Printf("ManifestManager: Error creating .beanckup directory in workspace: %v", err)
		return err
	}
	if err := ioutil.WriteFile(workspaceManifestPath, data, 0644); err != nil {
		log.Printf("ManifestManager: Error writing manifest to workspace: %v", err)
		return err
	}
	log.Printf("ManifestManager: Successfully saved manifest to %s", workspaceManifestPath)

	// 2. 如果提供了交付路径，也保存一份到交付路径
	if deliveryPath != "" {
		deliveryManifestPath := filepath.Join(deliveryPath, manifestFile)
		if err := ioutil.WriteFile(deliveryManifestPath, data, 0644); err != nil {
			log.Printf("ManifestManager: Error writing manifest to delivery path: %v", err)
			return err
		}
		log.Printf("ManifestManager: Successfully saved manifest to %s", deliveryManifestPath)
	}

	return nil
}
