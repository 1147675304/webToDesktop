//go:build linux

package pkg

import (
	"unsafe"

	"github.com/lhpanda/webtodesktop/native"
)

// platformInit Linux 平台初始化：在 WebView 创建前调用。
// 设置应用默认图标（所有窗口自动继承）。
func platformInit() {
	if AppIconPath != "" {
		native.SetDefaultAppIcon(AppIconPath)
	}
}

// platformOnShow Linux 平台窗口显示后处理。
func platformOnShow(winPtr unsafe.Pointer) {}
