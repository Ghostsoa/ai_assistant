package history

import (
	"encoding/json"
	"os"

	appconfig "ai_assistant/internal/config"
	"ai_assistant/internal/environment"
	"ai_assistant/internal/prompt"

	"github.com/sashabaranov/go-openai"
)

// Message 消息结构
type Message struct {
	Role             string            `json:"role"`
	Content          string            `json:"content,omitempty"`
	ToolCalls        []openai.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string            `json:"tool_call_id,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"` // 思维链内容（仅本地保存，不发送API）
}

// Load 加载历史（限制最近N轮对话）
func Load(historyFile string) []Message {
	// 检测环境
	env := environment.Detect()
	systemPrompt := prompt.BuildSystemPrompt(env)

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return []Message{{Role: "system", Content: systemPrompt}}
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return []Message{{Role: "system", Content: systemPrompt}}
	}

	if len(messages) == 0 {
		return []Message{{Role: "system", Content: systemPrompt}}
	}

	// 更新system prompt为当前环境
	if len(messages) > 0 && messages[0].Role == "system" {
		messages[0].Content = systemPrompt
	}

	// 限制历史记录轮数
	if appconfig.GlobalConfig.MaxHistoryRounds > 0 && len(messages) > 1 {
		// 计算要保留的消息数（system prompt + N轮对话）
		// 每轮可能包含：user, assistant, tool(多个)
		systemMsg := messages[0]
		restMessages := messages[1:]

		// 统计用户消息数量（作为轮数）
		userMsgCount := 0
		for i := len(restMessages) - 1; i >= 0; i-- {
			if restMessages[i].Role == "user" {
				userMsgCount++
				if userMsgCount > appconfig.GlobalConfig.MaxHistoryRounds {
					// 截取从这条user消息之后的所有消息
					messages = append([]Message{systemMsg}, restMessages[i+1:]...)
					break
				}
			}
		}
	}

	return messages
}

// Save 保存历史
func Save(historyFile string, messages []Message) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyFile, data, 0644)
}

// ConvertToOpenAI 转换为OpenAI消息格式（包含 reasoning_content）
// 根据 DeepSeek 文档，在工具调用期间需要保留 reasoning_content
func ConvertToOpenAI(messages []Message) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage
	for _, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role:             msg.Role,
			Content:          msg.Content,
			ToolCalls:        msg.ToolCalls,
			ToolCallID:       msg.ToolCallID,
			ReasoningContent: msg.ReasoningContent, // go-openai 库支持此字段！
		}

		result = append(result, openaiMsg)
	}
	return result
}

// ClearReasoningContent 清除历史消息中的思维链内容（新轮对话开始时调用）
func ClearReasoningContent(messages []Message) {
	for i := range messages {
		messages[i].ReasoningContent = ""
	}
}
