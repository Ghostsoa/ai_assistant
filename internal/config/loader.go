package config

import (
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
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		// 首次运行，引导用户创建配置
		fmt.Println()
		fmt.Println("╔═══════════════════════════════════════════════════════╗")
		fmt.Println("║         欢迎使用 J.A.R.V.I.S - 超级人工智能          ║")
		fmt.Println("╚═══════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("检测到首次运行，请进行初始配置：")
		fmt.Println()

		if err := setupConfig(); err != nil {
			return fmt.Errorf("配置设置失败: %v", err)
		}

		fmt.Println()
		fmt.Printf("[✓] 配置已保存到: %s\n", ConfigFile)
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
	}

	return nil
}

// setupConfig 引导用户创建配置
func setupConfig() error {
	config := defaultConfig

	// 1. API Key
	fmt.Print("请输入您的 DeepSeek API Key: ")
	var apiKey string
	fmt.Scanln(&apiKey)
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	config.APIKey = apiKey

	// 2. 选择模型
	fmt.Println()
	fmt.Println("请选择模型:")
	fmt.Println("  1. deepseek-chat (标准模型)")
	fmt.Println("  2. deepseek-reasoner (支持思维链)")
	fmt.Print("选择 [1/2，默认 2]: ")

	var modelChoice string
	fmt.Scanln(&modelChoice)
	modelChoice = strings.TrimSpace(modelChoice)

	if modelChoice == "1" {
		config.Model = "deepseek-chat"
	} else {
		config.Model = "deepseek-reasoner"
	}

	// 3. 思维链显示模式
	fmt.Println()
	fmt.Println("思维链显示模式:")
	fmt.Println("  1. 每次询问 (ask)")
	fmt.Println("  2. 始终显示 (show)")
	fmt.Println("  3. 始终隐藏 (hide)")
	fmt.Print("选择 [1/2/3，默认 1]: ")

	var reasoningChoice string
	fmt.Scanln(&reasoningChoice)
	reasoningChoice = strings.TrimSpace(reasoningChoice)

	switch reasoningChoice {
	case "2":
		config.ReasoningMode = "show"
	case "3":
		config.ReasoningMode = "hide"
	default:
		config.ReasoningMode = "ask"
	}

	// 4. 历史记录轮数
	fmt.Println()
	fmt.Print("历史记录保留轮数 [默认 60，0 表示全部]: ")

	var rounds string
	fmt.Scanln(&rounds)
	rounds = strings.TrimSpace(rounds)

	if rounds != "" {
		var r int
		if _, err := fmt.Sscanf(rounds, "%d", &r); err == nil && r >= 0 {
			config.MaxHistoryRounds = r
		}
	}

	// 5. 使用默认的其他配置
	config.BaseURL = "https://api.deepseek.com/v1"
	config.InterruptKey = "n"
	config.EnableInterrupt = true

	// 保存配置
	GlobalConfig = config
	return SaveConfig()
}

// createDefaultConfig 创建默认配置文件（现已不直接使用）
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
