package tools

import (
	"encoding/json"
	"fmt"

	"ai_assistant/internal/backup"
	"ai_assistant/internal/process"

	"github.com/sashabaranov/go-openai"
)

// Executor 工具执行器
type Executor struct {
	ProcessManager *process.Manager
	BackupManager  *backup.Manager
}

// NewExecutor 创建工具执行器
func NewExecutor(pm *process.Manager, bm *backup.Manager) *Executor {
	return &Executor{
		ProcessManager: pm,
		BackupManager:  bm,
	}
}

// Execute 执行工具
func (e *Executor) Execute(toolCall openai.ToolCall) string {
	var args map[string]interface{}
	json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

	switch toolCall.Function.Name {
	// 文件操作
	case "read_file":
		return ExecuteReadFile(args)
	case "edit_file":
		return ExecuteEditFile(toolCall.ID, args, e.BackupManager)
	case "rename_symbol":
		return ExecuteRenameSymbol(toolCall.ID, args, e.BackupManager)
	case "delete_file":
		return ExecuteDeleteFile(toolCall.ID, args, e.BackupManager)
	// 命令执行
	case "run_command":
		return ExecuteRunCommand(args, e.ProcessManager)
	case "send_input":
		return ExecuteSendInput(args, e.ProcessManager)
	case "get_output":
		return ExecuteGetOutput(args, e.ProcessManager)
	case "kill_process":
		return ExecuteKillProcess(args, e.ProcessManager)
	// 代码搜索
	case "search_code":
		return ExecuteSearchCode(args)
	case "find_symbol":
		return ExecuteFindSymbol(args)
	// 项目分析
	case "list_directory":
		return ExecuteListDirectory(args)
	case "get_project_structure":
		return ExecuteGetProjectStructure(args)
	case "get_file_stats":
		return ExecuteGetFileStats(args)
	// Git工具
	case "git_status":
		return ExecuteGitStatus(args)
	case "git_diff":
		return ExecuteGitDiff(args)
	case "git_commit":
		return ExecuteGitCommit(args)
	default:
		return fmt.Sprintf("[✗] 未知工具: %s", toolCall.Function.Name)
	}
}

// NeedsImmediateApproval 是否需要立即批准
func NeedsImmediateApproval(functionName string) bool {
	// 只有不可撤销的执行类操作需要立即批准
	dangerousTools := []string{
		"run_command",
		"send_input",
		"kill_process",
		"git_commit", // Git提交也不可撤销
	}
	for _, tool := range dangerousTools {
		if tool == functionName {
			return true
		}
	}
	return false
}

// IsModifyOperation 是否为修改操作
func IsModifyOperation(functionName string) bool {
	// 可撤销的修改操作
	modifyTools := []string{
		"edit_file",
		"rename_symbol",
		"delete_file",
	}
	for _, tool := range modifyTools {
		if tool == functionName {
			return true
		}
	}
	return false
}
