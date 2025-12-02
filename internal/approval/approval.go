package approval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ai_assistant/internal/backup"
	"ai_assistant/internal/tools"

	"github.com/sashabaranov/go-openai"
)

// 命令白名单：查询类命令，无需批准
var commandWhitelist = []string{
	// 目录操作
	"ls", "dir", "pwd", "cd", "tree", "pushd", "popd",
	// 文件查看
	"cat", "type", "more", "less", "head", "tail", "echo", "Get-Content",
	// 信息查询
	"whoami", "hostname", "date", "time", "ver", "uname", "systeminfo",
	// 进程查询
	"ps", "tasklist", "Get-Process",
	// 网络查询
	"ipconfig", "ifconfig", "ping", "tracert", "nslookup",
	// Git查询
	"git status", "git log", "git diff", "git branch",
	// 其他查询
	"which", "where", "env", "printenv", "set", "Get-Variable",
}

// 命令黑名单：需要TTY交互的命令，无法在持久Shell中工作，直接拒绝
var commandBlacklist = []string{
	// 文本编辑器（需要TTY）
	"nano", "vim", "vi", "emacs", "notepad",
	// 交互式数据库客户端（直接连接）
	"mysql", "psql", "mongo", "redis-cli",
	// 其他交互式程序
	"top", "htop", "less", "more",
	// 注意：ssh/telnet/ftp 如果使用非交互模式（如 ssh user@host "command"）是允许的
	// 只有交互式登录（如 ssh user@host）才会有问题，但我们允许AI尝试
}

// isCommandInList 检查命令是否在列表中
func isCommandInList(command string, list []string) bool {
	command = strings.ToLower(strings.TrimSpace(command))
	for _, item := range list {
		if strings.HasPrefix(command, strings.ToLower(item)) {
			return true
		}
	}
	return false
}

// HandleApproval 处理工具调用批准
func HandleApproval(toolCalls []openai.ToolCall) map[string]bool {
	// 分类工具调用
	var immediateApprovalOps []openai.ToolCall // 需要立即批准的
	var autoApprovalOps []openai.ToolCall      // 自动批准的
	var blacklistedOps []openai.ToolCall       // 黑名单命令

	for _, tc := range toolCalls {
		// 特殊处理 run_command：检查命令白名单/黑名单
		if tc.Function.Name == "run_command" {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			if command, ok := args["command"].(string); ok {
				// 黑名单：直接拒绝
				if isCommandInList(command, commandBlacklist) {
					blacklistedOps = append(blacklistedOps, tc)
					continue
				}
				// 白名单：自动批准
				if isCommandInList(command, commandWhitelist) {
					autoApprovalOps = append(autoApprovalOps, tc)
					continue
				}
				// 其他：需要批准
				immediateApprovalOps = append(immediateApprovalOps, tc)
			} else {
				immediateApprovalOps = append(immediateApprovalOps, tc)
			}
		} else if tools.NeedsImmediateApproval(tc.Function.Name) {
			immediateApprovalOps = append(immediateApprovalOps, tc)
		} else {
			autoApprovalOps = append(autoApprovalOps, tc)
		}
	}

	approvals := make(map[string]bool)

	// 黑名单命令：直接拒绝
	if len(blacklistedOps) > 0 {
		fmt.Println("\n[✗] 以下命令被黑名单拒绝（不支持交互式命令）：")
		for _, tc := range blacklistedOps {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			fmt.Printf("  - %s\n", args["command"])
			approvals[tc.ID] = false
		}
	}

	// 所有非立即批准的操作：自动批准（包括查询、修改等）
	for _, tc := range autoApprovalOps {
		approvals[tc.ID] = true
	}

	// 只有危险的执行操作才需要立即批准
	if len(immediateApprovalOps) > 0 {
		fmt.Println("\n[!] 警告：以下操作不可撤销，需要立即批准：")
		for i, tc := range immediateApprovalOps {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			fmt.Printf("%d. %s(%v)\n", i+1, tc.Function.Name, args)
		}
		fmt.Println("\n批准方式:")
		fmt.Println("  y       - 全部同意")
		fmt.Println("  n       - 全部拒绝")
		fmt.Println("  y 1,2,3 - 只同意指定的（白名单）")
		fmt.Println("  n 1,3   - 拒绝指定的（黑名单）")
		fmt.Print("\n请选择: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// 解析输入
		parts := strings.Fields(input) // 按空格分割
		if len(parts) == 0 {
			// 空输入，默认拒绝
			return approvals
		}

		action := strings.ToLower(parts[0])

		if action == "y" {
			if len(parts) == 1 {
				// 只有y，全部同意
				for _, tc := range immediateApprovalOps {
					approvals[tc.ID] = true
				}
			} else {
				// y 1,2,3 - 白名单模式
				indices := parseIndices(parts[1])
				for _, idx := range indices {
					if idx > 0 && idx <= len(immediateApprovalOps) {
						approvals[immediateApprovalOps[idx-1].ID] = true
					}
				}
			}
		} else if action == "n" {
			if len(parts) == 1 {
				// 只有n，全部拒绝（默认都是false）
			} else {
				// n 1,3 - 黑名单模式：先全部同意，再拒绝指定的
				for _, tc := range immediateApprovalOps {
					approvals[tc.ID] = true
				}
				indices := parseIndices(parts[1])
				for _, idx := range indices {
					if idx > 0 && idx <= len(immediateApprovalOps) {
						approvals[immediateApprovalOps[idx-1].ID] = false
					}
				}
			}
		}
	}

	return approvals
}

// ConfirmModifyOperations 确认修改操作
func ConfirmModifyOperations(bm *backup.Manager) {
	if !bm.HasBackups() {
		return
	}

	fmt.Println("\n[i] 以下修改已执行，请确认：")
	backups := bm.GetBackups()
	for i, backup := range backups {
		if backup.EditCount > 1 {
			fmt.Printf("%d. %s (%d次修改)\n", i+1, backup.FilePath, backup.EditCount)
		} else {
			fmt.Printf("%d. %s\n", i+1, backup.FilePath)
		}
	}

	fmt.Println("\n批准方式:")
	fmt.Println("  y       - 全部确认")
	fmt.Println("  n       - 全部撤销")
	fmt.Println("  y 1,2,3 - 只确认指定的（白名单）")
	fmt.Println("  n 1,3   - 撤销指定的（黑名单）")
	fmt.Print("\n请选择: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// 解析输入
	parts := strings.Fields(input)
	if len(parts) == 0 {
		fmt.Println("[!] 无输入，默认全部撤销")
		for _, backup := range backups {
			bm.UndoOperation(backup.ToolCallID)
		}
		return
	}

	action := strings.ToLower(parts[0])

	if action == "y" {
		if len(parts) == 1 {
			// y - 全部确认
			bm.CommitAll()
			fmt.Println("[✓] 所有修改已确认")
		} else {
			// y 1,2,3 - 白名单：只确认指定的
			indices := parseIndices(parts[1])
			approved := make(map[int]bool)
			for _, idx := range indices {
				if idx > 0 && idx <= len(backups) {
					approved[idx-1] = true
				}
			}

			for i := len(backups) - 1; i >= 0; i-- {
				if !approved[i] {
					bm.UndoOperation(backups[i].ToolCallID)
					fmt.Printf("[✗] 已撤销: %s\n", backups[i].FilePath)
				}
			}

			bm.CommitAll()
			fmt.Println("[✓] 已确认指定的修改")
		}
	} else if action == "n" {
		if len(parts) == 1 {
			// n - 全部撤销
			for _, backup := range backups {
				bm.UndoOperation(backup.ToolCallID)
			}
			fmt.Println("[✗] 所有修改已撤销")
		} else {
			// n 1,3 - 黑名单：只撤销指定的
			indices := parseIndices(parts[1])
			rejected := make(map[int]bool)
			for _, idx := range indices {
				if idx > 0 && idx <= len(backups) {
					rejected[idx-1] = true
				}
			}

			for i, backup := range backups {
				if rejected[i] {
					bm.UndoOperation(backup.ToolCallID)
					fmt.Printf("[✗] 已撤销: %s\n", backup.FilePath)
				}
			}

			bm.CommitAll()
			fmt.Println("[✓] 其他修改已确认")
		}
	}
}

// 解析序号列表：1,2,3
func parseIndices(s string) []int {
	var indices []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		var idx int
		fmt.Sscanf(strings.TrimSpace(p), "%d", &idx)
		if idx > 0 {
			indices = append(indices, idx)
		}
	}
	return indices
}
