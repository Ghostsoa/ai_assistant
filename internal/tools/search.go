package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteSearchCode 搜索代码
func ExecuteSearchCode(args map[string]interface{}) string {
	query := args["query"].(string)
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// 构建grep命令
	grepArgs := []string{"-rn", query, path}

	// 添加文件过滤
	if pattern, ok := args["file_pattern"].(string); ok && pattern != "" {
		grepArgs = append(grepArgs, "--include="+pattern)
	}

	// 正则模式
	if isRegex, ok := args["is_regex"].(bool); ok && isRegex {
		grepArgs = append(grepArgs, "-E")
	}

	cmd := exec.Command("grep", grepArgs...)
	output, err := cmd.CombinedOutput()

	if err != nil && len(output) == 0 {
		return fmt.Sprintf("[✗] 未找到匹配: %s", query)
	}

	// 解析结果，限制行数
	lines := strings.Split(string(output), "\n")
	if len(lines) > 50 {
		lines = lines[:50]
		return fmt.Sprintf("[搜索] 结果（前50条）:\n```\n%s```\n... 还有更多结果", strings.Join(lines, "\n"))
	}

	return fmt.Sprintf("[搜索] 结果:\n```\n%s```", string(output))
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
