package tools

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"regexp"
	"strings"

	"ai_assistant/internal/backup"
)

// ExecuteReadFile 读取文件
func ExecuteReadFile(args map[string]interface{}) string {
	file := args["file"].(string)
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Sprintf("[✗] 读取失败: %v", err)
	}
	return fmt.Sprintf("[文件] 内容:\n```\n%s```", string(content))
}

// ExecuteEditFile 编辑文件
func ExecuteEditFile(toolCallID string, args map[string]interface{}, bm *backup.Manager) string {
	file := args["file"].(string)
	old := args["old"].(string)
	new := args["new"].(string)

	// 读取原内容（用于备份）
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

// ExecuteDeleteFile 删除文件
func ExecuteDeleteFile(toolCallID string, args map[string]interface{}, bm *backup.Manager) string {
	file := args["file"].(string)

	// 备份原文件
	oldContent, err := os.ReadFile(file)
	if err != nil {
		return fmt.Sprintf("[✗] 读取文件失败: %v", err)
	}

	// 删除文件
	if err := os.Remove(file); err != nil {
		return fmt.Sprintf("[✗] 删除失败: %v", err)
	}

	// 保存备份
	bm.AddBackup(toolCallID, "delete", file, oldContent)

	return fmt.Sprintf("[✓] 文件已删除: %s（等待用户确认）", file)
}
