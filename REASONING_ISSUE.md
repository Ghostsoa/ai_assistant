# Reasoning Content 问题说明

## 🔴 当前问题

使用 `deepseek-reasoner` 模型或启用 `thinking` 模式时，会出现 400 错误：
```
Missing `reasoning_content` field in the assistant message
```

## 🎯 根本原因

**go-openai 库的限制**：标准的 `openai.ChatCompletionMessage` 结构体不包含 `reasoning_content` 字段。

根据 DeepSeek 官方文档：
- 在工具调用的**同一轮对话中**，必须把 `reasoning_content` 发送回 API
- 只有在**新轮对话开始时**，才清除之前的 `reasoning_content`

但是 go-openai 库在序列化消息时会忽略这个字段，导致 API 报错。

## 📊 官方文档说明

```python
# Python 示例 - 可以直接 append message
messages.append(response.choices[0].message)  # 包含 reasoning_content

# 工具调用时会自动发送 reasoning_content
response = client.chat.completions.create(
    model="deepseek-reasoner",
    messages=messages,  # reasoning_content 会自动包含
    tools=tools
)

# 新一轮对话时清除
clear_reasoning_content(messages)
```

## 🔧 临时解决方案

### 方案 1：使用 deep seek-chat（推荐）

修改 `config/config.go`：
```go
Model = "deepseek-chat"  // 不使用 reasoner 模式
ReasoningMode = "hide"   // 关闭思维链显示
```

### 方案 2：禁用思维链

```go
ReasoningMode = "hide"  // 始终隐藏思维链
```

这样即使模型返回 `reasoning_content`，我们也不会尝试发送回 API。

##  完整解决方案（需要重构）

要完全支持 `reasoning_content`，需要：

1. **不使用 go-openai 库的结构体**，改用 `map[string]interface{}`
2. **自定义 HTTP 请求**，完全控制JSON序列化
3. **重写 API 调用逻辑**

示例代码结构：
```go
// 自定义消息结构
type CustomMessage map[string]interface{}

// 构造带 reasoning_content 的消息
func BuildMessage(msg history.Message) CustomMessage {
    result := CustomMessage{
        "role": msg.Role,
        "content": msg.Content,
    }
    
    if msg.ReasoningContent != "" {
        result["reasoning_content"] = msg.ReasoningContent
    }
    
    if len(msg.ToolCalls) > 0 {
        result["tool_calls"] = msg.ToolCalls
    }
    
    return result
}

// 自定义 HTTP 请求
func CallAPIWithReasoning(messages []CustomMessage) {
    requestBody := map[string]interface{}{
        "model": "deepseek-reasoner",
        "messages": messages,
        "tools": tools,
        "stream": true,
    }
    
    // 发送 HTTP 请求...
}
```

## 📝 当前代码状态

我们的代码已经：
- ✅ 在本地保存 `reasoning_content`
- ✅ 支持思维链的显示/隐藏
- ✅ 实现了新轮对话时清除思维链的逻辑
- ❌ 但无法通过 go-openai 库发送 `reasoning_content`

## 🚀 建议

**短期**：使用 `deepseek-chat` 模型，不启用思维链功能

**长期**：如果需要完整支持 reasoner 模式，考虑以下选项：
1. 等待 go-openai 库更新支持
2. Fork go-openai 库并添加支持
3. 完全重写 API 调用部分，使用原生 HTTP 请求
4. 使用其他支持自定义字段的 Go HTTP 客户端

## 📖 参考

- [DeepSeek 思考模式文档](https://api-docs.deepseek.com/guides/thinking_with_tools)
- [go-openai GitHub](https://github.com/sashabaranov/go-openai)

---

**当前状态**：代码可以正常运行，但不支持 deepseek-reasoner 模型。建议使用 deepseek-chat。
