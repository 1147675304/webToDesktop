// tools/desktop/native/other.go
// 非 Windows/Linux 平台：空实现

//go:build !windows && !linux

package native

import "unsafe"

func Apply(winPtr unsafe.Pointer, cfg WindowConfig) {}

func ApplyPreShow(winPtr unsafe.Pointer, cfg WindowConfig) {}

// 调整方向常量（与 Windows 一致）
const (
	ResizeLeft        = 10
	ResizeRight       = 11
	ResizeTop         = 12
	ResizeTopLeft     = 13
	ResizeTopRight    = 14
	ResizeBottom      = 15
	ResizeBottomLeft  = 16
	ResizeBottomRight = 17
)

func DragWindow(winPtr unsafe.Pointer)             {}
func ResizeWindow(winPtr unsafe.Pointer, edge int) {}
func CloseWindow(winPtr unsafe.Pointer)            {}
func ToggleMaximize(winPtr unsafe.Pointer)         {}
func ToggleFullscreen(winPtr unsafe.Pointer)       {}

// HasDisplay 在其他平台上始终返回 true。
func HasDisplay() bool { return true }

func SetWebView2BackgroundColor(ctrlPtr unsafe.Pointer, a, r, g, b byte)     {}
func ReapplyAcrylic(winPtr unsafe.Pointer)                                   {}
func EnableBorderlessResize(winPtr unsafe.Pointer)                           {}
func ResizeWebView2Controller(ctrlPtr unsafe.Pointer, winPtr unsafe.Pointer) {}
func SetDefaultWindowSize(width, height int)                                 {}
func SetWindowIcon(winPtr unsafe.Pointer, iconPath string)                   {}
func SetDefaultAppIcon(iconPath string)                                      {}
func DisableWebKitHardwareAccel(winPtr unsafe.Pointer)                       {}
func EnableKeyboardHook(winPtr unsafe.Pointer)                               {}
func DisableKeyboardHook(winPtr unsafe.Pointer)                              {}
func PollKbEvent() string                                                    { return "" }
func syncBlockedKeysToC(keys []string)                                       {}
func ToggleMinimize(winPtr unsafe.Pointer)                                   {}
func EnableInputPassthrough(winPtr unsafe.Pointer)                           {}
func DisableInputPassthrough(winPtr unsafe.Pointer)                          {}
func IsInputPassthrough(winPtr unsafe.Pointer) bool                          { return false }

// OpaqueRegion 不透光矩形（与 Windows 定义保持一致）。
type OpaqueRegion struct {
	X, Y, W, H int
}

func SetOpaqueRegions(winPtr unsafe.Pointer, regions []OpaqueRegion) {}
func ClearOpaqueRegions(winPtr unsafe.Pointer)                       {}
func PollPassthroughState(winPtr unsafe.Pointer)                     {}
func PollCursorPos(winPtr unsafe.Pointer) (int32, int32)             { return -1, -1 }
