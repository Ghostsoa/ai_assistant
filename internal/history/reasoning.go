package history

import (
	"encoding/json"
)

// ConvertToOpenAIWithReasoning 转换为OpenAI消息格式（保留reasoning_content）
// 使用 JSON 序列化来支持额外字段
func ConvertToOpenAIWithReasoning(messages []Message) []interface{} {
	var result []interface{}

	for _, msg := range messages {
		// 构造基本消息
		msgMap := map[string]interface{}{
			"role": msg.Role,
		}

		// 添加内容
		if msg.Content != "" {
			msgMap["content"] = msg.Content
		}

		// 添加工具调用
		if len(msg.ToolCalls) > 0 {
			msgMap["tool_calls"] = msg.ToolCalls
		}

		// 添加工具调用ID
		if msg.ToolCallID != "" {
			msgMap["tool_call_id"] = msg.ToolCallID
		}

		// 添加思维链内容（如果存在）
		if msg.ReasoningContent != "" {
			msgMap["reasoning_content"] = msg.ReasoningContent
		}

		result = append(result, msgMap)
	}

	return result
}

// MessageToMap 将 Message 转换为 map（用于自定义请求）
func MessageToMap(msg Message) map[string]interface{} {
	result := map[string]interface{}{
		"role": msg.Role,
	}

	if msg.Content != "" {
		result["content"] = msg.Content
	}

	if len(msg.ToolCalls) > 0 {
		result["tool_calls"] = msg.ToolCalls
	}

	if msg.ToolCallID != "" {
		result["tool_call_id"] = msg.ToolCallID
	}

	if msg.ReasoningContent != "" {
		result["reasoning_content"] = msg.ReasoningContent
	}

	return result
}

// ConvertToOpenAIJSON 转换为 JSON 格式（用于自定义 API 调用）
func ConvertToOpenAIJSON(messages []Message) ([]byte, error) {
	result := make([]interface{}, 0, len(messages))

	for _, msg := range messages {
		result = append(result, MessageToMap(msg))
	}

	return json.Marshal(result)
}
