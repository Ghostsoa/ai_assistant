package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ai_assistant/internal/state"
)

// ExecuteListDirectory 列出目录（支持远程，优化版）
func ExecuteListDirectory(args map[string]interface{}, sm *state.Manager) string {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// 获取目标机器（由executor注入或使用slot1）
	var targetMachine string
	if machineID, ok := args["_target_machine"].(string); ok && machineID != "" {
		targetMachine = machineID
	} else {
		// 使用slot1的机器
		slot1Machine := sm.GetSlot1Machine()
		if slot1Machine != nil {
			targetMachine = slot1Machine.ID
		} else {
			targetMachine = "local"
		}
	}

	// 远程机器：直接使用ls命令
	if targetMachine != "local" {
		cmd := fmt.Sprintf("ls -lah '%s' 2>&1 || echo '[目录不存在]'", path)
		output, err := sm.ExecuteOnAgent(targetMachine, cmd)
		if err != nil {
			return fmt.Sprintf("[✗] 列出目录失败: %v", err)
		}

		if strings.Contains(output, "[目录不存在]") {
			return fmt.Sprintf("[✗] 目录不存在: %s", path)
		}

		return fmt.Sprintf("[目录] %s (机器: %s):\n```\n%s```", path, targetMachine, output)
	}

	// 本地机器：原逻辑
	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	// 安全检查：禁止扫描根目录和大型目录
	absPath, _ := filepath.Abs(path)
	dangerousPaths := []string{"/", "/root", "/home", "/usr", "/var", "C:\\", "D:\\"}
	for _, danger := range dangerousPaths {
		if absPath == danger || strings.HasPrefix(absPath, danger+string(filepath.Separator)) {
			return fmt.Sprintf("[✗] 为避免系统过载，禁止扫描大型目录: %s\n提示：请使用 run_command 执行 ls 命令", absPath)
		}
	}

	pattern := ""
	if p, ok := args["pattern"].(string); ok {
		pattern = p
	}

	var files []string
	var totalSize int64

	walkFn := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 跳过隐藏目录
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && p != path {
			return nil
		}

		// 如果不递归，跳过子目录
		if !recursive && info.IsDir() && p != path {
			return nil
		}

		// 过滤文件
		if !info.IsDir() {
			if pattern != "" {
				matched, _ := regexp.MatchString(strings.ReplaceAll(pattern, "*", ".*"), info.Name())
				if !matched {
					return nil
				}
			}

			relPath := strings.TrimPrefix(p, path+"/")
			if relPath == p {
				relPath = info.Name()
			}

			files = append(files, fmt.Sprintf("%s (%d lines, %s)",
				relPath,
				countLines(p),
				formatSize(info.Size())))
			totalSize += info.Size()
		}

		return nil
	}

	if err := filepath.Walk(path, walkFn); err != nil {
		return fmt.Sprintf("[✗] 列出目录失败: %v", err)
	}

	if len(files) == 0 {
		return fmt.Sprintf("[i] 目录为空或无匹配文件: %s", path)
	}

	// 限制输出
	if len(files) > 100 {
		files = files[:100]
		return fmt.Sprintf("[目录] 目录列表（前100个）:\n%s\n\n总计: %s, ... 还有更多",
			strings.Join(files, "\n"),
			formatSize(totalSize))
	}

	return fmt.Sprintf("[目录] 目录列表:\n%s\n\n总计: %d 个文件, %s",
		strings.Join(files, "\n"),
		len(files),
		formatSize(totalSize))
}

// 辅助函数（本地 ExecuteListDirectory 使用）
func countLines(file string) int {
	content, err := os.ReadFile(file)
	if err != nil {
		return 0
	}
	return strings.Count(string(content), "\n") + 1
}

func formatSize(size int64) string {
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

// 以下函数已删除，使用 run_command 替代：
// - ExecuteGetProjectStructure -> run_command("tree" 或 "find . -type f")
// - ExecuteGetFileStats -> run_command("wc -l file" 和 "ls -lh file")
