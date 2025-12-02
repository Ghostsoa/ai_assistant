package tools

import "github.com/sashabaranov/go-openai"

// GetTools 返回所有工具定义
func GetTools() []openai.Tool {
	return []openai.Tool{
		// 网络搜索工具
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
		// 文件操作工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "read_file",
				Description: "读取文件内容。如果文件超过1000行，将返回文件摘要和行数统计，需要使用start_line和end_line参数按范围读取。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径",
						},
						"start_line": map[string]interface{}{
							"type":        "integer",
							"description": "起始行号（可选，用于读取大文件的指定范围）",
						},
						"end_line": map[string]interface{}{
							"type":        "integer",
							"description": "结束行号（可选，用于读取大文件的指定范围）",
						},
					},
					"required": []string{"file"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "edit_file",
				Description: "精准编辑文件，替换old为new（需用户批准，可撤销）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径",
						},
						"old": map[string]interface{}{
							"type":        "string",
							"description": "要替换的内容（必须唯一匹配）",
						},
						"new": map[string]interface{}{
							"type":        "string",
							"description": "新内容",
						},
					},
					"required": []string{"file", "old", "new"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "rename_symbol",
				Description: "智能重命名符号，Go文件用AST，其他文件用正则（需用户批准，可撤销）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径",
						},
						"old_symbol": map[string]interface{}{
							"type":        "string",
							"description": "旧符号名",
						},
						"new_symbol": map[string]interface{}{
							"type":        "string",
							"description": "新符号名",
						},
					},
					"required": []string{"file", "old_symbol", "new_symbol"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "delete_file",
				Description: "删除文件（需用户批准，可撤销）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径",
						},
					},
					"required": []string{"file"},
				},
			},
		},
		// 命令执行工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_command",
				Description: "在持久Shell中执行命令（保持工作目录和环境变量）。白名单命令（ls/pwd/cd等查询）自动批准；黑名单命令（nano/vim等交互式）直接拒绝；其他命令需用户批准。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "要执行的命令",
						},
						"interactive": map[string]interface{}{
							"type":        "boolean",
							"description": "是否交互式运行",
						},
					},
					"required": []string{"command", "interactive"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "switch_machine",
				Description: "切换当前控制机。不传参数则列出所有可用机器。命令执行会自动路由到当前控制机。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"machine_id": map[string]interface{}{
							"type":        "string",
							"description": "机器ID（可选，不传则列出所有机器）",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "send_input",
				Description: "向交互式进程发送输入（需用户批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"process_id": map[string]interface{}{
							"type":        "string",
							"description": "进程ID",
						},
						"input": map[string]interface{}{
							"type":        "string",
							"description": "输入内容",
						},
					},
					"required": []string{"process_id", "input"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_output",
				Description: "查看进程输出（查询操作，无需用户批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"process_id": map[string]interface{}{
							"type":        "string",
							"description": "进程ID",
						},
					},
					"required": []string{"process_id"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "kill_process",
				Description: "终止进程（需用户批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"process_id": map[string]interface{}{
							"type":        "string",
							"description": "进程ID",
						},
					},
					"required": []string{"process_id"},
				},
			},
		},
		// 代码搜索工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_code",
				Description: "在项目中搜索代码（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "搜索内容",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "搜索路径，默认当前目录",
						},
						"file_pattern": map[string]interface{}{
							"type":        "string",
							"description": "文件过滤，如 *.go",
						},
						"is_regex": map[string]interface{}{
							"type":        "boolean",
							"description": "是否使用正则表达式",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "find_symbol",
				Description: "查找函数、类型、变量的定义位置（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"symbol": map[string]interface{}{
							"type":        "string",
							"description": "符号名称",
						},
						"symbol_type": map[string]interface{}{
							"type":        "string",
							"description": "符号类型：function, type, var, const（可选）",
						},
					},
					"required": []string{"symbol"},
				},
			},
		},
		// 项目分析工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list_directory",
				Description: "列出目录下的文件和子目录（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "目录路径，默认当前目录",
						},
						"recursive": map[string]interface{}{
							"type":        "boolean",
							"description": "是否递归列出子目录",
						},
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "文件过滤，如 *.go",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_project_structure",
				Description: "获取项目目录树结构（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"max_depth": map[string]interface{}{
							"type":        "integer",
							"description": "最大深度，默认3",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_file_stats",
				Description: "获取文件统计信息（行数、大小等）（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径",
						},
					},
					"required": []string{"file"},
				},
			},
		},
		// Git工具
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "git_status",
				Description: "查看Git仓库状态（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "git_diff",
				Description: "查看文件修改差异（查询操作，无需批准）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{
							"type":        "string",
							"description": "文件路径，不指定则显示所有",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "git_commit",
				Description: "提交更改到Git（需用户批准，不可撤销）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type":        "string",
							"description": "提交信息",
						},
						"files": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "要提交的文件列表",
						},
					},
					"required": []string{"message"},
				},
			},
		},
	}
}
