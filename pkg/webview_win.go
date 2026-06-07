//go:build windows

package pkg

/*
#cgo CXXFLAGS: -I${SRCDIR}/../include_win
*/
import "C"

import (
	"os"
	"unsafe"

	"github.com/lhpanda/webtodesktop/native"
)

// platformInit Windows 平台初始化：在 WebView 创建前调用。
// 设置 WebView2 透明背景环境变量（Acrylic/透明背景模式下需要）。
func platformInit() {
	needTransparentBg := AppCfg.Window.Acrylic || AppCfg.Window.WebViewBgTransparent
	if needTransparentBg {
		os.Setenv("WEBVIEW2_DEFAULT_BACKGROUND_COLOR", "00000000")
		dbg("WEBVIEW2_DEFAULT_BACKGROUND_COLOR=00000000")
	}
}

// platformOnShow Windows 平台窗口显示后处理。
// 重新应用 Acrylic 毛玻璃效果（窗口显示后 DWM 需要重新声明）。
func platformOnShow(winPtr unsafe.Pointer) {
	needTransparentBg := AppCfg.Window.Acrylic || AppCfg.Window.WebViewBgTransparent
	if needTransparentBg {
		native.ReapplyAcrylic(winPtr)
	}
}
