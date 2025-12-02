package tools

import (
	"encoding/json"
	"fmt"

	"ai_assistant/internal/backup"
	"ai_assistant/internal/process"
	"ai_assistant/internal/state"

	"github.com/sashabaranov/go-openai"
)

// ExecutorSimplified 简化版工具执行器
type ExecutorSimplified struct {
	ProcessManager *process.Manager
	BackupManager  *backup.Manager
	StateManager   *state.Manager
}

// NewExecutorSimplified 创建简化版执行器
func NewExecutorSimplified(pm *process.Manager, bm *backup.Manager, sm *state.Manager) *ExecutorSimplified {
	return &ExecutorSimplified{
		ProcessManager: pm,
		BackupManager:  bm,
		StateManager:   sm,
	}
}

// Execute 执行工具（简化版）
func (e *ExecutorSimplified) Execute(toolCall openai.ToolCall) string {
	var args map[string]interface{}
	json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

	switch toolCall.Function.Name {
	case "file_operation":
		return e.executeFileOperation(toolCall.ID, args)
	case "run_command":
		return ExecuteRunCommand(args, e.ProcessManager, e.StateManager)
	case "switch_machine":
		return ExecuteSwitchMachine(args, e.StateManager)
	case "web_search":
		return ExecuteWebSearch(args)
	default:
		return fmt.Sprintf("[✗] 未知工具: %s", toolCall.Function.Name)
	}
}

// executeFileOperation 执行文件操作（统一入口）
func (e *ExecutorSimplified) executeFileOperation(toolCallID string, args map[string]interface{}) string {
	action, ok := args["action"].(string)
	if !ok {
		return "[✗] 缺少action参数"
	}

	switch action {
	case "read":
		return ExecuteReadFile(args, e.StateManager)
	case "edit":
		return ExecuteEditFile(toolCallID, args, e.BackupManager, e.StateManager)
	case "rename":
		return ExecuteRenameSymbol(toolCallID, args, e.BackupManager)
	case "delete":
		return ExecuteDeleteFile(toolCallID, args, e.BackupManager)
	case "search":
		return ExecuteSearchCode(args, e.StateManager)
	default:
		return fmt.Sprintf("[✗] 未知文件操作: %s", action)
	}
}

// NeedsImmediateApproval 是否需要立即批准（简化版）
func (e *ExecutorSimplified) NeedsImmediateApproval(toolCall openai.ToolCall) bool {
	switch toolCall.Function.Name {
	case "run_command":
		return true
	case "file_operation":
		var args map[string]interface{}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		action, _ := args["action"].(string)
		// edit, rename, delete 需要批准
		return action == "edit" || action == "rename" || action == "delete"
	default:
		return false
	}
}
