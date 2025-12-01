package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	appconfig "ai_assistant/internal/config"
	"ai_assistant/internal/history"
)

// Session 会话信息
type Session struct {
	ID        string    `json:"id"`         // 会话ID（时间戳格式：20060102_150405）
	Title     string    `json:"title"`      // 会话标题
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 最后更新时间
	FilePath  string    `json:"file_path"`  // 历史记录文件路径
}

// Manager 会话管理器
type Manager struct {
	currentSession *Session
	sessionsDir    string
	indexFile      string
}

// NewManager 创建会话管理器
func NewManager() (*Manager, error) {
	sessionsDir := filepath.Join(appconfig.ConfigDir, "sessions")
	indexFile := filepath.Join(sessionsDir, "index.json")

	// 创建会话目录
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, err
	}

	m := &Manager{
		sessionsDir: sessionsDir,
		indexFile:   indexFile,
	}

	// 加载或创建默认会话
	sessions, err := m.loadIndex()
	if err != nil || len(sessions) == 0 {
		// 创建默认会话
		if err := m.NewSession("默认对话"); err != nil {
			return nil, err
		}
	} else {
		// 使用最新的会话
		m.currentSession = &sessions[len(sessions)-1]
	}

	return m, nil
}

// loadIndex 加载会话索引
func (m *Manager) loadIndex() ([]Session, error) {
	data, err := os.ReadFile(m.indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Session{}, nil
		}
		return nil, err
	}

	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

// saveIndex 保存会话索引
func (m *Manager) saveIndex(sessions []Session) error {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.indexFile, data, 0644)
}

// NewSession 创建新会话
func (m *Manager) NewSession(title string) error {
	now := time.Now()
	id := now.Format("20060102_150405")

	session := Session{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
		FilePath:  filepath.Join(m.sessionsDir, id+".json"),
	}

	// 创建空的历史记录文件
	emptyHistory := []history.Message{}
	data, err := json.MarshalIndent(emptyHistory, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(session.FilePath, data, 0644); err != nil {
		return err
	}

	// 保存到索引
	sessions, _ := m.loadIndex()
	sessions = append(sessions, session)
	if err := m.saveIndex(sessions); err != nil {
		return err
	}

	m.currentSession = &session
	return nil
}

// ListSessions 列出所有会话
func (m *Manager) ListSessions() ([]Session, error) {
	sessions, err := m.loadIndex()
	if err != nil {
		return nil, err
	}

	// 按更新时间倒序排序
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// SwitchSession 切换会话
func (m *Manager) SwitchSession(id string) error {
	sessions, err := m.loadIndex()
	if err != nil {
		return err
	}

	for i := range sessions {
		if sessions[i].ID == id {
			m.currentSession = &sessions[i]
			return nil
		}
	}

	return fmt.Errorf("会话不存在: %s", id)
}

// GetCurrentSession 获取当前会话
func (m *Manager) GetCurrentSession() *Session {
	return m.currentSession
}

// GetCurrentHistoryFile 获取当前会话的历史记录文件
func (m *Manager) GetCurrentHistoryFile() string {
	if m.currentSession == nil {
		return ""
	}
	return m.currentSession.FilePath
}

// UpdateSessionTime 更新会话时间
func (m *Manager) UpdateSessionTime() error {
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}

	m.currentSession.UpdatedAt = time.Now()

	sessions, err := m.loadIndex()
	if err != nil {
		return err
	}

	// 更新索引中的会话
	for i := range sessions {
		if sessions[i].ID == m.currentSession.ID {
			sessions[i].UpdatedAt = m.currentSession.UpdatedAt
			break
		}
	}

	return m.saveIndex(sessions)
}

// ClearCurrentSession 清空当前会话
func (m *Manager) ClearCurrentSession() error {
	if m.currentSession == nil {
		return fmt.Errorf("没有活动会话")
	}

	// 清空历史记录文件
	emptyHistory := []history.Message{}
	data, err := json.MarshalIndent(emptyHistory, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.currentSession.FilePath, data, 0644)
}

// DeleteSession 删除会话
func (m *Manager) DeleteSession(id string) error {
	sessions, err := m.loadIndex()
	if err != nil {
		return err
	}

	// 找到并删除会话
	found := false
	var newSessions []Session
	var deletedSession Session

	for _, s := range sessions {
		if s.ID == id {
			found = true
			deletedSession = s
		} else {
			newSessions = append(newSessions, s)
		}
	}

	if !found {
		return fmt.Errorf("会话不存在: %s", id)
	}

	// 删除历史记录文件
	os.Remove(deletedSession.FilePath)

	// 保存新索引
	if err := m.saveIndex(newSessions); err != nil {
		return err
	}

	// 如果删除的是当前会话，切换到最新的会话
	if m.currentSession != nil && m.currentSession.ID == id {
		if len(newSessions) > 0 {
			m.currentSession = &newSessions[len(newSessions)-1]
		} else {
			// 创建新的默认会话
			return m.NewSession("默认对话")
		}
	}

	return nil
}

// RenameSession 重命名会话
func (m *Manager) RenameSession(id, newTitle string) error {
	sessions, err := m.loadIndex()
	if err != nil {
		return err
	}

	found := false
	for i := range sessions {
		if sessions[i].ID == id {
			sessions[i].Title = newTitle
			found = true

			// 如果是当前会话，也更新
			if m.currentSession != nil && m.currentSession.ID == id {
				m.currentSession.Title = newTitle
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("会话不存在: %s", id)
	}

	return m.saveIndex(sessions)
}
