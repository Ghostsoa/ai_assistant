package tools

import (
	"fmt"
	"os/exec"
	"strings"
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
	outputStr := string(output)

	// 截取过长的输出
	outputStr = truncateOutput(outputStr)

	if err != nil {
		return fmt.Sprintf("%s\n[返回码: %d]", outputStr, cmd.ProcessState.ExitCode())
	}
	return fmt.Sprintf("%s\n[返回码: 0]", outputStr)
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
		// 截取过长的输出
		output = truncateOutput(output)
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

	// 截取过长的输出
	output = truncateOutput(output)
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

// truncateOutput 截取过长的输出
func truncateOutput(output string) string {
	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	// 如果少于30行，直接返回
	const maxLines = 30
	if totalLines <= maxLines {
		return output
	}

	// 截取前10行和后10行
	const headLines = 10
	const tailLines = 10

	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[totalLines-tailLines:], "\n")

	return fmt.Sprintf("%s\n\n... [已截断 %d 行输出] ...\n\n%s",
		head, totalLines-headLines-tailLines, tail)
}
