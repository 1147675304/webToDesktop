// tools/desktop/pkg/bridge/window.go
// Go ↔ JS 桥接 — 原生窗口控制
//
// 所有窗口操作方法必须在 UI 线程执行，因此通过 wv.Dispatch() 调度。
// 各平台实现位于 native/ 包：
//   - native/windows.go  Win32 API (SendMessage / SetWindowPos)
//   - native/linux.go    GTK3 API (gtk_window_begin_move_drag 等)
//   - native/other.go    空实现（非 Windows/Linux 平台降级）
//
// JS 调用示例:
//
//	window.__lhpanda__('dragWindow')
//	window.__lhpanda__('resizeWindow', {edge: 11})  // 11 = 右边缘
//	window.__lhpanda__('toggleMaximize')
//	window.__lhpanda__('closeWindow')
package bridge

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lhpanda/webtodesktop/native"
)

// handleDragWindow 触发窗口拖拽，模拟在标题栏按下鼠标左键。
//
// JS 调用: window.__lhpanda__('dragWindow')
// 返回:    {ok: true}
//
// 实现:
//
//	Windows → ReleaseCapture() + SendMessage(WM_NCLBUTTONDOWN, HT_CAPTION)
//	Linux   → gtk_window_begin_move_drag()
func (b *Bridge) HandleDragWindow(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.DragWindow(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// handleCloseWindow 关闭应用程序窗口。
//
// JS 调用: window.__lhpanda__('closeWindow')
// 返回:    {ok: true}
//
// 实现:
//
//	Windows → PostMessage(WM_CLOSE)
//	Linux   → gtk_window_close()
func (b *Bridge) HandleCloseWindow(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.CloseWindow(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// handleToggleMaximize 窗口最大化/还原切换。
//
// JS 调用: window.__lhpanda__('toggleMaximize')
// 返回:    {ok: true}
//
// 行为:
//   - 当前为普通状态 → 最大化到工作区（保留任务栏）
//   - 当前已最大化    → 还原到配置文件指定的 g_configWidth × g_configHeight 并居中
//   - 还原时不记忆用户手动拖拽的尺寸，始终使用配置文件的宽高
func (b *Bridge) HandleToggleMaximize(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.ToggleMaximize(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// handleToggleFullscreen 窗口全屏/退出全屏切换。
//
// JS 调用: window.__lhpanda__('toggleFullscreen')
// 返回:    {ok: true}
//
// 行为:
//   - 进入全屏 → 移除边框样式，覆盖整个显示器（含任务栏区域）
//   - 退出全屏 → 恢复边框样式，还原到配置文件尺寸并居中
func (b *Bridge) HandleToggleFullscreen(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.ToggleFullscreen(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// handleToggleMinimize 最小化窗口到任务栏。
//
// JS 调用: window.__lhpanda__('toggleMinimize')
// 返回:    {ok: true}
//
// 实现:
//
//	Windows → ShowWindow(hwnd, SW_MINIMIZE)
//	Linux   → gtk_window_iconify()
func (b *Bridge) HandleToggleMinimize(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.ToggleMinimize(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// HandleShowWindow 显示并置前窗口（从托盘恢复）。
//
// JS 调用: window.__lhpanda__('showWindow')
// 返回:    {ok: true}
//
// 实现:
//
//	Windows → ShowWindow(hwnd, SW_SHOW) + SetForegroundWindow
//	Linux   → gtk_window_deiconify + gtk_window_present
func (b *Bridge) HandleShowWindow(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.ShowWindowRestore(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// handleResizeWindow 从指定窗口边缘/角落触发系统级窗口缩放。
//
// JS 调用: window.__lhpanda__('resizeWindow', {edge: 11})
// 返回:    {ok: true}
//
// edge 参数对应 Win32 HTxxx 非客户区命中测试常量:
//
//	10 = HTLEFT        左边框
//	11 = HTRIGHT       右边框
//	12 = HTTOP         上边框
//	13 = HTTOPLEFT     左上角
//	14 = HTTOPRIGHT    右上角
//	15 = HTBOTTOM      下边框
//	16 = HTBOTTOMLEFT  左下角
//	17 = HTBOTTOMRIGHT 右下角
//
// 实现:
//
//	Windows → 临时加回 WS_THICKFRAME → ReleaseCapture + SendMessage(WM_NCLBUTTONDOWN, edge)
//	Linux   → gtk_window_begin_resize_drag(edge)
func (b *Bridge) HandleResizeWindow(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	edge, ok := params["edge"].(float64)
	if !ok {
		return nil, fmt.Errorf("缺少 edge 参数")
	}
	b.wv.Dispatch(func() { native.ResizeWindow(b.winPtr, int(edge)) })
	return map[string]interface{}{"ok": true}, nil
}

// HandleRestartApp 重启应用程序。
//
// JS 调用: window.__lhpanda__('restartApp')
// 返回:    {ok: true}
//
// 实现:
//  1. 获取当前可执行文件路径
//  2. 启动新进程（与当前进程参数相同，标准输出/错误继承）
//  3. 关闭当前窗口，触发应用退出
//
// 注意:
//   - 新进程与当前进程独立，当前进程退出不影响新进程运行
//   - 如果在浏览器模拟模式下调用，会尝试通过 location.reload() 刷新页面
func (b *Bridge) HandleRestartApp(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}

	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("无法获取可执行文件路径: %w", err)
	}

	// 启动新进程，与当前进程参数相同
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动新进程失败: %w", err)
	}

	// 新进程已启动，关闭当前窗口
	b.wv.Dispatch(func() { native.CloseWindow(b.winPtr) })

	return map[string]interface{}{"ok": true}, nil
}

// handleGetCursorPos JS 每帧调用，获取鼠标在窗口客户区内的坐标。
func (b *Bridge) HandleGetCursorPos(params map[string]interface{}) (interface{}, error) {
	x, y := native.PollCursorPos(b.winPtr)
	return map[string]interface{}{"x": x, "y": y}, nil
}

// HandleSetInputPassthrough 启用/禁用输入穿透（鼠标事件穿过透明区域）。
//
// JS 调用: window.__lhpanda__('setInputPassthrough', {enabled: true})
// 返回:    {ok: true, enabled: true/false}
//
// 参数:
//
//	enabled (bool): true=启用穿透（鼠标可穿过本窗口操作下层），false=禁用穿透（正常捕获）
//
// 实现:
//
//	Windows → WS_EX_TRANSPARENT|WS_EX_LAYERED 全局穿透
//	Linux   → 默认穿透，禁用时调用 setInputShapeFull 设置全窗口输入形状
//
// 注意:
//   - WS_EX_TRANSPARENT 使整个窗口穿透（包括不透明区域），
//     如需"仅透明区域穿透"，需配合前端 JS 检测透明度并动态调用此方法
func (b *Bridge) HandleSetInputPassthrough(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	enabled, ok := params["enabled"].(bool)
	if !ok {
		return nil, fmt.Errorf("缺少 enabled 参数（bool）")
	}
	if b.lastPassthroughValid && enabled == b.lastPassthroughState {
		return map[string]interface{}{"ok": true, "enabled": enabled, "changed": false}, nil
	}
	b.lastPassthroughState = enabled
	b.lastPassthroughValid = true
	if enabled {
		b.wv.Dispatch(func() { native.EnableInputPassthrough(b.winPtr) })
	} else {
		b.wv.Dispatch(func() { native.DisableInputPassthrough(b.winPtr) })
	}
	return map[string]interface{}{"ok": true, "enabled": enabled, "changed": true}, nil
}

// HandleGetInputPassthrough 查询当前是否启用输入穿透。
//
// JS 调用: window.__lhpanda__('getInputPassthrough')
// 返回:    {ok: true, enabled: true/false}
func (b *Bridge) HandleGetInputPassthrough(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	enabled := native.IsInputPassthrough(b.winPtr)
	return map[string]interface{}{"ok": true, "enabled": enabled}, nil
}

// HandleSetOpaqueRegions 设置不透光区域列表（精准穿透模式）。
//
//	JS 调用: window.__lhpanda__('setOpaqueRegions', {
//	    regions: [{x:100, y:200, w:300, h:50}, {x:500, y:400, w:200, h:60}]
//	})
//
// 返回:    {ok: true, count: 2}
//
// 参数:
//
//	regions (array): 不透光矩形数组，每个矩形为 {x, y, w, h}（相对于窗口客户区原点）
//
// 工作原理:
//
//  1. 移除 WS_EX_TRANSPARENT（全局穿透）
//  2. 将矩形列表存入 C 层静态数组
//  3. WM_NCHITTEST 中判断命中坐标：
//     - 在不透光区域内 → HTCLIENT（正常捕获鼠标事件）
//     - 不在不透光区域内 → HTTRANSPARENT（穿透到下层窗口）
//
// 适用场景:
//   - 画布背景透明但上面有按钮/卡片等可交互元素
//   - 需要"部分区域不穿透"的页面
//
// JS 侧获取不透光区域的方式:
//  1. 遍历 DOM 中所有 pointer-events:auto 的元素，获取 getBoundingClientRect()
//  2. 逐像素检测 getImageData() 的 Alpha 通道（性能开销大，不推荐）
//  3. 预定义 CSS 类标记不透光区域，JS 收集其坐标
func (b *Bridge) HandleSetOpaqueRegions(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}

	rawRegions, ok := params["regions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("缺少 regions 参数（数组）")
	}

	regions := make([]native.OpaqueRegion, 0, len(rawRegions))
	for _, item := range rawRegions {
		r, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		x, _ := toInt(r["x"])
		y, _ := toInt(r["y"])
		w, _ := toInt(r["w"])
		h, _ := toInt(r["h"])
		if w > 0 && h > 0 {
			regions = append(regions, native.OpaqueRegion{X: x, Y: y, W: w, H: h})
		}
	}

	b.wv.Dispatch(func() { native.SetOpaqueRegions(b.winPtr, regions) })
	return map[string]interface{}{"ok": true, "count": len(regions)}, nil
}

// HandleClearOpaqueRegions 清空不透光区域列表。
//
// JS 调用: window.__lhpanda__('clearOpaqueRegions')
// 返回:    {ok: true}
//
// 清空后行为取决于 input_passthrough 配置：
//   - 若配置为 true → 回退到全局穿透模式（WS_EX_TRANSPARENT）
//   - 若配置为 false → 关闭所有穿透
func (b *Bridge) HandleClearOpaqueRegions(params map[string]interface{}) (interface{}, error) {
	if b.wv == nil || b.winPtr == nil {
		return nil, fmt.Errorf("窗口未就绪")
	}
	b.wv.Dispatch(func() { native.ClearOpaqueRegions(b.winPtr) })
	return map[string]interface{}{"ok": true}, nil
}

// toInt 将 interface{} 安全转为 int（支持 float64 和 int）。
func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case int64:
		return int(val), true
	default:
		return 0, false
	}
}
