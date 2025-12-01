package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config 配置结构
type Config struct {
	APIKey           string `json:"api_key"`
	BaseURL          string `json:"base_url"`
	Model            string `json:"model"`
	MaxHistoryRounds int    `json:"max_history_rounds"`
	InterruptKey     string `json:"interrupt_key"`
	EnableInterrupt  bool   `json:"enable_interrupt"`
	ReasoningMode    string `json:"reasoning_mode"` // "ask", "show", "hide"
}

// 默认配置
var defaultConfig = Config{
	APIKey:           "your-api-key-here",
	BaseURL:          "https://api.deepseek.com/v1",
	Model:            "deepseek-reasoner",
	MaxHistoryRounds: 60,
	InterruptKey:     "n",
	EnableInterrupt:  true,
	ReasoningMode:    "ask",
}

// 全局配置实例
var GlobalConfig Config

// 路径
var (
	ConfigDir   string
	ConfigFile  string
	HistoryFile string
)

// GetConfigDir 获取配置目录
func GetConfigDir() string {
	var baseDir string

	if runtime.GOOS == "windows" {
		// Windows: %APPDATA%/jarvis
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = os.Getenv("USERPROFILE")
		}
	} else {
		// Linux/Mac: ~/.config/jarvis
		baseDir = os.Getenv("HOME")
		if baseDir == "" {
			baseDir = "."
		}
		baseDir = filepath.Join(baseDir, ".config")
	}

	return filepath.Join(baseDir, "jarvis")
}

// Initialize 初始化配置
func Initialize() error {
	// 获取配置目录
	ConfigDir = GetConfigDir()
	ConfigFile = filepath.Join(ConfigDir, "config.json")
	HistoryFile = filepath.Join(ConfigDir, "chat_history.json")

	// 检查并创建配置目录
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 检查配置文件是否存在
	isFirstRun := false
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		isFirstRun = true
		// 首次运行，交互式创建配置
		if err := interactiveConfig(); err != nil {
			return fmt.Errorf("配置初始化失败: %v", err)
		}
	}

	// 加载配置
	if err := loadConfig(); err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 检查历史记录文件
	if _, err := os.Stat(HistoryFile); os.IsNotExist(err) {
		// 创建空的历史记录文件
		if err := createDefaultHistory(); err != nil {
			return fmt.Errorf("创建历史记录文件失败: %v", err)
		}
		if isFirstRun {
			fmt.Printf("[初始化] 已创建历史记录文件: %s\n", HistoryFile)
		}
	}

	return nil
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig() error {
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile, data, 0644)
}

// loadConfig 加载配置
func loadConfig() error {
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &GlobalConfig); err != nil {
		return err
	}

	// 验证必填项
	if GlobalConfig.APIKey == "" || GlobalConfig.APIKey == "your-api-key-here" {
		return fmt.Errorf("请在配置文件中设置有效的 API Key: %s", ConfigFile)
	}

	return nil
}

// createDefaultHistory 创建空的历史记录文件
func createDefaultHistory() error {
	emptyHistory := []interface{}{}
	data, err := json.MarshalIndent(emptyHistory, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(HistoryFile, data, 0644)
}

// SaveConfig 保存配置
func SaveConfig() error {
	data, err := json.MarshalIndent(GlobalConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile, data, 0644)
}

// GetHistoryFile 获取历史记录文件路径
func GetHistoryFile() string {
	return HistoryFile
}

// interactiveConfig 交互式配置
func interactiveConfig() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                            ║")
	fmt.Println("║           欢迎使用 J.A.R.V.I.S - 超级人工智能             ║")
	fmt.Println("║        Just A Rather Very Intelligent System              ║")
	fmt.Println("║                                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("检测到这是首次运行，需要进行初始配置。")
	fmt.Println()

	// 配置项
	config := defaultConfig

	// 1. API Key (必填)
	fmt.Println("【1/4】API 密钥配置")
	fmt.Println("请输入您的 DeepSeek API Key：")
	fmt.Print("> ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	config.APIKey = apiKey

	// 2. API Base URL (可选)
	fmt.Println()
	fmt.Println("【2/4】API 地址配置")
	fmt.Printf("请输入 API Base URL (默认: %s)：\n", defaultConfig.BaseURL)
	fmt.Print("> ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	// 3. 模型选择
	fmt.Println()
	fmt.Println("【3/4】模型选择")
	fmt.Println("可用模型：")
	fmt.Println("  1. deepseek-chat      - 标准对话模型")
	fmt.Println("  2. deepseek-reasoner  - 思维链模型（推荐）")
	fmt.Printf("请选择模型 (默认: 2)：\n")
	fmt.Print("> ")
	modelChoice, _ := reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)

	switch modelChoice {
	case "1":
		config.Model = "deepseek-chat"
	case "2", "":
		config.Model = "deepseek-reasoner"
	default:
		config.Model = modelChoice // 允许自定义模型名
	}

	// 4. 思维链显示模式
	fmt.Println()
	fmt.Println("【4/4】思维链显示配置")
	fmt.Println("思维链显示模式：")
	fmt.Println("  1. ask  - 每次启动询问（推荐）")
	fmt.Println("  2. show - 始终显示")
	fmt.Println("  3. hide - 始终隐藏")
	fmt.Printf("请选择模式 (默认: 1)：\n")
	fmt.Print("> ")
	reasoningChoice, _ := reader.ReadString('\n')
	reasoningChoice = strings.TrimSpace(reasoningChoice)

	switch reasoningChoice {
	case "1", "":
		config.ReasoningMode = "ask"
	case "2":
		config.ReasoningMode = "show"
	case "3":
		config.ReasoningMode = "hide"
	default:
		config.ReasoningMode = "ask"
	}

	// 保存配置
	fmt.Println()
	fmt.Println("正在保存配置...")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(ConfigFile, data, 0644); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("✓ 配置已保存到: %s\n", ConfigFile)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("配置摘要：")
	fmt.Printf("  - API Key    : %s...%s\n", apiKey[:10], apiKey[len(apiKey)-4:])
	fmt.Printf("  - Base URL   : %s\n", config.BaseURL)
	fmt.Printf("  - 模型       : %s\n", config.Model)
	fmt.Printf("  - 思维链模式 : %s\n", config.ReasoningMode)
	fmt.Printf("  - 历史轮数   : %d\n", config.MaxHistoryRounds)
	fmt.Printf("  - 打断按键   : %s\n", config.InterruptKey)
	fmt.Println()
	fmt.Println("提示：可以随时编辑配置文件来修改这些设置")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	return nil
}
