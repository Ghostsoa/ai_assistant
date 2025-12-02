package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"ai_assistant/internal/backup"
	appconfig "ai_assistant/internal/config"
	"ai_assistant/internal/process"
	"ai_assistant/internal/tools"

	"github.com/sashabaranov/go-openai"
)

const SocketPath = "/tmp/jarvis.sock"

var (
	client         *openai.Client
	executor       *tools.Executor
	processManager *process.Manager
	backupManager  *backup.Manager
)

// Start 启动 JARVIS 守护进程
func Start() error {
	// 删除旧的 socket 文件
	os.Remove(SocketPath)

	// 创建 Unix socket 监听
	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("创建 socket 失败: %v", err)
	}
	defer os.Remove(SocketPath)

	fmt.Println("🤖 JARVIS Server 已启动")
	fmt.Printf("Socket: %s\n", SocketPath)

	// 初始化 AI 客户端（只初始化一次）
	client = openai.NewClient(appconfig.GlobalConfig.APIKey)
	if appconfig.GlobalConfig.BaseURL != "" {
		config := openai.DefaultConfig(appconfig.GlobalConfig.APIKey)
		config.BaseURL = appconfig.GlobalConfig.BaseURL
		client = openai.NewClientWithConfig(config)
	}

	// 初始化工具执行器
	processManager = process.NewManager()
	backupManager = backup.NewManager()
	executor = tools.NewExecutor(processManager, backupManager)

	// 监听连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		go handleClient(conn)
	}
}

// handleClient 处理客户端请求
func handleClient(conn net.Conn) {
	defer conn.Close()

	// 读取请求
	reader := bufio.NewReader(conn)
	query, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	query = strings.TrimSpace(query)

	if query == "" {
		return
	}

	// 处理 AI 对话，流式输出到 conn
	streamChat(conn, query)
}

// streamChat 流式对话
func streamChat(writer io.Writer, query string) {
	ctx := context.Background()

	// 构建消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: query,
		},
	}

	// 创建请求
	req := openai.ChatCompletionRequest{
		Model:    appconfig.GlobalConfig.Model,
		Messages: messages,
		Tools:    tools.GetTools(),
		Stream:   true,
	}

	// 创建流
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Fprintf(writer, "ERROR: %v\n", err)
		return
	}
	defer stream.Close()

	var fullResponse strings.Builder
	var toolCalls []openai.ToolCall
	var currentToolCall *openai.ToolCall

	// 流式接收
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(writer, "\nERROR: %v\n", err)
			return
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta

		// 处理内容
		if delta.Content != "" {
			fullResponse.WriteString(delta.Content)
			fmt.Fprint(writer, delta.Content)
		}

		// 处理工具调用
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.Index != nil {
					if int(*tc.Index) >= len(toolCalls) {
						toolCalls = append(toolCalls, openai.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: openai.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						})
						currentToolCall = &toolCalls[*tc.Index]
					} else {
						currentToolCall = &toolCalls[*tc.Index]
					}
				}

				if currentToolCall != nil {
					if tc.ID != "" {
						currentToolCall.ID = tc.ID
					}
					if tc.Function.Name != "" {
						currentToolCall.Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						currentToolCall.Function.Arguments += tc.Function.Arguments
					}
				}
			}
		}
	}

	fmt.Fprintln(writer)

	// 执行工具调用
	if len(toolCalls) > 0 {
		for _, toolCall := range toolCalls {
			fmt.Fprintf(writer, "\n[工具调用: %s]\n", toolCall.Function.Name)
			result := executor.Execute(toolCall)
			fmt.Fprintln(writer, result)
		}
	}
}
