package command

import (
	"fmt"
	"strings"

	"ai_assistant/internal/session"
	"ai_assistant/internal/state"
	"ai_assistant/internal/ui"
)

// Handler 命令处理器
type Handler struct {
	sessionManager *session.Manager
	stateManager   *state.Manager
}

// NewHandler 创建命令处理器
func NewHandler(sm *session.Manager, stm *state.Manager) *Handler {
	return &Handler{
		sessionManager: sm,
		stateManager:   stm,
	}
}

// IsCommand 判断是否是命令
func IsCommand(input string) bool {
	return strings.HasPrefix(input, "/")
}

// Handle 处理命令
func (h *Handler) Handle(input string) (bool, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false, nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/clear":
		return true, h.handleClear()
	case "/exit", "/quit", "/q":
		return true, h.handleExit()
	case "/new":
		return true, h.handleNew(args)
	case "/list":
		return true, h.handleList()
	case "/switch":
		return true, h.handleSwitch(args)
	case "/delete", "/del":
		return true, h.handleDelete(args)
	case "/rename":
		return true, h.handleRename(args)
	case "/infect":
		return true, h.handleInfect(args)
	case "/machines":
		return true, h.handleMachines()
	case "/help":
		return true, h.handleHelp()
	default:
		return false, fmt.Errorf("未知命令: %s (输入 /help 查看帮助)", cmd)
	}
}

// handleClear 清空当前会话
func (h *Handler) handleClear() error {
	if err := h.sessionManager.ClearCurrentSession(); err != nil {
		return err
	}
	ui.PrintSuccess("当前会话已清空")
	return nil
}

// handleExit 退出程序
func (h *Handler) handleExit() error {
	ui.PrintGoodbye()
	return nil
}

// handleNew 创建新会话
func (h *Handler) handleNew(args []string) error {
	title := "新对话"
	if len(args) > 0 {
		title = strings.Join(args, " ")
	}

	if err := h.sessionManager.NewSession(title); err != nil {
		return err
	}

	session := h.sessionManager.GetCurrentSession()
	ui.PrintSuccess(fmt.Sprintf("已创建新会话: %s [%s]", session.Title, session.ID))
	return nil
}

// handleList 列出所有会话
func (h *Handler) handleList() error {
	sessions, err := h.sessionManager.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		ui.PrintInfo("暂无会话")
		return nil
	}

	currentSession := h.sessionManager.GetCurrentSession()

	fmt.Println()
	ui.PrintInfo("所有会话：")
	fmt.Println()

	for i, s := range sessions {
		marker := "  "
		if currentSession != nil && s.ID == currentSession.ID {
			marker = "◆ " // 当前会话标记
		}

		fmt.Printf("%s[%d] %s\n", marker, i+1, s.Title)
		fmt.Printf("    ID: %s\n", s.ID)
		fmt.Printf("    创建: %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("    更新: %s\n", s.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

// handleSwitch 切换会话
func (h *Handler) handleSwitch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请指定会话ID或序号")
	}

	id := args[0]

	// 如果是数字，转换为ID
	if len(id) > 0 && id[0] >= '0' && id[0] <= '9' {
		sessions, err := h.sessionManager.ListSessions()
		if err != nil {
			return err
		}

		var index int
		fmt.Sscanf(id, "%d", &index)
		if index < 1 || index > len(sessions) {
			return fmt.Errorf("无效的序号: %s", id)
		}

		id = sessions[index-1].ID
	}

	if err := h.sessionManager.SwitchSession(id); err != nil {
		return err
	}

	session := h.sessionManager.GetCurrentSession()
	ui.PrintSuccess(fmt.Sprintf("已切换到会话: %s [%s]", session.Title, session.ID))
	return nil
}

// handleDelete 删除会话
func (h *Handler) handleDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请指定要删除的会话ID或序号")
	}

	id := args[0]

	// 如果是数字，转换为ID
	if len(id) > 0 && id[0] >= '0' && id[0] <= '9' {
		sessions, err := h.sessionManager.ListSessions()
		if err != nil {
			return err
		}

		var index int
		fmt.Sscanf(id, "%d", &index)
		if index < 1 || index > len(sessions) {
			return fmt.Errorf("无效的序号: %s", id)
		}

		id = sessions[index-1].ID
	}

	if err := h.sessionManager.DeleteSession(id); err != nil {
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("已删除会话: %s", id))
	return nil
}

// handleRename 重命名会话
func (h *Handler) handleRename(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法: /rename <ID或序号> <新标题>")
	}

	id := args[0]
	newTitle := strings.Join(args[1:], " ")

	// 如果是数字，转换为ID
	if len(id) > 0 && id[0] >= '0' && id[0] <= '9' {
		sessions, err := h.sessionManager.ListSessions()
		if err != nil {
			return err
		}

		var index int
		fmt.Sscanf(id, "%d", &index)
		if index < 1 || index > len(sessions) {
			return fmt.Errorf("无效的序号: %s", id)
		}

		id = sessions[index-1].ID
	}

	if err := h.sessionManager.RenameSession(id, newTitle); err != nil {
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("已重命名会话: %s", newTitle))
	return nil
}

// handleInfect 寄生目标服务器
func (h *Handler) handleInfect(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("用法: /infect <host> <user> <password> [alias]")
	}

	host := args[0]
	user := args[1]
	password := args[2]
	alias := fmt.Sprintf("server-%d", len(h.stateManager.GetState().Machines))
	if len(args) > 3 {
		alias = args[3]
	}

	ui.PrintInfo(fmt.Sprintf("🦠 正在寄生目标服务器 %s@%s...", user, host))

	// 调用infect脚本
	if err := h.stateManager.InfectServer(host, user, password, alias); err != nil {
		return fmt.Errorf("寄生失败: %v", err)
	}

	ui.PrintSuccess(fmt.Sprintf("✓ 成功寄生服务器！机器ID: %s", alias))
	ui.PrintInfo("可以使用 /machines 查看所有控制机")
	return nil
}

// handleMachines 列出所有控制机
func (h *Handler) handleMachines() error {
	fmt.Println()
	ui.PrintInfo("可用控制机：")
	fmt.Println()
	fmt.Println(h.stateManager.ListMachines())
	fmt.Println()
	ui.PrintInfo("提示：对AI说\"切换到 <机器ID>\"可以切换控制机")
	return nil
}

// handleHelp 显示帮助
func (h *Handler) handleHelp() error {
	fmt.Println()
	ui.PrintInfo("可用命令：")
	fmt.Println()
	fmt.Println("  /new [标题]       - 创建新会话")
	fmt.Println("  /list             - 列出所有会话")
	fmt.Println("  /switch <ID|序号> - 切换到指定会话")
	fmt.Println("  /clear            - 清空当前会话历史")
	fmt.Println("  /rename <ID|序号> <新标题> - 重命名会话")
	fmt.Println()
	fmt.Println("  /infect <host> <user> <password> [alias] - 寄生目标服务器")
	fmt.Println("  /machines         - 列出所有控制机")
	fmt.Println("  /delete <ID|序号> - 删除会话")
	fmt.Println("  /help             - 显示此帮助")
	fmt.Println("  /exit, /quit, /q  - 退出程序")
	fmt.Println()
	fmt.Println("  quit, exit, q     - 退出程序（不需要 /）")
	fmt.Println()

	return nil
}
