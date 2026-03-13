package ui

import (
	"os"
	"strings"
)

// Theme 主题类型
type Theme string

const (
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

// DetectTheme 检测终端主题
func DetectTheme() Theme {
	// 首先检查环境变量覆盖
	themeEnv := os.Getenv("GOPENCLAW_THEME")
	themeEnv = strings.ToLower(themeEnv)
	if themeEnv == "light" {
		return ThemeLight
	}
	if themeEnv == "dark" {
		return ThemeDark
	}

	// 检测 COLORFGBG 环境变量
	colorfgbg := os.Getenv("COLORFGBG")
	if colorfgbg != "" {
		// COLORFGBG 格式: "15;2" (前景;背景) 或 "7;0" (反色)
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			// 背景色 0-7 是深色，8-15 是浅色
			// 但更准确的是检测是否有 ; 表示有背景色设置
			// 如果 COLORFGBG 包含 ; 说明是彩色终端
			// 通常 COLORFGBG=7 是白色背景（终端模拟器浅色）
			// COLORFGBG=0 是黑色背景（终端模拟器深色）
			bg := strings.TrimSpace(parts[len(parts)-1])
			if bg == "7" || bg == "15" {
				return ThemeLight
			}
		}
		// 简单的 "7" 表示白色背景
		if colorfgbg == "7" || strings.HasSuffix(colorfgbg, ";7") {
			return ThemeLight
		}
	}

	// 检测 TERM 环境变量
	term := os.Getenv("TERM")
	if strings.Contains(term, "light") {
		return ThemeLight
	}

	// 默认返回深色主题
	return ThemeDark
}

// IsLightTheme 返回是否为浅色主题
func IsLightTheme() bool {
	return DetectTheme() == ThemeLight
}
