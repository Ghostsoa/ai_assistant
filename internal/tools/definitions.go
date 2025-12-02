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
				Description: "统一的文件操作工具。支持：read(读取)、edit(编辑)、rename(重命名符号)、delete(删除)、search(搜索代码)。不指定machine则在slot1机器执行。",
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
						"machine": map[string]interface{}{
							"type":        "string",
							"description": "机器ID（可选，不填则使用slot1的机器）",
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
				Description: "执行Shell命令（持久Shell）。不指定machine则在slot1机器执行，可指定任意机器ID直接执行。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "要执行的Shell命令",
						},
						"machine": map[string]interface{}{
							"type":        "string",
							"description": "机器ID（可选，不填则使用slot1的机器）",
						},
					},
					"required": []string{"command"},
				},
			},
		},

		// 3. 网络搜索工具
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

		// 5. 文件同步工具（统一）
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "sync",
				Description: "文件同步工具。支持：push(推送文件)、pull(拉取文件)、status(查询任务状态)。5秒内完成直接返回，超过5秒后台运行。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "操作类型：push/pull/status",
							"enum":        []string{"push", "pull", "status"},
						},
						// push/pull 参数
						"local": map[string]interface{}{
							"type":        "string",
							"description": "本地文件路径（push/pull时必需）",
						},
						"remote": map[string]interface{}{
							"type":        "string",
							"description": "远程文件路径（push/pull时必需）",
						},
						"machine": map[string]interface{}{
							"type":        "string",
							"description": "远程机器ID（push/pull时必需）",
						},
						// status 参数
						"task_id": map[string]interface{}{
							"type":        "string",
							"description": "任务ID（status时必需）",
						},
					},
					"required": []string{"action"},
				},
			},
		},

		// 6. 终端管理工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "terminal_manage",
				Description: "管理终端槽位。支持：open(打开slot2)、close(关闭slot2)、switch(切换slot到另一台机器)、status(查看终端状态)。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": "操作类型",
							"enum":        []string{"open", "close", "switch", "status"},
						},
						"slot": map[string]interface{}{
							"type":        "string",
							"description": "槽位ID：slot1或slot2（open/close/switch时必需）",
							"enum":        []string{"slot1", "slot2"},
						},
						"machine": map[string]interface{}{
							"type":        "string",
							"description": "机器ID（open/switch时必需）",
						},
					},
					"required": []string{"action"},
				},
			},
		},
	}
}
