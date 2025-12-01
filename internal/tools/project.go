package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExecuteListDirectory 列出目录
func ExecuteListDirectory(args map[string]interface{}) string {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
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

// ExecuteGetProjectStructure 获取项目树结构
func ExecuteGetProjectStructure(args map[string]interface{}) string {
	maxDepth := 3
	if md, ok := args["max_depth"].(float64); ok {
		maxDepth = int(md)
	}

	var result strings.Builder
	result.WriteString("[项目] 结构:\n```\n")

	var walk func(path string, depth int, prefix string)
	walk = func(path string, depth int, prefix string) {
		if depth > maxDepth {
			return
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		for i, entry := range entries {
			// 跳过隐藏文件和常见忽略目录
			if strings.HasPrefix(entry.Name(), ".") ||
				entry.Name() == "node_modules" ||
				entry.Name() == "vendor" {
				continue
			}

			isLast := i == len(entries)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			result.WriteString(prefix + connector + entry.Name())

			if entry.IsDir() {
				result.WriteString("/\n")
				newPrefix := prefix
				if isLast {
					newPrefix += "    "
				} else {
					newPrefix += "│   "
				}
				walk(path+"/"+entry.Name(), depth+1, newPrefix)
			} else {
				result.WriteString("\n")
			}
		}
	}

	walk(".", 0, "")
	result.WriteString("```")

	return result.String()
}

// ExecuteGetFileStats 获取文件统计
func ExecuteGetFileStats(args map[string]interface{}) string {
	file := args["file"].(string)

	info, err := os.Stat(file)
	if err != nil {
		return fmt.Sprintf("[✗] 获取文件信息失败: %v", err)
	}

	lineCount := countLines(file)

	return fmt.Sprintf("[统计] 文件统计: %s\n"+
		"- 大小: %s\n"+
		"- 行数: %d\n"+
		"- 修改时间: %s",
		file,
		formatSize(info.Size()),
		lineCount,
		info.ModTime().Format("2006-01-02 15:04:05"))
}

// 辅助函数
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
