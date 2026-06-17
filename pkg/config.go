// tools/desktop/pkg/config.go
// 配置类型定义 + 解析逻辑
package pkg

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ———— 配置结构 ————

type Config struct {
	App      AppConfig       `yaml:"app"`
	Security SecurityConfig  `yaml:"security"`
	Window   WindowConfig    `yaml:"window"`
	Projects []ProjectConfig `yaml:"projects"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type SecurityConfig struct {
	AESKey string `yaml:"aes_key"`
}

type WindowConfig struct {
	Title                   string            `yaml:"title"`
	Width                   int               `yaml:"width"`
	Height                  int               `yaml:"height"`
	Fullscreen              bool              `yaml:"fullscreen"`
	Maximized               bool              `yaml:"maximized"`
	Borderless              bool              `yaml:"borderless"`
	AlwaysOnTop             bool              `yaml:"always_on_top"`
	Opacity                 float64           `yaml:"opacity"`
	WebViewBgTransparent    bool              `yaml:"webview_bg_transparent"`
	InputPassthrough        bool              `yaml:"input_passthrough"` // 透明区域点击穿透
	SystemTray              bool              `yaml:"system_tray"`       // 系统托盘：关闭时隐藏到托盘
	TrayHideTaskbar         bool              `yaml:"tray_hide_taskbar"` // 托盘模式下隐藏任务栏图标
	WindowPosition          string            `yaml:"window_position"`
	DarkTitleBar            bool              `yaml:"dark_title_bar"`
	RoundCorners            bool              `yaml:"round_corners"`
	Acrylic                 bool              `yaml:"acrylic"`
	KeyboardShortcuts       bool              `yaml:"keyboard_shortcuts"`
	DisableBrowserShortcuts bool              `yaml:"disable_browser_shortcuts"`
	DefaultBlockedShortcuts []string          `yaml:"default_blocked_shortcuts"`
	KeyMappings             map[string]string `yaml:"key_mappings"`
}

type ProjectConfig struct {
	Name        string `yaml:"name"`
	VueDir      string `yaml:"vue_dir"`
	Description string `yaml:"description"`
}

// AppCfg 全局配置实例
var AppCfg Config

// InitConfig 解析嵌入的 config.yaml 数据。
func InitConfig(data []byte) error {
	if err := yaml.Unmarshal(data, &AppCfg); err != nil {
		return fmt.Errorf("解析 config.yaml 失败: %w", err)
	}
	if AppCfg.Security.AESKey == "" {
		return fmt.Errorf("config.yaml: security.aes_key 不能为空")
	}
	if AppCfg.Window.Opacity == 0 {
		AppCfg.Window.Opacity = 1.0
	}
	if AppCfg.Window.Width == 0 {
		AppCfg.Window.Width = 1024
	}
	if AppCfg.Window.Height == 0 {
		AppCfg.Window.Height = 768
	}
	return nil
}

// WindowConfigData 窗口配置持久化数据结构
type WindowConfigData struct {
	Title                string  `json:"title"`
	Width                int     `json:"width"`
	Height               int     `json:"height"`
	Fullscreen           bool    `json:"fullscreen"`
	Maximized            bool    `json:"maximized"`
	Borderless           bool    `json:"borderless"`
	AlwaysOnTop          bool    `json:"always_on_top"`
	Opacity              float64 `json:"opacity"`
	WebViewBgTransparent bool    `json:"webview_bg_transparent"`
	InputPassthrough     bool    `json:"input_passthrough"`
	SystemTray           bool    `json:"system_tray"`
	TrayHideTaskbar      bool    `json:"tray_hide_taskbar"`
	WindowPosition       string  `json:"window_position"`
	DarkTitleBar         bool    `json:"dark_title_bar"`
	RoundCorners         bool    `json:"round_corners"`
	Acrylic              bool    `json:"acrylic"`
}

// UpdateAppWindowConfig 用持久化配置覆盖全局 AppCfg.Window。
// 注意：KeyboardShortcuts 是构建时配置，不从此处覆盖。
func UpdateAppWindowConfig(cfg *WindowConfigData) {
	if cfg.Title != "" {
		AppCfg.Window.Title = cfg.Title
	}
	AppCfg.Window.Width = cfg.Width
	AppCfg.Window.Height = cfg.Height
	AppCfg.Window.Fullscreen = cfg.Fullscreen
	AppCfg.Window.Maximized = cfg.Maximized
	AppCfg.Window.Borderless = cfg.Borderless
	AppCfg.Window.AlwaysOnTop = cfg.AlwaysOnTop
	AppCfg.Window.Opacity = cfg.Opacity
	AppCfg.Window.WebViewBgTransparent = cfg.WebViewBgTransparent
	AppCfg.Window.InputPassthrough = cfg.InputPassthrough
	AppCfg.Window.SystemTray = cfg.SystemTray
	AppCfg.Window.TrayHideTaskbar = cfg.TrayHideTaskbar
	AppCfg.Window.WindowPosition = cfg.WindowPosition
	AppCfg.Window.DarkTitleBar = cfg.DarkTitleBar
	AppCfg.Window.RoundCorners = cfg.RoundCorners
	AppCfg.Window.Acrylic = cfg.Acrylic
	// KeyboardShortcuts 不从此处覆盖，保持 config.yaml 的值
}
