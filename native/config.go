// tools/desktop/native/config.go
// 统一窗口外观配置接口 — 所有平台通用

package native

// WindowConfig 窗口外观配置。
// 在 webview.go 中统一设置，各平台 Apply() 选择实现支持的效果。
type WindowConfig struct {
	// 窗口透明度 (0.0 全透明 ~ 1.0 不透明)，默认 1.0
	Opacity float64

	// WebView 控件背景透明
	WebViewBgTransparent bool

	// 透明区域点击穿透：false=本窗口捕获所有点击，true=透明区域点击透传到下层
	InputPassthrough bool

	// 系统托盘模式：关闭窗口时隐藏到托盘图标
	SystemTray bool

	// 托盘模式下隐藏任务栏图标（WS_EX_TOOLWINDOW），默认 true
	TrayHideTaskbar bool

	// 是否移除标题栏和边框（无边框窗口）
	Borderless bool

	// 是否始终置顶
	AlwaysOnTop bool

	// 全屏模式（隐藏任务栏和标题栏，覆盖整个屏幕）
	Fullscreen bool

	// 窗口最大化（保留任务栏和标题栏）
	Maximized bool

	// 窗口位置（仅非最大化/非全屏时生效）：""=默认 "center"=居中 "x,y"=绝对坐标
	WindowPosition string

	// ———— Windows 专属 ————

	// Acrylic 毛玻璃背景效果 (Win10+)
	Acrylic bool

	// 暗色标题栏 (Win10 1809+)
	DarkTitleBar bool

	// 窗口圆角 (Win11)
	RoundCorners bool

	// 键盘快捷键拦截：启用后安装低层键盘钩子
	KeyboardShortcuts bool

	// 默认拦截的快捷键列表：钩子安装时自动注册
	// 设为空列表则不拦截任何默认快捷键（但钩子仍安装，可用于按键映射场景）
	DefaultBlockedShortcuts []string

	// 按键映射：物理按键名 → 映射名
	// 映射后的按键被钩子拦截，不会传递到系统，同时以映射名触发事件
	KeyMappings map[string]string
}

// DefaultConfig 默认配置（不改变窗口外观）
func DefaultConfig() WindowConfig {
	return WindowConfig{
		Opacity: 1.0,
	}
}
