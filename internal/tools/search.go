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

	currentMachine := sm.GetCurrentMachineID()

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
	if currentMachine != "local" {
		output, err = sm.ExecuteOnAgent(currentMachine, grepCmd)
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

// ExecuteFindSymbol 查找符号定义
func ExecuteFindSymbol(args map[string]interface{}) string {
	symbol := args["symbol"].(string)
	symbolType := ""
	if st, ok := args["symbol_type"].(string); ok {
		symbolType = st
	}

	var patterns []string
	switch symbolType {
	case "function":
		patterns = []string{fmt.Sprintf("func.*%s", symbol), fmt.Sprintf("def %s", symbol)}
	case "type":
		patterns = []string{fmt.Sprintf("type %s", symbol), fmt.Sprintf("class %s", symbol)}
	case "var":
		patterns = []string{fmt.Sprintf("var %s", symbol), fmt.Sprintf("%s :?=", symbol)}
	case "const":
		patterns = []string{fmt.Sprintf("const %s", symbol)}
	default:
		// 搜索所有类型
		patterns = []string{
			fmt.Sprintf("func.*%s", symbol),
			fmt.Sprintf("type %s", symbol),
			fmt.Sprintf("class %s", symbol),
			fmt.Sprintf("def %s", symbol),
		}
	}

	var allResults []string
	for _, pattern := range patterns {
		cmd := exec.Command("grep", "-rn", "-E", pattern, ".")
		output, _ := cmd.CombinedOutput()
		if len(output) > 0 {
			allResults = append(allResults, string(output))
		}
	}

	if len(allResults) == 0 {
		return fmt.Sprintf("[✗] 未找到符号定义: %s", symbol)
	}

	combined := strings.Join(allResults, "")
	lines := strings.Split(combined, "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}

	return fmt.Sprintf("[搜索] 找到符号 '%s' 的定义:\n```\n%s```", symbol, strings.Join(lines, "\n"))
}
