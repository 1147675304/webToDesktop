// tools/desktop/native/config.go
// 统一窗口外观配置接口 — 所有平台通用

package native

// WindowConfig 窗口外观配置。
// 在 webview.go 中统一设置，各平台 Apply() 选择实现支持的效果。
type WindowConfig struct {
	// 窗口透明度 (0.0 全透明 ~ 1.0 不透明)，默认 1.0
	Opacity float64

	// WebView2 控件背景透明（Windows 专用）
	WebViewBgTransparent bool

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
}

// DefaultConfig 默认配置（不改变窗口外观）
func DefaultConfig() WindowConfig {
	return WindowConfig{
		Opacity: 1.0,
	}
}
