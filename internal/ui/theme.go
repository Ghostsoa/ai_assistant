package ui

// Theme 主题配置（预留）
type Theme struct {
	PrimaryColor   string
	SecondaryColor string
	ErrorColor     string
	SuccessColor   string
	WarningColor   string
}

// DefaultTheme 默认主题
var DefaultTheme = Theme{
	PrimaryColor:   "#00BFFF", // 深天蓝
	SecondaryColor: "#32CD32", // 酸橙绿
	ErrorColor:     "#FF4444", // 红色
	SuccessColor:   "#00FF00", // 绿色
	WarningColor:   "#FFA500", // 橙色
}

// TODO: 未来可以实现主题切换功能
// - 支持多种预设主题（暗色/亮色）
// - 支持自定义主题
// - 支持从配置文件加载主题
// - 使用 fatih/color 或 gookit/color 实现彩色输出
