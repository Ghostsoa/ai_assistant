# 首次运行指南

## 🚀 首次启动

运行 `jarvis.exe` 后，系统会自动检测是否为首次运行。如果是，会引导您完成初始配置。

### 配置步骤

```
╔═══════════════════════════════════════════════════════╗
║         欢迎使用 J.A.R.V.I.S - 超级人工智能          ║
╚═══════════════════════════════════════════════════════╝

检测到首次运行，请进行初始配置：

请输入您的 DeepSeek API Key: sk-xxxxxxxxxxxxxx

请选择模型:
  1. deepseek-chat (标准模型)
  2. deepseek-reasoner (支持思维链)
选择 [1/2，默认 2]: 2

思维链显示模式:
  1. 每次询问 (ask)
  2. 始终显示 (show)
  3. 始终隐藏 (hide)
选择 [1/2/3，默认 1]: 1

历史记录保留轮数 [默认 60，0 表示全部]: 60

[✓] 配置已保存到: C:\Users\...\AppData\Roaming\jarvis\config.json

[✓] 初始化完成！
```

## 📝 配置项说明

### 1. API Key
- **必填项**
- 您的 DeepSeek API 密钥
- 可从 [DeepSeek 官网](https://platform.deepseek.com) 获取

### 2. 模型选择
- **deepseek-chat**: 标准对话模型，响应速度快
- **deepseek-reasoner**: 支持思维链的推理模型，回答更深入

### 3. 思维链显示模式
- **ask (每次询问)**: 每次启动时询问是否显示思维链
- **show (始终显示)**: 总是显示AI的思考过程
- **hide (始终隐藏)**: 不显示思维链，只看最终答案

### 4. 历史记录轮数
- 决定保留多少轮对话历史
- `60`: 保留最近60轮对话（推荐）
- `0`: 保留全部历史（可能消耗较多token）

## 🔧 配置文件位置

**Windows:**
```
C:\Users\<用户名>\AppData\Roaming\jarvis\config.json
```

**Linux/Mac:**
```
~/.config/jarvis/config.json
```

## 📄 配置文件示例

```json
{
  "api_key": "sk-xxxxxxxxxxxxxx",
  "base_url": "https://api.deepseek.com/v1",
  "model": "deepseek-reasoner",
  "max_history_rounds": 60,
  "interrupt_key": "n",
  "enable_interrupt": true,
  "reasoning_mode": "ask"
}
```

## 🔄 重新配置

如果需要修改配置，有两种方式：

### 方式 1：手动编辑配置文件
1. 找到配置文件位置（见上方）
2. 用文本编辑器打开 `config.json`
3. 修改对应的配置项
4. 保存并重启程序

### 方式 2：删除配置文件重新初始化
1. 删除配置文件 `config.json`
2. 重新运行程序
3. 系统会再次引导您进行配置

## ⚠️ 注意事项

1. **API Key 安全**
   - 请妥善保管您的 API Key
   - 不要将配置文件分享给他人
   - 配置文件已自动排除在 Git 之外

2. **首次运行体验**
   - 配置过程只需1-2分钟
   - 所有选项都有合理的默认值
   - 可以直接按回车使用默认值

3. **配置生效**
   - 修改配置后需要重启程序
   - 会话历史不受配置更改影响

## 🎯 快速开始

如果您想快速开始使用，只需：

1. 输入您的 API Key
2. 其他全部按回车使用默认值

默认配置已经非常适合大多数使用场景！

---

**J.A.R.V.I.S** - 让AI助手的配置像对话一样简单！
