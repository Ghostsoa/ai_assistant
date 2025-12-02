package tools

import (
	"fmt"

	"ai_assistant/internal/process"
	"ai_assistant/internal/state"
)

// ExecuteRunCommand 执行命令（支持指定机器）
func ExecuteRunCommand(args map[string]interface{}, pm *process.Manager, sm *state.Manager) string {
	command := args["command"].(string)

	// 确定目标机器：优先使用参数指定的machine，否则使用slot1的机器
	var targetMachine string
	if machineID, ok := args["machine"].(string); ok && machineID != "" {
		targetMachine = machineID
	} else {
		// 使用slot1的机器
		slot1Machine := sm.GetSlot1Machine()
		if slot1Machine != nil {
			targetMachine = slot1Machine.ID
		} else {
			targetMachine = "local"
		}
	}

	var output string
	var err error

	// 根据机器类型路由
	if targetMachine == "local" {
		// 本地执行
		output, err = pm.ExecuteInPersistentShell(command)
	} else {
		// 远程寄生虫执行
		output, err = sm.ExecuteOnAgent(targetMachine, command)
	}

	if err != nil {
		// 错误也记录到终端
		sm.AppendTerminalOutput(targetMachine, command, fmt.Sprintf("[✗] %v", err))
		return "[✗] 执行失败，请查看【终端快照】"
	}

	// 更新终端快照
	sm.AppendTerminalOutput(targetMachine, command, output)

	// 返回简短提示
	return "[✓] 命令已执行，请查看【终端快照】"
}
