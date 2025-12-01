package backup

import (
	"fmt"
	"os"
	"sync"
)

// OperationBackup 操作备份（用于撤销）
type OperationBackup struct {
	ToolCallID string
	Type       string // edit, rename, delete
	FilePath   string
	OldContent []byte
	EditCount  int // 对同一文件的修改次数
}

// Manager 备份管理器
type Manager struct {
	backups []OperationBackup
	mutex   sync.Mutex
}

// NewManager 创建备份管理器
func NewManager() *Manager {
	return &Manager{
		backups: []OperationBackup{},
	}
}

// AddBackup 添加备份
func (m *Manager) AddBackup(toolCallID, opType, filePath string, oldContent []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查是否已有该文件的备份
	hasBackup := false
	for i := range m.backups {
		if m.backups[i].FilePath == filePath {
			hasBackup = true
			m.backups[i].EditCount++
			if opType != "" {
				m.backups[i].Type = opType
			}
			break
		}
	}

	// 只在第一次修改时保存备份
	if !hasBackup {
		m.backups = append(m.backups, OperationBackup{
			ToolCallID: toolCallID,
			Type:       opType,
			FilePath:   filePath,
			OldContent: oldContent,
			EditCount:  1,
		})
	}
}

// UndoOperation 撤销操作
func (m *Manager) UndoOperation(toolCallID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, backup := range m.backups {
		if backup.ToolCallID == toolCallID {
			// 恢复文件
			switch backup.Type {
			case "edit", "rename":
				if err := os.WriteFile(backup.FilePath, backup.OldContent, 0644); err != nil {
					return fmt.Errorf("恢复文件失败: %v", err)
				}
			case "delete":
				if err := os.WriteFile(backup.FilePath, backup.OldContent, 0644); err != nil {
					return fmt.Errorf("恢复文件失败: %v", err)
				}
			}

			// 删除备份
			m.backups = append(m.backups[:i], m.backups[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("未找到备份")
}

// CommitAll 提交所有操作（清空备份）
func (m *Manager) CommitAll() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.backups = []OperationBackup{}
}

// GetBackups 获取所有备份
func (m *Manager) GetBackups() []OperationBackup {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.backups
}

// HasBackups 是否有备份
func (m *Manager) HasBackups() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.backups) > 0
}
