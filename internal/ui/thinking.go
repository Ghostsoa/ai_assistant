package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

// ThinkingSpinner 思考动画spinner
type ThinkingSpinner struct {
	s *spinner.Spinner
}

// StartThinking 开始思考动画
func StartThinking() *ThinkingSpinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) // 使用 ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
	s.Prefix = " "
	s.Suffix = " 正在思考..."
	s.Writer = os.Stderr
	s.Color("cyan")
	s.Start()

	return &ThinkingSpinner{s: s}
}

// Stop 停止思考动画
func (ts *ThinkingSpinner) Stop() {
	if ts.s != nil {
		ts.s.Stop()
		// 清除spinner行
		fmt.Print("\r\033[K")
	}
}

// PrintReasoningStart 打印思维链开始标记
func PrintReasoningStart() {
	fmt.Println()
	colorMuted.Println(strings.Repeat("─", 10) + " 思维链路 " + strings.Repeat("─", 10))
}

// PrintReasoningEnd 打印思维链结束标记
func PrintReasoningEnd() {
	fmt.Println()
	// 计算长度：10 + " 思维链路 "(5个汉字=10字节+2空格) + 10 = 30
	// 但由于汉字占用，实际显示需要调整
	colorMuted.Println(strings.Repeat("─", 30))
}

// PrintReasoningContent 打印思维链内容（流式）
func PrintReasoningContent(content string) {
	// 直接打印内容，不添加左右边框
	colorMuted.Print(content)
}
