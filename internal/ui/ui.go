package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ai_assistant/internal/environment"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// 颜色定义
var (
	colorTitle   = color.New(color.FgCyan, color.Bold)
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorError   = color.New(color.FgRed, color.Bold)
	colorWarning = color.New(color.FgYellow, color.Bold)
	colorInfo    = color.New(color.FgBlue)
	colorMuted   = color.New(color.FgHiBlack)
	colorUser    = color.New(color.FgGreen, color.Bold)
	colorAI      = color.New(color.FgCyan, color.Bold)
	colorTool    = color.New(color.FgMagenta)
)

// 符号定义（兼容Windows/Linux）
const (
	SymbolSuccess = "[✓]"
	SymbolError   = "[✗]"
	SymbolInfo    = "[i]"
	SymbolWarning = "[!]"
	SymbolTool    = "[>]"
	SymbolUser    = "USER"
	SymbolAI      = "JARVIS"
)

// PrintWelcome 打印欢迎信息
func PrintWelcome(env environment.SystemEnvironment) {
	fmt.Println()
	colorTitle.Println("=" + strings.Repeat("=", 59))
	colorTitle.Println("        J.A.R.V.I.S - Just A Rather Very Intelligent System")
	colorTitle.Println("=" + strings.Repeat("=", 59))

	// 显示环境信息
	colorInfo.Println("\n[系统环境]")
	fmt.Printf("  - 操作系统  : %s\n", env.OS)
	fmt.Printf("  - Shell    : %s\n", env.Shell)
	if env.PythonCommand != "none" {
		colorSuccess.Printf("  - Python   : %s %s\n", env.PythonCommand, SymbolSuccess)
	} else {
		colorError.Printf("  - Python   : 未安装 %s\n", SymbolError)
	}
	if env.HasGit {
		colorSuccess.Printf("  - Git      : 可用 %s\n", SymbolSuccess)
	} else {
		colorError.Printf("  - Git      : 未安装 %s\n", SymbolError)
	}

	colorInfo.Println("\n[功能特性]")
	fmt.Println("  - 查询操作（read, list）  : 自动执行")
	fmt.Println("  - 修改操作（edit, rename）: 先执行后确认，可撤销")
	fmt.Println("  - 危险操作（run, commit） : 提前批准，不可撤销")

	colorMuted.Println("\n" + strings.Repeat("-", 60))
	fmt.Println()
}

// PrintHistoryLoaded 打印历史加载信息
func PrintHistoryLoaded(count int) {
	colorInfo.Printf("%s 已加载 %d 条历史消息\n\n", SymbolInfo, count)
}

// PrintUserPrompt 打印用户输入提示
func PrintUserPrompt() {
	// 打印分隔线
	fmt.Println()
	colorMuted.Println(strings.Repeat("─", 60))
	fmt.Println()
	colorUser.Print("▶ " + SymbolUser + " >> ")
}

// PrintAIPrompt 打印AI输出提示
func PrintAIPrompt() {
	fmt.Println()
	colorAI.Print("◆ " + SymbolAI + " >> ")
}

// ToolSpinner 工具执行的spinner
type ToolSpinner struct {
	s        *spinner.Spinner
	toolName string
}

// StartToolExecution 开始工具执行（显示spinner）
func StartToolExecution(toolName string) *ToolSpinner {
	// 打印工具调用框
	fmt.Println()
	colorMuted.Println("  ┌" + strings.Repeat("─", 56) + "┐")
	colorMuted.Print("  │ ")
	colorTool.Printf("%-54s", "工具调用: "+toolName)
	colorMuted.Println(" │")
	colorMuted.Print("  │ ")

	// 创建spinner（Windows兼容的字符集）
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = ""
	s.Suffix = " 执行中..."
	s.Writer = os.Stderr
	s.Start()

	return &ToolSpinner{
		s:        s,
		toolName: toolName,
	}
}

// Stop 停止spinner
func (ts *ToolSpinner) Stop() {
	if ts.s != nil {
		ts.s.Stop()
	}
}

// Success 显示成功结果
func (ts *ToolSpinner) Success(message string) {
	ts.Stop()
	// 清除spinner行
	fmt.Print("\r\033[K")

	// 替换emoji为符号
	message = strings.ReplaceAll(message, "❌", SymbolError)
	message = strings.ReplaceAll(message, "✅", SymbolSuccess)

	// 判断结果类型
	var statusSymbol string
	var statusColor *color.Color

	if strings.Contains(message, SymbolError) || strings.Contains(message, "失败") || strings.Contains(message, "错误") {
		statusSymbol = SymbolError
		statusColor = colorError
	} else if strings.HasPrefix(message, "[✓]") || strings.Contains(message, "成功") {
		statusSymbol = SymbolSuccess
		statusColor = colorSuccess
	} else {
		statusSymbol = SymbolSuccess
		statusColor = colorSuccess
	}

	// 打印状态
	statusColor.Print(statusSymbol + " ")
	colorMuted.Print("完成")

	// 打印结果（缩进显示）
	fmt.Println()
	colorMuted.Print("  │ ")
	colorMuted.Println(strings.Repeat("─", 56))

	// 处理多行消息
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if line != "" {
			colorMuted.Print("  │ ")
			fmt.Println(line)
		}
	}

	colorMuted.Println("  └" + strings.Repeat("─", 56) + "┘")
}

// Error 显示错误结果
func (ts *ToolSpinner) Error(message string) {
	ts.Stop()
	// 清除spinner行
	fmt.Print("\r\033[K")

	// 替换emoji
	message = strings.ReplaceAll(message, "❌", SymbolError)
	message = strings.ReplaceAll(message, "✅", SymbolSuccess)

	// 打印错误状态
	colorError.Print(SymbolError + " ")
	colorMuted.Print("失败")

	// 打印结果
	fmt.Println()
	colorMuted.Print("  │ ")
	colorMuted.Println(strings.Repeat("─", 56))

	// 处理多行消息
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if line != "" {
			colorMuted.Print("  │ ")
			colorError.Println(line)
		}
	}

	colorMuted.Println("  └" + strings.Repeat("─", 56) + "┘")
}

// PrintToolResult 打印工具执行结果（带spinner的版本）
func PrintToolResult(toolName, result string) {
	fmt.Print("\n")

	// 替换所有emoji
	result = strings.ReplaceAll(result, "❌", SymbolError)
	result = strings.ReplaceAll(result, "✅", SymbolSuccess)
	result = strings.ReplaceAll(result, "📄", "[FILE]")
	result = strings.ReplaceAll(result, "🔍", "[SEARCH]")
	result = strings.ReplaceAll(result, "📁", "[DIR]")
	result = strings.ReplaceAll(result, "📝", "[GIT]")
	result = strings.ReplaceAll(result, "📊", "[STATS]")
	result = strings.ReplaceAll(result, "📦", "[PROJECT]")
	result = strings.ReplaceAll(result, "📤", "[OUTPUT]")
	result = strings.ReplaceAll(result, "ℹ️", SymbolInfo)
	result = strings.ReplaceAll(result, "⚠️", SymbolWarning)

	// 判断是成功还是失败
	if strings.Contains(result, SymbolError) {
		colorError.Printf("%s %s: ", SymbolError, toolName)
		fmt.Println(result)
	} else if strings.Contains(result, SymbolSuccess) {
		colorSuccess.Printf("%s %s: ", SymbolSuccess, toolName)
		fmt.Println(result)
	} else {
		colorTool.Printf("%s %s: ", SymbolTool, toolName)
		fmt.Println(result)
	}
}

// PrintGoodbye 打印再见信息
func PrintGoodbye() {
	fmt.Println()
	colorInfo.Println("再见！祝你工作愉快！")
	fmt.Println()
}

// PrintWarning 打印警告信息
func PrintWarning(message string) {
	colorWarning.Printf("\n%s %s\n", SymbolWarning, message)
}

// PrintError 打印错误信息
func PrintError(message string) {
	colorError.Printf("\n%s %s\n", SymbolError, message)
}

// PrintSuccess 打印成功信息
func PrintSuccess(message string) {
	colorSuccess.Printf("[✓] %s\n", message)
}

// PrintInfo 打印信息
func PrintInfo(message string) {
	colorInfo.Println(message)
}
