package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
		// 创建默认配置文件
		if err := createDefaultConfig(); err != nil {
			return fmt.Errorf("创建默认配置失败: %v", err)
		}
		fmt.Printf("[初始化] 已创建配置文件: %s\n", ConfigFile)
		fmt.Println("[提示] 请编辑配置文件，填入您的 API Key")
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
		fmt.Printf("[初始化] 已创建历史记录文件: %s\n", HistoryFile)
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
