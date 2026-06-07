//go:build linux

package pkg

import "unsafe"

// platformInit Linux 平台初始化：在 WebView 创建前调用。
// Linux 无需 WebView2 环境变量等 Windows 特定初始化。
func platformInit() {}

// platformOnShow Linux 平台窗口显示后处理。
// Linux 无需 ReapplyAcrylic 等 Windows 特定后处理。
func platformOnShow(winPtr unsafe.Pointer) {}
