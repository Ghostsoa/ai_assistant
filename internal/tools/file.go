package tools

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"regexp"
	"strings"

	"ai_assistant/internal/backup"
	"ai_assistant/internal/state"
)

// 文件读取限制
const (
	MaxFileSize  = 10 * 1024 * 1024 // 10MB
	MaxReadLines = 1000             // 最多读取1000行
)

// ExecuteReadFile 读取文件（支持远程，带大小限制）
func ExecuteReadFile(args map[string]interface{}, sm *state.Manager) string {
	file := args["file"].(string)

	// 获取目标机器（由executor注入）
	targetMachine, _ := args["_target_machine"].(string)
	if targetMachine == "" {
		targetMachine = "local"
	}

	// 远程机器：通过寄生虫读取
	if targetMachine != "local" {
		// 使用base64编码传输，避免特殊字符问题
		cmd := fmt.Sprintf("cat '%s' 2>/dev/null | base64 -w 0 || base64 < '%s'", file, file)
		output, err := sm.ExecuteOnAgent(targetMachine, cmd)
		if err != nil {
			return fmt.Sprintf("[✗] 读取失败: %v", err)
		}

		// 清理所有空白字符
		output = strings.ReplaceAll(output, "\n", "")
		output = strings.ReplaceAll(output, "\r", "")
		output = strings.ReplaceAll(output, " ", "")
		output = strings.TrimSpace(output)

		// 解码base64
		decoded, err := base64.StdEncoding.DecodeString(output)
		if err != nil {
			// 如果解码失败，尝试直接读取
			directCmd := fmt.Sprintf("cat '%s' 2>/dev/null", file)
			directOutput, err2 := sm.ExecuteOnAgent(targetMachine, directCmd)
			if err2 != nil {
				return fmt.Sprintf("[✗] 读取失败: base64解码错误(%v), 直接读取也失败(%v)", err, err2)
			}
			content := []byte(directOutput)
			return processFileContent(file, content, args)
		}
		return processFileContent(file, decoded, args)
	}

	// 本地机器：直接读取
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Sprintf("[✗] 读取失败: %v", err)
	}
	return processFileContent(file, content, args)
}

// hasLineRange 检查是否指定了行号范围
func hasLineRange(args map[string]interface{}) bool {
	_, hasStart := args["start_line"]
	_, hasEnd := args["end_line"]
	return hasStart || hasEnd
}

// processFileContent 处理文件内容（提取公共逻辑，带大小检查）
func processFileContent(file string, content []byte, args map[string]interface{}) string {
	// 检查文件大小
	fileSize := int64(len(content))
	if fileSize > MaxFileSize {
		return fmt.Sprintf("[✗] 文件过大: %s (%.2f MB)\n"+
			"限制: 10 MB\n"+
			"提示: 请使用 run_command('head -n 100 %s') 查看部分内容",
			file,
			float64(fileSize)/(1024*1024),
			file)
	}

	// 分割成行
	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	// 检查行数限制
	if totalLines > MaxReadLines && !hasLineRange(args) {
		return fmt.Sprintf("[✗] 文件行数过多: %s (%d 行)\n"+
			"限制: %d 行\n"+
			"提示: 使用 start_line 和 end_line 参数分段读取\n"+
			"示例: {\"file\": \"%s\", \"start_line\": 1, \"end_line\": 100}",
			file,
			totalLines,
			MaxReadLines,
			file)
	}

	// 获取可选的行号范围参数
	var startLine, endLine int
	if start, ok := args["start_line"].(float64); ok {
		startLine = int(start)
	} else {
		startLine = 1
	}
	if end, ok := args["end_line"].(float64); ok {
		endLine = int(end)
	} else {
		endLine = totalLines
	}

	// 检查文件大小
	const maxLines = 1000

	// 如果没有指定行号范围，且文件过大
	if _, hasStart := args["start_line"]; !hasStart {
		if _, hasEnd := args["end_line"]; !hasEnd {
			if totalLines > maxLines {
				// 文件过大，返回摘要
				fileInfo, _ := os.Stat(file)
				fileSize := fileInfo.Size()
				sizeStr := formatFileSize(fileSize)

				return fmt.Sprintf("[文件] %s\n"+
					"[!] 文件过大，无法完整读取\n"+
					"文件大小: %s\n"+
					"总行数: %d 行\n\n"+
					"提示: 请使用 start_line 和 end_line 参数按行号范围读取\n"+
					"示例: {\"file\": \"%s\", \"start_line\": 1, \"end_line\": 100}",
					file, sizeStr, totalLines, file)
			}
		}
	}

	// 验证行号范围
	if startLine < 1 {
		startLine = 1
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine > endLine {
		return fmt.Sprintf("[✗] 行号范围无效: start_line(%d) > end_line(%d)", startLine, endLine)
	}

	// 提取指定范围的行
	selectedLines := lines[startLine-1 : endLine]
	result := strings.Join(selectedLines, "\n")

	// 格式化输出
	if startLine == 1 && endLine == totalLines {
		return fmt.Sprintf("[文件] %s (共 %d 行):\n```\n%s\n```", file, totalLines, result)
	} else {
		return fmt.Sprintf("[文件] %s (第 %d-%d 行，共 %d 行):\n```\n%s\n```",
			file, startLine, endLine, totalLines, result)
	}
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// ExecuteEditFile 编辑文件（支持远程）
func ExecuteEditFile(toolCallID string, args map[string]interface{}, bm *backup.Manager, sm *state.Manager) string {
	file := args["file"].(string)
	old := args["old"].(string)
	new := args["new"].(string)

	// 获取目标机器（由executor注入）
	targetMachine, _ := args["_target_machine"].(string)
	if targetMachine == "" {
		targetMachine = "local"
	}

	// 远程机器：先读取备份，然后用sed编辑
	if targetMachine != "local" {
		// 1. 先读取原文件内容（用于备份）
		// 使用 base64 -w 0 确保输出没有换行符
		readCmd := fmt.Sprintf("cat '%s' 2>/dev/null | base64 -w 0 || base64 < '%s'", file, file)
		b64Content, err := sm.ExecuteOnAgent(targetMachine, readCmd)
		if err != nil {
			return fmt.Sprintf("[✗] 读取文件失败: %v", err)
		}

		// 清理所有空白字符（换行、空格等）
		b64Content = strings.ReplaceAll(b64Content, "\n", "")
		b64Content = strings.ReplaceAll(b64Content, "\r", "")
		b64Content = strings.ReplaceAll(b64Content, " ", "")
		b64Content = strings.TrimSpace(b64Content)

		// 验证是否为有效的base64
		if b64Content == "" {
			return fmt.Sprintf("[✗] 文件为空或读取失败: %s", file)
		}

		oldContent, err := base64.StdEncoding.DecodeString(b64Content)
		if err != nil {
			// 如果base64解码失败，尝试直接读取
			directCmd := fmt.Sprintf("cat '%s' 2>/dev/null", file)
			directContent, err2 := sm.ExecuteOnAgent(targetMachine, directCmd)
			if err2 != nil {
				return fmt.Sprintf("[✗] 读取文件失败: base64解码错误(%v), 直接读取也失败(%v)", err, err2)
			}
			oldContent = []byte(directContent)
		}

		// 2. 检查匹配数量
		text := string(oldContent)
		count := strings.Count(text, old)

		if count == 0 {
			return "[✗] 未找到要替换的内容"
		}
		if count > 1 {
			return fmt.Sprintf("[✗] 找到%d处匹配，无法确定唯一位置", count)
		}

		// 3. 执行替换（在Go中完成，确保一致性）
		newText := strings.Replace(text, old, new, 1)

		// 4. 写回远程文件（使用base64避免特殊字符问题）
		newB64 := base64.StdEncoding.EncodeToString([]byte(newText))
		writeCmd := fmt.Sprintf("echo '%s' | base64 -d > '%s'", newB64, file)
		_, err = sm.ExecuteOnAgent(targetMachine, writeCmd)
		if err != nil {
			return fmt.Sprintf("[✗] 写入失败: %v", err)
		}

		// 5. 保存备份（和本地一样）
		bm.AddBackup(toolCallID, "edit", file+"@"+targetMachine, oldContent)

		return fmt.Sprintf("[✓] 文件已修改: %s (机器: %s, 等待用户确认)", file, targetMachine)
	}

	// 本地机器：原逻辑
	oldContent, err := os.ReadFile(file)
	if err != nil {
		return fmt.Sprintf("[✗] 读取文件失败: %v", err)
	}

	text := string(oldContent)
	count := strings.Count(text, old)

	if count == 0 {
		return "[✗] 未找到要替换的内容"
	}
	if count > 1 {
		return fmt.Sprintf("[✗] 找到%d处匹配，无法确定唯一位置", count)
	}

	// 执行替换
	newText := strings.Replace(text, old, new, 1)
	if err := os.WriteFile(file, []byte(newText), 0644); err != nil {
		return fmt.Sprintf("[✗] 写入失败: %v", err)
	}

	// 保存备份
	bm.AddBackup(toolCallID, "edit", file, oldContent)

	return fmt.Sprintf("[✓] 文件已修改: %s（等待用户确认）", file)
}

// ExecuteRenameSymbol 重命名符号
func ExecuteRenameSymbol(toolCallID string, args map[string]interface{}, bm *backup.Manager) string {
	file := args["file"].(string)
	oldSymbol := args["old_symbol"].(string)
	newSymbol := args["new_symbol"].(string)

	// 备份原文件
	oldContent, err := os.ReadFile(file)
	if err != nil {
		return fmt.Sprintf("[✗] 读取文件失败: %v", err)
	}

	var result string
	var newContent []byte

	if strings.HasSuffix(file, ".go") {
		// Go文件用AST
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return fmt.Sprintf("[✗] 解析Go文件失败: %v", err)
		}

		changeCount := 0
		ast.Inspect(node, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name == oldSymbol {
					ident.Name = newSymbol
					changeCount++
				}
			}
			return true
		})

		if changeCount == 0 {
			return fmt.Sprintf("[✗] 未找到符号: %s", oldSymbol)
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			return fmt.Sprintf("[✗] 生成代码失败: %v", err)
		}

		newContent = buf.Bytes()
		result = fmt.Sprintf("[✓] Go智能重命名: %s → %s（共%d处，等待批准）", oldSymbol, newSymbol, changeCount)
	} else {
		// 其他文件用正则
		text := string(oldContent)
		pattern := `\b` + regexp.QuoteMeta(oldSymbol) + `\b`
		re := regexp.MustCompile(pattern)

		matches := re.FindAllStringIndex(text, -1)
		if len(matches) == 0 {
			return fmt.Sprintf("[✗] 未找到符号: %s", oldSymbol)
		}

		newText := re.ReplaceAllString(text, newSymbol)
		newContent = []byte(newText)
		result = fmt.Sprintf("[✓] 通用智能重命名: %s → %s（共%d处，等待批准）", oldSymbol, newSymbol, len(matches))
	}

	// 写入新内容
	if err := os.WriteFile(file, newContent, 0644); err != nil {
		return fmt.Sprintf("[✗] 写入文件失败: %v", err)
	}

	// 保存备份
	bm.AddBackup(toolCallID, "rename", file, oldContent)

	return result
}

// ExecuteDeleteFile 删除文件（支持远程）
func ExecuteDeleteFile(toolCallID string, args map[string]interface{}, bm *backup.Manager) string {
	file := args["file"].(string)

	// 获取目标机器（由executor注入）
	targetMachine, _ := args["_target_machine"].(string)
	if targetMachine == "" {
		targetMachine = "local"
	}

	// 远程删除
	if targetMachine != "local" {
		// 远程删除不支持备份恢复（太复杂），直接提示用户
		return fmt.Sprintf("[✗] 远程删除暂不支持，请使用 run_command: rm '%s' (机器: %s)", file, targetMachine)
	}

	// 本地删除（带备份）
	oldContent, err := os.ReadFile(file)
	if err != nil {
		return fmt.Sprintf("[✗] 读取文件失败: %v", err)
	}

	if err := os.Remove(file); err != nil {
		return fmt.Sprintf("[✗] 删除失败: %v", err)
	}

	bm.AddBackup(toolCallID, "delete", file, oldContent)

	return fmt.Sprintf("[✓] 文件已删除: %s（等待用户确认）", file)
}
