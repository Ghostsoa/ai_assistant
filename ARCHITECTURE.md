# 架构设计文档

## 📐 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                      main.go                            │
│              (主程序入口 - 约170行)                        │
└────────────────────┬────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ▼                     ▼
    ┌─────────┐          ┌──────────┐
    │ config  │          │ internal │
    └─────────┘          └─────┬────┘
                               │
        ┌──────────────────────┼────────────────────────┐
        ▼          ▼           ▼        ▼        ▼      ▼
    ┌────────┐ ┌───────┐ ┌─────────┐ ┌──────┐ ┌────┐ ┌────┐
    │process │ │backup │ │approval │ │tools │ │env │ │ui  │
    └────────┘ └───────┘ └─────────┘ └──────┘ └────┘ └────┘
                                         │
                    ┌────────────────────┼────────────────┐
                    ▼          ▼         ▼        ▼       ▼
                 file.go  command.go search.go project.go git.go
```

## 🏗️ 模块职责

### 1. `main.go` - 主程序
**职责**: 程序入口，协调各模块
- 初始化所有管理器
- 处理用户输入循环
- 调用OpenAI API（流式）
- 协调工具执行和批准流程

**关键流程**:
```
用户输入 → API调用 → 流式响应 → 工具调用 → 批准 → 执行 → 确认修改
```

---

### 2. `config/` - 配置包
**职责**: 集中管理配置常量

**文件**:
- `config.go`: API密钥、BaseURL、模型名、历史文件路径

**优点**:
- 修改配置只需改一处
- 便于环境变量化（未来）

---

### 3. `internal/process/` - 进程管理
**职责**: 管理交互式进程的生命周期

**核心类型**:
```go
type Manager struct {
    processes map[string]*ProcessInfo  // 进程表
    mutex     sync.Mutex               // 并发安全
    counter   int                      // ID生成器
}
```

**功能**:
- `StartProcess()`: 启动bash进程
- `SendInput()`: 发送输入到进程stdin
- `GetOutput()`: 获取进程输出（非阻塞）
- `KillProcess()`: 终止进程
- `collectOutput()`: 后台收集进程输出（goroutine）

**特点**:
- 支持交互式命令（如Python REPL）
- 输出缓冲（最多1000行）
- 线程安全

---

### 4. `internal/backup/` - 备份撤销
**职责**: 文件修改的备份与恢复

**核心类型**:
```go
type OperationBackup struct {
    ToolCallID string    // 工具调用ID
    Type       string    // edit/rename/delete
    FilePath   string    // 文件路径
    OldContent []byte    // 原始内容
    EditCount  int       // 修改次数
}
```

**策略**:
- 同一文件只保留**第一次备份**
- 记录修改次数（EditCount）
- 支持撤销到初始状态

**功能**:
- `AddBackup()`: 添加备份（智能去重）
- `UndoOperation()`: 撤销操作
- `CommitAll()`: 提交所有修改（清空备份）
- `GetBackups()`: 获取备份列表

---

### 5. `internal/approval/` - 批准流程
**职责**: 处理工具调用的批准逻辑

**批准策略**:
```
┌─────────────────────────────────────┐
│ 工具调用分类                          │
├─────────────────────────────────────┤
│ 查询操作 → 自动批准                   │
│   - read_file                       │
│   - get_output                      │
│   - list_directory                  │
│   - search_code                     │
│   - git_status                      │
│   - git_diff                        │
├─────────────────────────────────────┤
│ 修改操作 → 先执行，后确认（可撤销）     │
│   - edit_file                       │
│   - rename_symbol                   │
│   - delete_file                     │
├─────────────────────────────────────┤
│ 危险操作 → 提前批准（不可撤销）        │
│   - run_command                     │
│   - send_input                      │
│   - kill_process                    │
│   - git_commit                      │
└─────────────────────────────────────┘
```

**批准语法**:
```
y       - 全部同意
n       - 全部拒绝
y 1,2,3 - 白名单（只同意1,2,3）
n 2,4   - 黑名单（拒绝2,4，其他同意）
```

**核心函数**:
- `HandleApproval()`: 处理立即批准
- `ConfirmModifyOperations()`: 确认修改操作
- `parseIndices()`: 解析序号列表

---

### 6. `internal/tools/` - 工具系统
**职责**: 定义和执行所有工具

#### 文件结构
```
tools/
├── definitions.go   # 15个工具的OpenAI定义
├── executor.go      # 工具路由和执行
├── file.go         # 文件操作（4个工具）
├── command.go      # 命令执行（4个工具）
├── search.go       # 代码搜索（2个工具）
├── project.go      # 项目分析（3个工具）
└── git.go          # Git工具（3个工具）
```

#### 工具分类表

| 类别 | 工具名 | 需批准 | 可撤销 |
|------|--------|--------|--------|
| 文件 | read_file | ❌ | - |
| 文件 | edit_file | ✅ | ✅ |
| 文件 | rename_symbol | ✅ | ✅ |
| 文件 | delete_file | ✅ | ✅ |
| 命令 | run_command | ✅ | ❌ |
| 命令 | send_input | ✅ | ❌ |
| 命令 | get_output | ❌ | - |
| 命令 | kill_process | ✅ | ❌ |
| 搜索 | search_code | ❌ | - |
| 搜索 | find_symbol | ❌ | - |
| 项目 | list_directory | ❌ | - |
| 项目 | get_project_structure | ❌ | - |
| 项目 | get_file_stats | ❌ | - |
| Git | git_status | ❌ | - |
| Git | git_diff | ❌ | - |
| Git | git_commit | ✅ | ❌ |

#### Executor模式
```go
type Executor struct {
    ProcessManager *process.Manager  // 依赖注入
    BackupManager  *backup.Manager   // 依赖注入
}

func (e *Executor) Execute(toolCall) string {
    // 根据工具名路由到对应的执行函数
}
```

#### 特色功能: AST重命名
```go
// rename_symbol 对 .go 文件使用AST
fset := token.NewFileSet()
node, _ := parser.ParseFile(fset, file, nil, parser.ParseComments)
ast.Inspect(node, func(n ast.Node) bool {
    if ident, ok := n.(*ast.Ident); ok {
        if ident.Name == oldSymbol {
            ident.Name = newSymbol
        }
    }
    return true
})
```

---

### 7. `internal/environment/` - 环境检测
**职责**: 自动检测系统环境

**检测项**:
```go
type SystemEnvironment struct {
    OS            string  // windows/linux/darwin
    Shell         string  // powershell/bash
    PythonCommand string  // python3/python/none
    HasGrep       bool    // grep工具
    HasTree       bool    // tree工具
    HasGit        bool    // git工具
}
```

**检测方法**:
- 运行时检测: `runtime.GOOS`
- 命令检测: `exec.Command("python3", "--version")`
- which/where查找

**用途**:
- 动态生成系统提示词
- 调整命令建议
- 避免使用不存在的工具

---

### 8. `internal/prompt/` - 系统提示词
**职责**: 根据环境生成系统提示

**示例**:
```
你是一个强大的AI编程助手...

## 当前系统环境 (系统自检)

**操作系统** (自检): windows
**Shell** (自检): powershell
**Python命令** (自检): python3 ✅ 已安装
**Git** (自检): ✅ 可用

## 命令使用指南 (根据系统自检结果)

### 当前是Windows系统，请使用以下命令：
- 列出文件: `dir` 或 `Get-ChildItem` (不要用ls)
- 搜索文本: `findstr` (不要用grep)
...
```

**关键点**:
- 明确标注"(自检)"，强调这是事实
- 提供正确命令，禁止错误命令
- 跨平台适配

---

### 9. `internal/history/` - 历史管理
**职责**: 对话历史的加载、保存和转换

**核心类型**:
```go
type Message struct {
    Role       string            // user/assistant/tool/system
    Content    string            // 消息内容
    ToolCalls  []ToolCall        // 工具调用
    ToolCallID string            // 工具调用ID
}
```

**功能**:
- `Load()`: 从JSON加载历史，更新system prompt
- `Save()`: 保存历史到JSON
- `ConvertToOpenAI()`: 转换为OpenAI格式

**设计理念**:
- system prompt每次启动时更新（适配当前环境）
- 历史持久化（跨会话上下文）

---

### 10. `internal/ui/` - 界面美化（已实现）
**职责**: 终端界面美化

**已实现功能**:
- `PrintWelcome()`: 彩色欢迎信息
- `PrintUserPrompt()`: 带颜色的用户输入提示 `[U]`
- `PrintAIPrompt()`: 带颜色的AI输出提示 `[A]`
- `PrintToolResult()`: 彩色工具结果（自动替换emoji）
- `PrintGoodbye()`: 再见信息
- `StartToolExecution()`: **启动Spinner动画**
- `ToolSpinner.Success()`: **显示成功结果**
- `ToolSpinner.Error()`: **显示错误结果**

**核心特性**:

#### 1. Spinner动画
```go
spinner := ui.StartToolExecution("read_file")
// ... 执行工具 ...
spinner.Success(result)  // 或 spinner.Error(result)
```

工具执行时显示旋转动画：
```
[>] ⠋ Executing: read_file...
```

完成后显示结果：
```
[✓] read_file: [FILE] Content:...
```

#### 2. 彩色输出
```go
colorSuccess = color.New(color.FgGreen, color.Bold)  // 成功
colorError   = color.New(color.FgRed, color.Bold)    // 错误
colorWarning = color.New(color.FgYellow, color.Bold) // 警告
colorInfo    = color.New(color.FgBlue)               // 信息
colorUser    = color.New(color.FgGreen, color.Bold)  // 用户
colorAI      = color.New(color.FgCyan, color.Bold)   // AI
colorTool    = color.New(color.FgMagenta)            // 工具
```

#### 3. 兼容性符号
不使用emoji，避免Windows/Linux乱码：
```go
const (
    SymbolSuccess = "[✓]"  // 成功
    SymbolError   = "[✗]"  // 失败
    SymbolInfo    = "[i]"  // 信息
    SymbolWarning = "[!]"  // 警告
    SymbolTool    = "[>]"  // 工具
    SymbolUser    = "[U]"  // 用户
    SymbolAI      = "[A]"  // AI
)
```

#### 4. Emoji替换
自动将所有工具返回的emoji替换为符号：
- `❌` → `[✗]`
- `✅` → `[✓]`
- `📄` → `[FILE]`
- `🔍` → `[SEARCH]`
- `📁` → `[DIR]`
- 等等...

**预留扩展**:
- 进度条 (schollz/progressbar)
- 表格展示 (olekukonko/tablewriter)
- Markdown渲染 (charmbracelet/glamour)
- 交互菜单 (manifoldco/promptui)

**主题系统**:
```go
type Theme struct {
    PrimaryColor   string  // 预留用于主题切换
    ErrorColor     string
    SuccessColor   string
    ...
}
```

---

## 🔄 数据流

### 完整请求流程
```
1. 用户输入
   │
2. 添加到messages → history.Save()
   │
3. 调用API (流式)
   │
   ├─→ 内容输出（实时打印）
   │
   └─→ 工具调用
       │
4. 分类工具（approval包）
   ├─→ 查询操作: 自动批准
   ├─→ 修改操作: 自动批准（记录到backup）
   └─→ 危险操作: 用户批准
       │
5. 执行工具（executor）
   │
6. 工具结果 → messages → history.Save()
   │
7. 继续API循环（如果还有工具调用）
   │
8. 确认修改操作（approval包）
   ├─→ 用户确认: CommitAll()
   └─→ 用户拒绝: UndoOperation()
```

### 工具执行流程
```
ToolCall
   ↓
Executor.Execute()
   ↓
switch toolName
   ├─→ file.go    (文件操作)
   ├─→ command.go (命令执行) ← 依赖 ProcessManager
   ├─→ search.go  (代码搜索)
   ├─→ project.go (项目分析)
   └─→ git.go     (Git操作)
   ↓
返回结果字符串
```

### 备份恢复流程
```
edit_file / rename_symbol / delete_file
   ↓
读取原文件内容
   ↓
执行修改
   ↓
backup.AddBackup(toolCallID, type, path, oldContent)
   ↓
等待AI回复完成
   ↓
approval.ConfirmModifyOperations()
   ├─→ 用户确认: backup.CommitAll() (清空)
   └─→ 用户拒绝: backup.UndoOperation() (恢复原内容)
```

---

## 🎯 设计原则

### 1. 单一职责原则
每个包只负责一个核心功能：
- `process`: 只管进程
- `backup`: 只管备份
- `approval`: 只管批准
- ...

### 2. 依赖注入
不使用全局变量，通过参数传递依赖：
```go
// ❌ 不好
var globalProcessManager *process.Manager

// ✅ 好
executor := tools.NewExecutor(processManager, backupManager)
```

### 3. 接口隔离
每个工具执行函数接收必要的依赖：
```go
ExecuteReadFile(args)                              // 无依赖
ExecuteEditFile(toolCallID, args, backupManager)   // 需要备份
ExecuteRunCommand(args, processManager)            // 需要进程管理
```

### 4. 开闭原则
- 添加新工具：只需修改 `tools/` 包
- 修改批准逻辑：只需修改 `approval/` 包
- 添加UI美化：只需修改 `ui/` 包

### 5. 关注点分离
- `main.go`: 只负责组装，不包含业务逻辑
- 业务逻辑分散在各个包中
- 每个包可独立测试

---

## 🧪 测试策略

### 单元测试
```go
// internal/backup/backup_test.go
func TestAddBackup(t *testing.T) {
    bm := NewManager()
    bm.AddBackup("id1", "edit", "test.go", []byte("old"))
    // 断言
}
```

### 集成测试
```go
// internal/tools/file_test.go
func TestEditFile(t *testing.T) {
    bm := backup.NewManager()
    result := ExecuteEditFile("id", args, bm)
    // 验证文件修改和备份
}
```

### 模拟测试
```go
// 可以mock ProcessManager, BackupManager
type MockProcessManager struct{}
func (m *MockProcessManager) StartProcess(cmd string) (string, error) {
    return "mock-id", nil
}
```

---

## 🚀 扩展建议

### 1. 添加配置文件支持
```yaml
# config.yaml
api:
  key: sk-xxx
  base_url: https://api.deepseek.com/v1
  model: deepseek-chat
  
ui:
  theme: dark
  enable_color: true
```

### 2. 添加插件系统
```go
type Plugin interface {
    Name() string
    Tools() []openai.Tool
    Execute(toolCall openai.ToolCall) string
}

// 用户可以编写自己的插件
```

### 3. 添加Web界面
```
internal/web/
├── server.go       # HTTP服务器
├── websocket.go    # WebSocket实时通信
└── templates/      # HTML模板
```

### 4. 添加多模型支持
```go
type ModelProvider interface {
    Chat(messages []Message) (Response, error)
}

type DeepSeekProvider struct{}
type OpenAIProvider struct{}
type ClaudeProvider struct{}
```

### 5. 添加RAG支持
```
internal/rag/
├── vector_store.go  # 向量存储
├── embeddings.go    # 文本嵌入
└── retrieval.go     # 检索增强
```

---

## 📊 性能优化

### 1. 并发优化
```go
// 并发执行多个查询工具
var wg sync.WaitGroup
for _, tc := range queryTools {
    wg.Add(1)
    go func(tc ToolCall) {
        defer wg.Done()
        Execute(tc)
    }(tc)
}
wg.Wait()
```

### 2. 缓存优化
```go
type CachedExecutor struct {
    cache map[string]string  // 工具结果缓存
    ttl   time.Duration
}
```

### 3. 流式输出优化
- 已实现：API流式响应
- 可优化：工具执行也可以流式（如grep大文件）

---

## 🔒 安全考虑

### 1. 命令注入防护
```go
// ❌ 危险
exec.Command("bash", "-c", userInput)

// ✅ 安全
// 在approval阶段显示完整命令，用户确认
```

### 2. 文件访问限制
```go
// 可以添加白名单
allowedPaths := []string{"/project", "/workspace"}
```

### 3. API密钥保护
```go
// 未来: 从环境变量读取
apiKey := os.Getenv("DEEPSEEK_API_KEY")
```

---

## 📈 未来规划

- [ ] 支持配置文件
- [ ] 插件系统
- [ ] Web界面
- [ ] 多模型支持
- [ ] RAG检索增强
- [ ] Docker部署
- [ ] 测试覆盖率 >80%
- [ ] 性能监控
- [ ] 日志系统
- [ ] 错误恢复机制
