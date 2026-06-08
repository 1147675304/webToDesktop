// tools/desktop/pkg/bridge/config.go
// Go ↔ JS 桥接 — 窗口配置持久化 + JS 参数提取辅助函数
//
// 窗口配置数据流:
//
//	用户修改偏好设置
//	  → JS: callBridge('saveWindowConfig', {...})
//	  → Go:  handleSaveWindowConfig → 校验 → Store.SaveWindowConfig → window_config.json
//	  → 同时更新内存中的 pkg.AppCfg.Window（部分设置需重启生效）
//
//	程序启动
//	  → main.go → Store.LoadWindowConfig → pkg.UpdateAppWindowConfig
//	  → 覆盖 config.yaml 的嵌入式默认值
//	  → pkg/webview.go RunApp 读取 AppCfg.Window 创建窗口
//
// JS 调用示例:
//
//	window.__lhpanda__('getWindowConfig')
//	→ {title: "后台管理系统", width: 1366, height: 768, ...}
//
//	window.__lhpanda__('saveWindowConfig', {
//	    title: "我的应用", width: 1280, height: 720,
//	    borderless: true, opacity: 0.95, ...
//	})
//	→ {saved: true, needRestart: true}
package bridge

import (
	"fmt"

	"github.com/lhpanda/webtodesktop/pkg"
)

// handleGetWindowConfig 返回当前内存中的窗口配置。
//
// JS 调用: window.__lhpanda__('getWindowConfig')
// 返回所有 13 个 window 配置字段（与 config.yaml window 节对应）:
//
//	title, width, height,
//	fullscreen, maximized,
//	borderless, always_on_top,
//	opacity, webview_bg_transparent,
//	window_position,
//	dark_title_bar, round_corners, acrylic
func (b *Bridge) HandleGetWindowConfig(params map[string]interface{}) (interface{}, error) {
	cfg := pkg.AppCfg.Window
	return map[string]interface{}{
		"title":                  cfg.Title,
		"width":                  cfg.Width,
		"height":                 cfg.Height,
		"fullscreen":             cfg.Fullscreen,
		"maximized":              cfg.Maximized,
		"borderless":             cfg.Borderless,
		"always_on_top":          cfg.AlwaysOnTop,
		"opacity":                cfg.Opacity,
		"webview_bg_transparent": cfg.WebViewBgTransparent,
		"input_passthrough":      cfg.InputPassthrough,
		"window_position":        cfg.WindowPosition,
		"dark_title_bar":         cfg.DarkTitleBar,
		"round_corners":          cfg.RoundCorners,
		"acrylic":                cfg.Acrylic,
	}, nil
}

// handleSaveWindowConfig 保存窗口配置到磁盘并更新内存。
//
// JS 调用: window.__lhpanda__('saveWindowConfig', {title: "xxx", width: 1280, ...})
// 返回:    {saved: true, needRestart: true}
//
// 处理流程:
//  1. 从 params 提取各字段（使用 getString/getInt/getFloat/getBool）
//  2. 基本校验: width≥300, height≥200, opacity∈[0.1,1.0], title 非空时用 AppName 兜底
//  3. 调用 Store.SaveWindowConfig 写入 JSON 文件
//  4. 调用 pkg.UpdateAppWindowConfig 更新内存配置（部分设置如边框/全屏需重启才能生效）
//
// 注意: 窗口尺寸、标题等基本设置在下次启动时生效；
// 透明度、置顶等设置立即生效（取决于 native.Apply 是否支持运行时修改）。
func (b *Bridge) HandleSaveWindowConfig(params map[string]interface{}) (interface{}, error) {
	if b.store == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	// 调试日志: 打印收到的所有参数
	fmt.Printf("[bridge] saveWindowConfig received %d keys:\n", len(params))
	for k, v := range params {
		fmt.Printf("  %s = %v (type: %T)\n", k, v, v)
	}

	cfg := &pkg.WindowConfigData{
		Title:                getString(params, "title"),
		Width:                getInt(params, "width"),
		Height:               getInt(params, "height"),
		Fullscreen:           getBool(params, "fullscreen"),
		Maximized:            getBool(params, "maximized"),
		Borderless:           getBool(params, "borderless"),
		AlwaysOnTop:          getBool(params, "always_on_top"),
		Opacity:              getFloat(params, "opacity"),
		WebViewBgTransparent: getBool(params, "webview_bg_transparent"),
		InputPassthrough:     getBool(params, "input_passthrough"),
		WindowPosition:       getString(params, "window_position"),
		DarkTitleBar:         getBool(params, "dark_title_bar"),
		RoundCorners:         getBool(params, "round_corners"),
		Acrylic:              getBool(params, "acrylic"),
	}

	// 基本校验
	if cfg.Width < 300 {
		cfg.Width = 300
	}
	if cfg.Height < 200 {
		cfg.Height = 200
	}
	if cfg.Opacity < 0.1 {
		cfg.Opacity = 0.1
	}
	if cfg.Opacity > 1.0 {
		cfg.Opacity = 1.0
	}
	if cfg.Title == "" {
		cfg.Title = pkg.AppCfg.App.Name
	}

	if err := b.store.SaveWindowConfig(cfg); err != nil {
		return nil, fmt.Errorf("保存窗口配置失败: %w", err)
	}
	pkg.UpdateAppWindowConfig(cfg)
	return map[string]interface{}{"saved": true, "needRestart": true}, nil
}

// ———— 辅助函数：从 JS 传入的 map[string]interface{} 中安全提取类型化参数 ————
//
// JS 通过 webview.Bind 传入的参数经过 JSON 序列化/反序列化，类型映射:
//
//	JS number  → Go float64（JSON 数字统一为 float64）
//	JS boolean → Go bool
//	JS string  → Go string
//	JS null    → Go nil
//
// 这些辅助函数处理了类型断言失败和缺失 key 的情况，返回零值而不 panic。

// getString 从 params 中提取 string 值，key 不存在或类型不匹配时返回 ""。
func getString(params map[string]interface{}, key string) string {
	v, _ := params[key].(string)
	return v
}

// getInt 从 params 中提取 int 值。
// JSON 数字反序列化为 float64，需要做 float64→int 转换。
// key 不存在或类型不匹配时返回 0。
func getInt(params map[string]interface{}, key string) int {
	switch v := params[key].(type) {
	case float64:
		return int(v) // JSON number → Go float64 → int
	case int:
		return v
	}
	return 0
}

// getFloat 从 params 中提取 float64 值，key 不存在或类型不匹配时返回 0。
func getFloat(params map[string]interface{}, key string) float64 {
	v, _ := params[key].(float64)
	return v
}

// getBool 从 params 中提取 bool 值，key 不存在或类型不匹配时返回 false。
//
// 注意: 由于 Go 零值 false 与 JS 传入 false 无法区分，
// 依赖此函数的业务逻辑应确保 JS 侧总是显式传入 boolean 值。
func getBool(params map[string]interface{}, key string) bool {
	v, _ := params[key].(bool)
	return v
}
