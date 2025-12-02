package tools

import "github.com/sashabaranov/go-openai"

// GetToolsSimplified 返回简化后的工具定义
func GetToolsSimplified() []openai.Tool {
	return []openai.Tool{
		// 1. 文件操作工具（整合：read/edit/rename/delete/search）
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "file_operation",
				Description: "统一的文件操作工具。支持：read(读取)、edit(编辑)、rename(重命名符号)、delete(删除)、search(搜索代码)。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "操作类型：read/edit/rename/delete/search",
							"enum":        []string{"read", "edit", "rename", "delete", "search"},
						},
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径（所有操作都需要）",
						},
						// read 专用
						"start_line": map[string]interface{}{
							"type":        "integer",
							"description": "起始行号（read操作，读取大文件时使用）",
						},
						"end_line": map[string]interface{}{
							"type":        "integer",
							"description": "结束行号（read操作，读取大文件时使用）",
						},
						// edit 专用
						"old": map[string]interface{}{
							"type":        "string",
							"description": "要替换的内容（edit操作必需，必须唯一匹配）",
						},
						"new": map[string]interface{}{
							"type":        "string",
							"description": "新内容（edit操作必需）",
						},
						// rename 专用
						"old_symbol": map[string]interface{}{
							"type":        "string",
							"description": "旧符号名（rename操作必需）",
						},
						"new_symbol": map[string]interface{}{
							"type":        "string",
							"description": "新符号名（rename操作必需）",
						},
						// search 专用
						"query": map[string]interface{}{
							"type":        "string",
							"description": "搜索关键词（search操作必需）",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "搜索路径（search操作，默认当前目录）",
						},
						"file_pattern": map[string]interface{}{
							"type":        "string",
							"description": "文件过滤（search操作，如*.go）",
						},
					},
					"required": []string{"action", "file"},
				},
			},
		},

		// 2. 命令执行工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_command",
				Description: "在持久Shell中执行命令（保持工作目录和环境变量）。自动路由到当前控制机。白名单命令自动批准，其他需用户批准。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "要执行的命令",
						},
						"interactive": map[string]interface{}{
							"type":        "boolean",
							"description": "是否交互式运行（通常为false）",
							"default":     false,
						},
					},
					"required": []string{"command"},
				},
			},
		},

		// 3. 机器切换工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "switch_machine",
				Description: "切换当前控制机。命令执行会自动路由到当前控制机。可用控制机列表已在系统提示词中显示。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"machine_id": map[string]interface{}{
							"type":        "string",
							"description": "机器ID",
						},
					},
					"required": []string{"machine_id"},
				},
			},
		},

		// 4. 网络搜索工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "web_search",
				Description: "在互联网上搜索信息。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "搜索关键词或问题",
						},
						"max_results": map[string]interface{}{
							"type":        "integer",
							"description": "最多返回结果数，默认5",
							"default":     5,
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}
