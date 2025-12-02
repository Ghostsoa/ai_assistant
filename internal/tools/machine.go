package tools

import (
	"fmt"

	"ai_assistant/internal/state"
)

// ExecuteSwitchMachine 切换控制机
func ExecuteSwitchMachine(args map[string]interface{}, sm *state.Manager) string {
	machineID, ok := args["machine_id"].(string)

	// 不传参数：列出所有机器
	if !ok || machineID == "" {
		machines := sm.ListMachines()
		return fmt.Sprintf("【可用控制机】\n%s", machines)
	}

	// 切换机器
	if err := sm.SwitchMachine(machineID); err != nil {
		return fmt.Sprintf("[✗] 切换失败: %v", err)
	}

	return fmt.Sprintf("[✓] 已切换到控制机: %s", machineID)
}
