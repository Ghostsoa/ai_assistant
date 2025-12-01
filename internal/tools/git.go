package tools

import (
	"fmt"
	"os/exec"
)

// ExecuteGitStatus Git状态
func ExecuteGitStatus(args map[string]interface{}) string {
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("[✗] Git错误: %v\n%s", err, string(output))
	}

	if len(output) == 0 {
		return "[✓] 工作目录干净，没有修改"
	}

	return fmt.Sprintf("[Git] 状态:\n```\n%s```", string(output))
}

// ExecuteGitDiff Git差异
func ExecuteGitDiff(args map[string]interface{}) string {
	var cmd *exec.Cmd
	if file, ok := args["file"].(string); ok && file != "" {
		cmd = exec.Command("git", "diff", file)
	} else {
		cmd = exec.Command("git", "diff")
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("[✗] Git错误: %v", err)
	}

	if len(output) == 0 {
		return "[i] 没有差异"
	}

	// 限制输出长度
	if len(output) > 5000 {
		output = output[:5000]
		return fmt.Sprintf("[Git] 差异（前5000字符）:\n```diff\n%s```\n... 输出被截断", string(output))
	}

	return fmt.Sprintf("[Git] 差异:\n```diff\n%s```", string(output))
}

// ExecuteGitCommit Git提交
func ExecuteGitCommit(args map[string]interface{}) string {
	message := args["message"].(string)

	var files []string
	if f, ok := args["files"].([]interface{}); ok {
		for _, file := range f {
			if fileStr, ok := file.(string); ok {
				files = append(files, fileStr)
			}
		}
	}

	// 添加文件
	if len(files) > 0 {
		for _, file := range files {
			cmd := exec.Command("git", "add", file)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Sprintf("[✗] Git add失败: %v\n%s", err, string(output))
			}
		}
	} else {
		// 添加所有修改
		cmd := exec.Command("git", "add", "-A")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Sprintf("[✗] Git add失败: %v\n%s", err, string(output))
		}
	}

	// 提交
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("[✗] Git commit失败: %v\n%s", err, string(output))
	}

	return fmt.Sprintf("[✓] Git提交成功:\n```\n%s```", string(output))
}
