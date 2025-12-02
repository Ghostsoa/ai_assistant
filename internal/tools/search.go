package tools

import (
	"ai_assistant/internal/state"
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteSearchCode 搜索代码（支持远程）
func ExecuteSearchCode(args map[string]interface{}, sm *state.Manager) string {
	query := args["query"].(string)
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// 获取目标机器（由executor注入）
	targetMachine, _ := args["_target_machine"].(string)
	if targetMachine == "" {
		targetMachine = "local"
	}

	// 构建grep命令
	var grepCmd string
	if pattern, ok := args["file_pattern"].(string); ok && pattern != "" {
		grepCmd = fmt.Sprintf("grep -rn '%s' '%s' --include='%s' 2>/dev/null || echo '[未找到匹配]'", query, path, pattern)
	} else {
		grepCmd = fmt.Sprintf("grep -rn '%s' '%s' 2>/dev/null || echo '[未找到匹配]'", query, path)
	}

	var output string
	var err error

	// 远程或本地执行
	if targetMachine != "local" {
		output, err = sm.ExecuteOnAgent(targetMachine, grepCmd)
	} else {
		cmd := exec.Command("sh", "-c", grepCmd)
		outBytes, _ := cmd.CombinedOutput()
		output = string(outBytes)
	}

	if err != nil || strings.Contains(output, "[未找到匹配]") {
		return fmt.Sprintf("[✗] 未找到匹配: %s", query)
	}

	// 解析结果，限制行数
	lines := strings.Split(output, "\n")
	if len(lines) > 50 {
		lines = lines[:50]
		return fmt.Sprintf("[搜索] 结果（前50条）:\n```\n%s```\n... 还有更多结果", strings.Join(lines, "\n"))
	}

	return fmt.Sprintf("[搜索] 结果:\n```\n%s```", output)
}

// ExecuteFindSymbol 已删除
// 使用 file_operation({action: "search", query: "func.*symbolName"}) 替代
