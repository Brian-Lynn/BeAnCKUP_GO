package state_manager

import (
	"beanckup/backend/types"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateManager 状态管理器接口
type StateManager interface {
	// 保存会话状态
	SaveSessionState(state types.SessionState) error

	// 加载会话状态
	LoadSessionState(seriesID string) (*types.SessionState, error)

	// 清除会话状态
	ClearSessionState(seriesID string) error

	// 获取所有会话状态
	GetAllSessionStates() ([]types.SessionState, error)

	// 更新会话状态
	UpdateSessionState(seriesID string, updates map[string]interface{}) error
}

// Manager 状态管理器实现
type Manager struct {
	stateDir string
	mu       sync.RWMutex
}

// NewManager 创建新的状态管理器
func NewManager(stateDir string) *Manager {
	return &Manager{
		stateDir: stateDir,
	}
}

// SaveSessionState 保存会话状态
func (m *Manager) SaveSessionState(state types.SessionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保目录存在
	err := os.MkdirAll(m.stateDir, 0755)
	if err != nil {
		return err
	}

	// 更新时间戳
	state.LastUpdate = time.Now()

	// 序列化状态
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	statePath := filepath.Join(m.stateDir, state.SeriesID+".session.json")
	err = os.WriteFile(statePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// LoadSessionState 加载会话状态
func (m *Manager) LoadSessionState(seriesID string) (*types.SessionState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statePath := filepath.Join(m.stateDir, seriesID+".session.json")

	// 检查文件是否存在
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil, ErrSessionNotFound
	}

	// 读取状态文件
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state types.SessionState
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// ClearSessionState 清除会话状态
func (m *Manager) ClearSessionState(seriesID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	statePath := filepath.Join(m.stateDir, seriesID+".session.json")

	// 检查文件是否存在
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	// 删除状态文件
	err := os.Remove(statePath)
	if err != nil {
		return err
	}

	return nil
}

// GetAllSessionStates 获取所有会话状态
func (m *Manager) GetAllSessionStates() ([]types.SessionState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 确保目录存在
	if _, err := os.Stat(m.stateDir); os.IsNotExist(err) {
		return []types.SessionState{}, nil
	}

	// 读取目录中的所有.session.json文件
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		return nil, err
	}

	var states []types.SessionState

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !isSessionFile(entry.Name()) {
			continue
		}

		statePath := filepath.Join(m.stateDir, entry.Name())
		data, err := os.ReadFile(statePath)
		if err != nil {
			continue // 跳过无法读取的文件
		}

		var state types.SessionState
		err = json.Unmarshal(data, &state)
		if err != nil {
			continue // 跳过损坏的文件
		}

		states = append(states, state)
	}

	return states, nil
}

// UpdateSessionState 更新会话状态
func (m *Manager) UpdateSessionState(seriesID string, updates map[string]interface{}) error {
	// 加载当前状态
	state, err := m.LoadSessionState(seriesID)
	if err != nil {
		if err == ErrSessionNotFound {
			// 创建新状态
			state = &types.SessionState{
				SeriesID:   seriesID,
				LastUpdate: time.Now(),
				Status:     "初始化",
			}
		} else {
			return err
		}
	}

	// 应用更新
	if currentEpisode, ok := updates["current_episode"].(string); ok {
		state.CurrentEpisode = currentEpisode
	}
	if processedFiles, ok := updates["processed_files"].([]string); ok {
		state.ProcessedFiles = processedFiles
	}
	if pendingFiles, ok := updates["pending_files"].([]string); ok {
		state.PendingFiles = pendingFiles
	}
	if status, ok := updates["status"].(string); ok {
		state.Status = status
	}

	// 保存更新后的状态
	return m.SaveSessionState(*state)
}

// isSessionFile 检查是否为会话文件
func isSessionFile(filename string) bool {
	return len(filename) > 13 && filename[len(filename)-13:] == ".session.json"
}

// CleanupOldSessions 清理过期的会话状态
func (m *Manager) CleanupOldSessions(maxAge time.Duration) error {
	states, err := m.GetAllSessionStates()
	if err != nil {
		return err
	}

	now := time.Now()

	for _, state := range states {
		if now.Sub(state.LastUpdate) > maxAge {
			err := m.ClearSessionState(state.SeriesID)
			if err != nil {
				// 记录错误但继续清理其他会话
				continue
			}
		}
	}

	return nil
}
