package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"ai_assistant/internal/approval"
	"ai_assistant/internal/backup"
	"ai_assistant/internal/command"
	appconfig "ai_assistant/internal/config"
	"ai_assistant/internal/environment"
	"ai_assistant/internal/history"
	"ai_assistant/internal/keyboard"
	"ai_assistant/internal/process"
	"ai_assistant/internal/session"
	"ai_assistant/internal/tools"
	"ai_assistant/internal/ui"

	"github.com/sashabaranov/go-openai"
)

// getReasoningContent 从响应中提取思维链内容
func getReasoningContent(response openai.ChatCompletionStreamResponse) string {
	if len(response.Choices) == 0 {
		return ""
	}

	// 尝试通过JSON反序列化获取reasoning_content
	// 因为openai库可能不支持这个字段，我们用map来访问
	data, err := json.Marshal(response.Choices[0].Delta)
	if err != nil {
		return ""
	}

	var deltaMap map[string]interface{}
	if err := json.Unmarshal(data, &deltaMap); err != nil {
		return ""
	}

	if reasoning, ok := deltaMap["reasoning_content"].(string); ok {
		return reasoning
	}

	return ""
}

func main() {
	// 初始化配置
	if err := appconfig.Initialize(); err != nil {
		fmt.Printf("[✗] 初始化失败: %v\n", err)
		fmt.Println("\n按回车键退出...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	// 检测环境
	env := environment.Detect()

	// 打印欢迎信息
	ui.PrintWelcome(env)

	// 显示配置信息
	fmt.Printf("\n[配置] 目录: %s\n", appconfig.ConfigDir)
	fmt.Printf("[配置] 模型: %s\n", appconfig.GlobalConfig.Model)
	fmt.Printf("[配置] 历史轮数: %d\n\n", appconfig.GlobalConfig.MaxHistoryRounds)

	// 主输入reader
	reader := bufio.NewReader(os.Stdin)

	// 询问思维链显示模式
	showReasoning := false
	if appconfig.GlobalConfig.ReasoningMode == "ask" {
		fmt.Println()
		ui.PrintInfo("是否显示AI思维链路？(y/n，默认n)")
		fmt.Print("选择: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(strings.ToLower(choice))
		showReasoning = (choice == "y" || choice == "yes")
	} else if appconfig.GlobalConfig.ReasoningMode == "show" {
		showReasoning = true
	} else {
		showReasoning = false
	}

	// 初始化会话管理器
	sessionManager, err := session.NewManager()
	if err != nil {
		fmt.Printf("[✗] 会话初始化失败: %v\n", err)
		fmt.Println("\n按回车键退出...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	// 初始化命令处理器
	cmdHandler := command.NewHandler(sessionManager)

	// 显示当前会话
	currentSession := sessionManager.GetCurrentSession()
	fmt.Printf("[会话] %s [%s]\n", currentSession.Title, currentSession.ID)

	// 初始化管理器
	processManager := process.NewManager()
	backupManager := backup.NewManager()
	toolExecutor := tools.NewExecutor(processManager, backupManager)

	// 配置API客户端
	clientConfig := openai.DefaultConfig(appconfig.GlobalConfig.APIKey)
	clientConfig.BaseURL = appconfig.GlobalConfig.BaseURL
	client := openai.NewClientWithConfig(clientConfig)

	// 加载历史
	historyFile := sessionManager.GetCurrentHistoryFile()
	messages := history.Load(historyFile)
	ui.PrintHistoryLoaded(len(messages) - 1)

	// 主循环
	for {
		ui.PrintUserPrompt()
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "" {
			continue
		}

		// 处理退出命令（不需要 /）
		if userInput == "quit" || userInput == "exit" || userInput == "q" {
			ui.PrintGoodbye()
			break
		}

		// 处理斜杠命令
		if command.IsCommand(userInput) {
			handled, err := cmdHandler.Handle(userInput)
			if err != nil {
				ui.PrintError(err.Error())
				continue
			}
			if handled {
				// 特殊处理退出命令
				if strings.HasPrefix(userInput, "/exit") || strings.HasPrefix(userInput, "/quit") {
					break
				}
				// 如果是切换会话或新建会话，重新加载历史
				if strings.HasPrefix(userInput, "/switch") || strings.HasPrefix(userInput, "/new") {
					historyFile = sessionManager.GetCurrentHistoryFile()
					messages = history.Load(historyFile)
					currentSession = sessionManager.GetCurrentSession()
					fmt.Printf("\n[会话] %s [%s]\n", currentSession.Title, currentSession.ID)
					ui.PrintHistoryLoaded(len(messages) - 1)
				}
				// 如果是清空会话，重新加载历史
				if strings.HasPrefix(userInput, "/clear") {
					messages = history.Load(historyFile)
				}
				continue
			}
		}

		messages = append(messages, history.Message{Role: "user", Content: userInput})

		// 清除之前轮次的思维链内容（新一轮对话开始）
		history.ClearReasoningContent(messages)

		// 创建打断监听器
		var interruptMonitor *keyboard.InterruptMonitor
		if appconfig.GlobalConfig.EnableInterrupt {
			interruptMonitor = keyboard.NewInterruptMonitor(appconfig.GlobalConfig.InterruptKey)
			interruptMonitor.Start()
			fmt.Printf("\n[提示] 输入 '%s' 并回车可打断当前操作\n", appconfig.GlobalConfig.InterruptKey)
		}

		// 打印 JARVIS 提示符（整轮对话只打印一次）
		ui.PrintAIPrompt()

		// 调用API（流式）
		for {
			stream, err := client.CreateChatCompletionStream(context.Background(), openai.ChatCompletionRequest{
				Model:    appconfig.GlobalConfig.Model,
				Messages: history.ConvertToOpenAI(messages),
				Tools:    tools.GetTools(),
			})

			if err != nil {
				fmt.Printf("\n[✗] API错误: %v\n", err)
				break
			}
			defer stream.Close()

			// 收集流式响应
			var fullContent strings.Builder
			var fullReasoning strings.Builder
			var toolCalls []openai.ToolCall
			var displayedContent bool
			var displayedReasoning bool
			var thinkingSpinner *ui.ThinkingSpinner

			// 启动思考动画（等待首个token）
			thinkingSpinner = ui.StartThinking()

			for {
				// 检查是否被打断
				if interruptMonitor != nil && interruptMonitor.IsInterrupted() {
					stream.Close()
					if thinkingSpinner != nil {
						thinkingSpinner.Stop()
					}
					fmt.Println("\n[✗] 操作已被打断")
					goto interrupted
				}

				response, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					if thinkingSpinner != nil {
						thinkingSpinner.Stop()
					}
					fmt.Printf("\n[✗] 流式错误: %v\n", err)
					break
				}

				delta := response.Choices[0].Delta

				// 处理思维链内容（reasoning_content）
				if reasoningContent := getReasoningContent(response); reasoningContent != "" {
					fullReasoning.WriteString(reasoningContent)

					if showReasoning {
						// 停止思考动画
						if thinkingSpinner != nil {
							thinkingSpinner.Stop()
							thinkingSpinner = nil
						}

						// 首次输出思维链时显示标题
						if !displayedReasoning {
							ui.PrintReasoningStart()
							displayedReasoning = true
						}
						ui.PrintReasoningContent(reasoningContent)
					}
				}

				// 流式显示内容
				if delta.Content != "" {
					// 首次输出正文时，处理思维链结束
					if !displayedContent {
						if thinkingSpinner != nil {
							thinkingSpinner.Stop()
							thinkingSpinner = nil
						}

						// 如果有思维链，结束它
						if displayedReasoning {
							ui.PrintReasoningEnd()
							fmt.Println() // 额外换行，分隔思维链和回复内容
						}

						displayedContent = true
					}
					fmt.Print(delta.Content)
					fullContent.WriteString(delta.Content)
				}

				// 收集tool_calls（在最后的chunk中）
				if len(delta.ToolCalls) > 0 {
					for _, tc := range delta.ToolCalls {
						// 如果是新的tool call
						if tc.Index != nil && *tc.Index >= len(toolCalls) {
							toolCalls = append(toolCalls, openai.ToolCall{
								ID:   tc.ID,
								Type: tc.Type,
								Function: openai.FunctionCall{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							})
						} else if tc.Index != nil {
							// 累加arguments
							idx := *tc.Index
							toolCalls[idx].Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}

			if displayedContent {
				fmt.Println() // 换行
			} else if thinkingSpinner != nil {
				// 如果一直在思考没有输出内容，也要停止spinner
				thinkingSpinner.Stop()
			}

			// 保存AI消息（包含思维链）
			// 注意：reasoning_content 在同一轮工具调用中需要保留并发送给 API
			assistantMsg := history.Message{
				Role:             "assistant",
				Content:          fullContent.String(),
				ToolCalls:        toolCalls,
				ReasoningContent: fullReasoning.String(), // 保存并在工具调用时发送
			}

			messages = append(messages, assistantMsg)
			history.Save(historyFile, messages)
			sessionManager.UpdateSessionTime()

			// 处理工具调用
			if len(toolCalls) == 0 {
				break
			}

			// 批准流程
			approvals := approval.HandleApproval(toolCalls)

			// 执行工具
			for _, toolCall := range toolCalls {
				// 检查是否被打断
				if interruptMonitor != nil && interruptMonitor.IsInterrupted() {
					fmt.Println("\n[✗] 工具调用已被打断")
					goto interrupted
				}

				var result string
				if approved, exists := approvals[toolCall.ID]; exists && approved {
					// 启动spinner
					spinner := ui.StartToolExecution(toolCall.Function.Name)

					// 检查工具执行期间是否被打断
					result = toolExecutor.Execute(toolCall)
					if interruptMonitor != nil && interruptMonitor.IsInterrupted() {
						result = "[✗] 工具执行已被打断"
						spinner.Error(result)
					} else {
						// 根据结果显示成功或失败（只检查前100个字符，避免文件内容干扰判断）
						resultPrefix := result
						if len(result) > 100 {
							resultPrefix = result[:100]
						}
						if strings.HasPrefix(resultPrefix, "[✗]") || strings.Contains(resultPrefix, "失败") || strings.Contains(resultPrefix, "错误") {
							spinner.Error(result)
						} else {
							spinner.Success(result)
						}
					}
				} else {
					result = "[✗] 用户拒绝执行此操作"
					ui.PrintToolResult(toolCall.Function.Name, result)
				}

				messages = append(messages, history.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}

			history.Save(historyFile, messages)
			sessionManager.UpdateSessionTime()
		}

	interrupted:
		// 停止打断监听器
		if interruptMonitor != nil {
			interruptMonitor.Stop()
		}

		// AI回复完成后，确认修改操作
		approval.ConfirmModifyOperations(backupManager)
	}
}
