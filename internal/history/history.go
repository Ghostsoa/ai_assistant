package history

import (
	"encoding/json"
	"os"

	"ai_assistant/config"
	"ai_assistant/internal/environment"
	"ai_assistant/internal/prompt"

	"github.com/sashabaranov/go-openai"
)

// Message 消息结构
type Message struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []openai.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

// Load 加载历史
func Load() []Message {
	// 检测环境
	env := environment.Detect()
	systemPrompt := prompt.BuildSystemPrompt(env)

	data, err := os.ReadFile(config.HistoryFile)
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

	return messages
}

// Save 保存历史
func Save(messages []Message) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(config.HistoryFile, data, 0644)
}

// ConvertToOpenAI 转换为OpenAI消息格式
func ConvertToOpenAI(messages []Message) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage
	for _, msg := range messages {
		result = append(result, openai.ChatCompletionMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCalls:  msg.ToolCalls,
			ToolCallID: msg.ToolCallID,
		})
	}
	return result
}
