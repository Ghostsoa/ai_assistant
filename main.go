package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"ai_assistant/config"
	"ai_assistant/internal/approval"
	"ai_assistant/internal/backup"
	"ai_assistant/internal/environment"
	"ai_assistant/internal/history"
	"ai_assistant/internal/process"
	"ai_assistant/internal/tools"
	"ai_assistant/internal/ui"

	"github.com/sashabaranov/go-openai"
)

func main() {
	// 检测环境
	env := environment.Detect()

	// 打印欢迎信息
	ui.PrintWelcome(env)

	// 初始化管理器
	processManager := process.NewManager()
	backupManager := backup.NewManager()
	toolExecutor := tools.NewExecutor(processManager, backupManager)

	// 配置API客户端
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = config.BaseURL
	client := openai.NewClientWithConfig(clientConfig)

	// 加载历史
	messages := history.Load()
	ui.PrintHistoryLoaded(len(messages) - 1)

	// 主循环
	reader := bufio.NewReader(os.Stdin)
	for {
		ui.PrintUserPrompt()
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "" {
			continue
		}

		if userInput == "quit" || userInput == "exit" || userInput == "q" {
			ui.PrintGoodbye()
			break
		}

		messages = append(messages, history.Message{Role: "user", Content: userInput})

		// 调用API（流式）
		for {
			stream, err := client.CreateChatCompletionStream(context.Background(), openai.ChatCompletionRequest{
				Model:    config.Model,
				Messages: history.ConvertToOpenAI(messages),
				Tools:    tools.GetTools(),
			})

			if err != nil {
				fmt.Printf("\n❌ API错误: %v\n", err)
				break
			}
			defer stream.Close()

			// 收集流式响应
			var fullContent strings.Builder
			var toolCalls []openai.ToolCall
			var displayedContent bool

			ui.PrintAIPrompt()

			for {
				response, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Printf("\n❌ 流式错误: %v\n", err)
					break
				}

				delta := response.Choices[0].Delta

				// 流式显示内容
				if delta.Content != "" {
					if !displayedContent {
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
			}

			// 保存AI消息
			messages = append(messages, history.Message{
				Role:      "assistant",
				Content:   fullContent.String(),
				ToolCalls: toolCalls,
			})
			history.Save(messages)

			// 处理工具调用
			if len(toolCalls) == 0 {
				break
			}

			// 批准流程
			approvals := approval.HandleApproval(toolCalls)

			// 执行工具
			for _, toolCall := range toolCalls {
				var result string
				if approved, exists := approvals[toolCall.ID]; exists && approved {
					// 启动spinner
					spinner := ui.StartToolExecution(toolCall.Function.Name)
					result = toolExecutor.Execute(toolCall)

					// 根据结果显示成功或失败
					if strings.Contains(result, "[✗]") || strings.Contains(result, "失败") || strings.Contains(result, "错误") {
						spinner.Error(result)
					} else {
						spinner.Success(result)
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

			history.Save(messages)
		}

		// AI回复完成后，确认修改操作
		approval.ConfirmModifyOperations(backupManager)
	}
}
