package tools

import (
	"encoding/json"
	"fmt"
	"strings"

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
	case "web_search":
		return ExecuteWebSearch(args)
	case "sync":
		return ExecuteSync(args, e.StateManager)
	case "terminal_manage":
		return ExecuteTerminalManage(args, e.StateManager)
	default:
		return fmt.Sprintf("[✗] 未知工具: %s", toolCall.Function.Name)
	}
}

// executeFileOperation 执行文件操作（统一入口，支持machine参数）
func (e *ExecutorSimplified) executeFileOperation(toolCallID string, args map[string]interface{}) string {
	action, ok := args["action"].(string)
	if !ok {
		return "[✗] 缺少action参数"
	}

	// 确定目标机器
	var targetMachine string
	if machineID, ok := args["machine"].(string); ok && machineID != "" {
		targetMachine = machineID
	} else {
		// 使用slot1的机器
		slot1Machine := e.StateManager.GetSlot1Machine()
		if slot1Machine != nil {
			targetMachine = slot1Machine.ID
		} else {
			targetMachine = "local"
		}
	}

	// 将targetMachine注入到args中供后续函数使用
	args["_target_machine"] = targetMachine

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
		var args map[string]interface{}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		command, _ := args["command"].(string)
		return !isReadOnlyCommand(command)
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

// isReadOnlyCommand 检查命令是否为只读命令（白名单）
func isReadOnlyCommand(command string) bool {
	// 提取命令的第一个单词（忽略 LC_ALL=C 等前缀）
	cmd := strings.TrimSpace(command)

	// 移除常见的环境变量前缀
	cmd = strings.TrimPrefix(cmd, "LC_ALL=C ")
	cmd = strings.TrimPrefix(cmd, "LANG=C ")
	cmd = strings.TrimSpace(cmd)

	// 提取第一个单词
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	baseCmd := parts[0]

	// 白名单：只读查询命令
	readOnlyCommands := []string{
		"ls", "ll", "dir", // 列目录
		"pwd", "cd", // 目录操作
		"cat", "head", "tail", "less", "more", // 查看文件
		"grep", "find", "locate", // 搜索
		"echo", "printf", // 输出
		"whoami", "id", "groups", // 用户信息
		"hostname", "uname", // 系统信息
		"date", "uptime", // 时间信息
		"ps", "top", "htop", // 进程信息
		"df", "du", // 磁盘信息
		"free", "vmstat", // 内存信息
		"netstat", "ss", "ip", // 网络信息
		"systemctl status", "service status", // 服务状态
		"git status", "git log", "git diff", "git show", // Git只读
		"which", "whereis", "type", // 命令查找
		"file", "stat", // 文件信息
		"wc", "sort", "uniq", // 文本处理
	}

	for _, safeCmd := range readOnlyCommands {
		if baseCmd == safeCmd || strings.HasPrefix(cmd, safeCmd+" ") {
			return true
		}
	}

	return false
}
