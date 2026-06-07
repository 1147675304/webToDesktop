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
