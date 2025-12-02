package tools

import (
	"fmt"

	"ai_assistant/internal/state"
)

// ExecuteSwitchMachine 切换控制机
func ExecuteSwitchMachine(args map[string]interface{}, sm *state.Manager) string {
	machineID, ok := args["machine_id"].(string)

	if !ok || machineID == "" {
		return "[✗] 参数错误：需要指定 machine_id"
	}

	// 切换机器
	if err := sm.SwitchMachine(machineID); err != nil {
		return fmt.Sprintf("[✗] 切换失败: %v", err)
	}

	return fmt.Sprintf("[✓] 已切换到控制机: %s", machineID)
}
