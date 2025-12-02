package tools

import (
	"ai_assistant/internal/state"
	"fmt"
)

// ExecuteTerminalManage 管理终端槽位
func ExecuteTerminalManage(args map[string]interface{}, sm *state.Manager) string {
	action := args["action"].(string)

	switch action {
	case "open":
		slotID := args["slot"].(string)
		machineID := args["machine"].(string)
		return openTerminalSlot(slotID, machineID, sm)

	case "close":
		slotID := args["slot"].(string)
		return closeTerminalSlot(slotID, sm)

	case "switch":
		slotID := args["slot"].(string)
		machineID := args["machine"].(string)
		return switchTerminalSlot(slotID, machineID, sm)

	case "status":
		return getTerminalStatus(sm)

	default:
		return fmt.Sprintf("[✗] 未知操作: %s", action)
	}
}

// openTerminalSlot 打开终端槽位
func openTerminalSlot(slotID, machineID string, sm *state.Manager) string {
	if slotID != "slot1" && slotID != "slot2" {
		return "[✗] 无效的槽位ID，只能是 slot1 或 slot2"
	}

	if slotID == "slot1" {
		return "[✗] Slot1 是主槽位，无法手动打开/关闭"
	}

	// 检查机器是否存在
	machine := sm.GetMachine(machineID)
	if machine == nil {
		return fmt.Sprintf("[✗] 机器不存在: %s", machineID)
	}

	// 打开slot
	err := sm.OpenTerminalSlot(slotID, machineID)
	if err != nil {
		return fmt.Sprintf("[✗] 打开槽位失败: %v", err)
	}

	return fmt.Sprintf("[✓] Slot2 已打开: %s (%s)", machineID, machine.Description)
}

// closeTerminalSlot 关闭终端槽位
func closeTerminalSlot(slotID string, sm *state.Manager) string {
	if slotID != "slot2" {
		return "[✗] 只能关闭 slot2"
	}

	err := sm.CloseTerminalSlot(slotID)
	if err != nil {
		return fmt.Sprintf("[✗] 关闭槽位失败: %v", err)
	}

	return "[✓] Slot2 已关闭"
}

// switchTerminalSlot 切换槽位到另一个机器
func switchTerminalSlot(slotID, machineID string, sm *state.Manager) string {
	if slotID != "slot1" && slotID != "slot2" {
		return "[✗] 无效的槽位ID"
	}

	// 检查机器是否存在
	machine := sm.GetMachine(machineID)
	if machine == nil {
		return fmt.Sprintf("[✗] 机器不存在: %s", machineID)
	}

	err := sm.SwitchTerminalSlot(slotID, machineID)
	if err != nil {
		return fmt.Sprintf("[✗] 切换失败: %v", err)
	}

	return fmt.Sprintf("[✓] %s 已切换到: %s (%s)", slotID, machineID, machine.Description)
}

// getTerminalStatus 获取终端状态
func getTerminalStatus(sm *state.Manager) string {
	return sm.GetTerminalStatus()
}
