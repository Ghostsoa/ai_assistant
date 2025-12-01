package tools

import (
	"fmt"
	"os/exec"
	"time"

	"ai_assistant/internal/process"
)

// ExecuteRunCommand 执行命令
func ExecuteRunCommand(args map[string]interface{}, pm *process.Manager) string {
	command := args["command"].(string)
	interactive := args["interactive"].(bool)

	if interactive {
		processID, err := pm.StartProcess(command)
		if err != nil {
			return fmt.Sprintf("[✗] 启动失败: %v", err)
		}
		return fmt.Sprintf("[✓] 进程已启动\n进程ID: %s\n命令: %s", processID, command)
	}

	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s\n[返回码: %d]", string(output), cmd.ProcessState.ExitCode())
	}
	return fmt.Sprintf("%s\n[返回码: 0]", string(output))
}

// ExecuteSendInput 向进程发送输入
func ExecuteSendInput(args map[string]interface{}, pm *process.Manager) string {
	processID := args["process_id"].(string)
	input := args["input"].(string)

	if err := pm.SendInput(processID, input); err != nil {
		return fmt.Sprintf("[✗] %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	output, status, err := pm.GetOutput(processID)
	if err != nil {
		return fmt.Sprintf("[✗] %v", err)
	}

	if output != "" {
		return fmt.Sprintf("[输出] 进程响应:\n```\n%s```\n状态: %s", output, status)
	}
	return fmt.Sprintf("[✓] 输入已发送（暂无输出）\n状态: %s", status)
}

// ExecuteGetOutput 获取进程输出
func ExecuteGetOutput(args map[string]interface{}, pm *process.Manager) string {
	processID := args["process_id"].(string)

	output, status, err := pm.GetOutput(processID)
	if err != nil {
		return fmt.Sprintf("[✗] %v", err)
	}

	if output == "" {
		return fmt.Sprintf("[i] 暂无输出\n状态: %s", status)
	}
	return fmt.Sprintf("[输出] 进程输出:\n```\n%s```\n状态: %s", output, status)
}

// ExecuteKillProcess 终止进程
func ExecuteKillProcess(args map[string]interface{}, pm *process.Manager) string {
	processID := args["process_id"].(string)

	if err := pm.KillProcess(processID); err != nil {
		return fmt.Sprintf("[✗] %v", err)
	}
	return fmt.Sprintf("[✓] 进程已终止: %s", processID)
}
