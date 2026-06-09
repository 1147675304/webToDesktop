// tools/desktop/native/windows.go
// Windows 原生窗口定制：通过 HWND 调用 Win32/DWM API
//
//   可用能力：
//     - 无边框窗口 / 暗色标题栏
//     - DWM Acrylic 毛玻璃背景
//     - Win11 圆角
//     - 窗口阴影 / 拖拽区域
//     - 全屏 / 最大化

//go:build windows

package native

/*
#cgo LDFLAGS: -ldwmapi -lgdi32 -luser32

#include <windows.h>
#include <dwmapi.h>
#include <stdio.h>
#include <string.h>

// ———— WebView2 背景色 COM 接口（避免引入庞大的 WebView2.h） ————
typedef struct { BYTE A; BYTE R; BYTE G; BYTE B; } _WTD_COLOR;

// IID_ICoreWebView2Controller2 = {c979903e-d4ca-4228-92eb-47ee3fa96eab}
static const GUID _WTD_IID_CTRL2 =
    {0xc979903e,0xd4ca,0x4228,{0x92,0xeb,0x47,0xee,0x3f,0xa9,0x6e,0xab}};

// 通过 COM vtable 调用 put_DefaultBackgroundColor（vtable 偏移 27）
// 结构：IUnknown(0-2) + ICoreWebView2Controller(3-24) + ICoreWebView2Controller2(25-26)
//                                         get_DefaultBackgroundColor=26, put=27
static void setWebView2BgColor(void* ctrl, BYTE a, BYTE r, BYTE g, BYTE b) {
    if (!ctrl) return;
    void** vtbl = *(void***)ctrl;
    // QueryInterface
    HRESULT (STDMETHODCALLTYPE *qi)(void*,const GUID*,void**) =
        (HRESULT (STDMETHODCALLTYPE*)(void*,const GUID*,void**))vtbl[0];
    void* ctrl2 = NULL;
    if (FAILED(qi(ctrl, &_WTD_IID_CTRL2, &ctrl2)) || !ctrl2) return;
    // put_DefaultBackgroundColor
    void** vtbl2 = *(void***)ctrl2;
    HRESULT (STDMETHODCALLTYPE *putClr)(void*,_WTD_COLOR) =
        (HRESULT (STDMETHODCALLTYPE*)(void*,_WTD_COLOR))vtbl2[27];
    _WTD_COLOR clr = {a, r, g, b};
    putClr(ctrl2, clr);
    // Release
    ULONG (STDMETHODCALLTYPE *release)(void*) =
        (ULONG (STDMETHODCALLTYPE*)(void*))vtbl2[2];
    release(ctrl2);
}

// 强制 WebView2 控件 bounds 匹配窗口客户区（解决 1x1→全尺寸后控件不跟随的问题）
// ICoreWebView2Controller vtable: IUnknown(0-2), get_IsVisible(3), put_IsVisible(4),
//                                  get_Bounds(5), put_Bounds(6)
static void resizeWebView2Controller(void* ctrl, HWND hwnd) {
    if (!ctrl) return;
    RECT rc;
    GetClientRect(hwnd, &rc);
    void** vtbl = *(void***)ctrl;
    HRESULT (STDMETHODCALLTYPE *putBounds)(void*, RECT) =
        (HRESULT (STDMETHODCALLTYPE*)(void*, RECT))vtbl[6];
    putBounds(ctrl, rc);
}

// ———— 调试日志（写入 %TEMP%\wtd_native.log） ————
static void dbgLog(const char *msg) {
	char path[MAX_PATH];
	if (GetEnvironmentVariableA("TEMP", path, MAX_PATH) == 0) {
		return;
	}
	strcat(path, "\\wtd_native.log");
	FILE *f = fopen(path, "a");
	if (f) {
		fprintf(f, "%s\n", msg);
		fclose(f);
	}
	OutputDebugStringA(msg);
}

// MinGW 头文件中缺失的常量（Win10 1903+ / Win11）
#ifndef DWMWA_USE_IMMERSIVE_DARK_MODE
#define DWMWA_USE_IMMERSIVE_DARK_MODE 20
#endif
#ifndef DWMWA_WINDOW_CORNER_PREFERENCE
#define DWMWA_WINDOW_CORNER_PREFERENCE 33
#endif
#ifndef DWM_WINDOW_CORNER_PREFERENCE
typedef enum { DWMWCP_DEFAULT = 0, DWMWCP_DONOTROUND = 1, DWMWCP_ROUND = 2, DWMWCP_ROUNDSMALL = 3 } DWM_WINDOW_CORNER_PREFERENCE;
#endif

static void setDarkTitleBar(HWND hwnd) {
	dbgLog("[native] setDarkTitleBar");
	BOOL dark = TRUE;
	DwmSetWindowAttribute(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, &dark, sizeof(dark));
}

static void setRoundCorners(HWND hwnd) {
	dbgLog("[native] setRoundCorners");
	DWM_WINDOW_CORNER_PREFERENCE corner = DWMWCP_ROUND;
	DwmSetWindowAttribute(hwnd, DWMWA_WINDOW_CORNER_PREFERENCE, &corner, sizeof(corner));
}

static void setAcrylic(HWND hwnd) {
	dbgLog("[native] setAcrylic");
	DWM_BLURBEHIND bb = {0};
	bb.dwFlags = DWM_BB_ENABLE | DWM_BB_BLURREGION;
	bb.fEnable = TRUE;
	bb.hRgnBlur = CreateRectRgn(0, 0, -1, -1);
	DwmEnableBlurBehindWindow(hwnd, &bb);
	DeleteObject(bb.hRgnBlur);
}

static void setBorderless(HWND hwnd) {
	dbgLog("[native] setBorderless — remove WS_CAPTION|WS_THICKFRAME, keep WS_SYSMENU|WS_MAXIMIZEBOX");
	LONG style = GetWindowLong(hwnd, GWL_STYLE);
	// 移除标题栏和可调边框。WS_SYSMENU（拖拽）、WS_MAXIMIZEBOX（最大化）保留
	style &= ~(WS_CAPTION | WS_THICKFRAME);
	SetWindowLong(hwnd, GWL_STYLE, style);
	LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
	exStyle &= ~(WS_EX_DLGMODALFRAME | WS_EX_CLIENTEDGE | WS_EX_STATICEDGE);
	SetWindowLong(hwnd, GWL_EXSTYLE, exStyle);
	SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
		SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER);
}

// ———— 窗口位置设置 ————
// pos: ""=跳过 "center"=居中 "x,y"=绝对坐标
static void setWindowPosition(HWND hwnd, const char *pos) {
	if (!pos || !pos[0]) return;  // 空字符串 → 不设置

	RECT wr;
	if (!GetWindowRect(hwnd, &wr)) return;
	int width  = wr.right - wr.left;
	int height = wr.bottom - wr.top;

	int x = 0, y = 0;

	if (strcmp(pos, "center") == 0) {
		RECT wa;
		if (SystemParametersInfoA(SPI_GETWORKAREA, 0, &wa, 0)) {
			int waW = wa.right - wa.left;
			int waH = wa.bottom - wa.top;
			x = wa.left + (waW - width) / 2;
			y = wa.top  + (waH - height) / 2;
		}
	} else if (sscanf(pos, "%d,%d", &x, &y) == 2) {
		// 直接使用解析的 x, y
	} else {
		char buf[256];
		snprintf(buf, sizeof(buf), "[native] setWindowPosition: invalid format '%s', expected 'center' or 'x,y'", pos);
		dbgLog(buf);
		return;
	}

	char buf[128];
	snprintf(buf, sizeof(buf), "[native] setWindowPosition: %d,%d (%dx%d)", x, y, width, height);
	dbgLog(buf);
	SetWindowPos(hwnd, NULL, x, y, 0, 0, SWP_NOZORDER | SWP_NOSIZE | SWP_NOACTIVATE);
}

// 全屏模式：覆盖整个显示器（含任务栏区域）
static void setFullscreen(HWND hwnd) {
	dbgLog("[native] setFullscreen");
	HMONITOR monitor = MonitorFromWindow(hwnd, MONITOR_DEFAULTTONEAREST);
	MONITORINFO info = {0};
	info.cbSize = sizeof(MONITORINFO);
	if (GetMonitorInfo(monitor, &info)) {
		LONG style = GetWindowLong(hwnd, GWL_STYLE);
		style &= ~(WS_CAPTION | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX | WS_SYSMENU);
		SetWindowLong(hwnd, GWL_STYLE, style);
		SetWindowPos(hwnd, HWND_TOP,
			info.rcMonitor.left, info.rcMonitor.top,
			info.rcMonitor.right - info.rcMonitor.left,
			info.rcMonitor.bottom - info.rcMonitor.top,
			SWP_FRAMECHANGED);
	}
}

// ———— 共享状态：最大化/还原用 ————
static RECT g_normalRect;       // 最大化前的原始窗口尺寸（由 setMaximized / setConfigSize 初始化）
static BOOL g_isMaxed = FALSE;
static int  g_configWidth  = 0; // 配置文件中的窗口宽度（永远不变）
static int  g_configHeight = 0; // 配置文件中的窗口高度（永远不变）

// setConfigSize 保存配置文件中的窗口尺寸，供最大化/全屏恢复时使用。
static void setConfigSize(int w, int h) {
	g_configWidth  = w;
	g_configHeight = h;
	// 同时初始化 g_normalRect（用于首次还原时的回退值）
	g_normalRect.left   = 0;
	g_normalRect.top    = 0;
	g_normalRect.right  = w;
	g_normalRect.bottom = h;
}

// 窗口最大化 — 先保存当前尺寸为"默认"，再定位到工作区
static void setMaximized(HWND hwnd) {
	dbgLog("[native] setMaximized");
	GetWindowRect(hwnd, &g_normalRect);  // 保存 Config 的 width×height
	RECT wa;
	if (SystemParametersInfoA(SPI_GETWORKAREA, 0, &wa, 0)) {
		SetWindowPos(hwnd, NULL, wa.left, wa.top,
			wa.right - wa.left, wa.bottom - wa.top,
			SWP_NOZORDER | SWP_NOACTIVATE);
	}
	g_isMaxed = TRUE;
}

// 窗口置顶
static void setAlwaysOnTop(HWND hwnd) {
	dbgLog("[native] setAlwaysOnTop");
	LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
	SetWindowLong(hwnd, GWL_EXSTYLE, exStyle | WS_EX_TOPMOST);
	SetWindowPos(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
		SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE);
}

// 从 EXE 资源加载图标并设置到窗口（任务栏 + 标题栏）
// 图标资源 ID = 1（由 windres 从 _icon.rc 编译到 rsrc.syso）
static void setWindowIcon(HWND hwnd) {
	dbgLog("[native] setWindowIcon");
	HINSTANCE hInst = GetModuleHandle(NULL);
	// 尝试加载大图标（任务栏 Alt+Tab 切换界面使用）
	HICON hIconBig = LoadIcon(hInst, MAKEINTRESOURCE(1));
	if (hIconBig) {
		SendMessage(hwnd, WM_SETICON, ICON_BIG, (LPARAM)hIconBig);
		dbgLog("[native] setWindowIcon: ICON_BIG set");
	}
	// 小图标（任务栏、标题栏使用）
	HICON hIconSmall = (HICON)LoadImage(hInst, MAKEINTRESOURCE(1), IMAGE_ICON,
		GetSystemMetrics(SM_CXSMICON), GetSystemMetrics(SM_CYSMICON), 0);
	if (hIconSmall) {
		SendMessage(hwnd, WM_SETICON, ICON_SMALL, (LPARAM)hIconSmall);
		dbgLog("[native] setWindowIcon: ICON_SMALL set");
	}
}

// 窗口透明度 (0.0 全透明 ~ 1.0 不透明)
static void setOpacity(HWND hwnd, double opacity) {
	dbgLog("[native] setOpacity");
	LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
	SetWindowLong(hwnd, GWL_EXSTYLE, exStyle | WS_EX_LAYERED);
	SetLayeredWindowAttributes(hwnd, 0, (BYTE)(opacity * 255), LWA_ALPHA);
}

// ———— 窗口拖拽与调整大小 ————
// 模拟在窗口非客户区按下鼠标左键，由系统接管后续拖拽/缩放行为。

// HTxxx 常量（非客户区命中测试）
#define HT_CAPTION       2   // 标题栏 → 拖拽窗口
#define HT_LEFT         10   // 左边框
#define HT_RIGHT        11   // 右边框
#define HT_TOP          12   // 上边框
#define HT_TOPLEFT      13   // 左上角
#define HT_TOPRIGHT     14   // 右上角
#define HT_BOTTOM       15   // 下边框
#define HT_BOTTOMLEFT   16   // 左下角
#define HT_BOTTOMRIGHT  17   // 右下角

// ———— 无边框窗口边缘 resize：子类化 WndProc 拦截 WM_NCHITTEST ————
static void reapplyAcrylicAndFlush(HWND hwnd); // 前置声明

static WNDPROC g_origBorderlessProc = NULL;
static const LONG kResizeMargin = 6;

static LRESULT CALLBACK BorderlessWndProc(HWND hwnd, UINT msg, WPARAM wp, LPARAM lp) {
    if (msg == WM_NCHITTEST) {
        POINT pt = { (short)LOWORD(lp), (short)HIWORD(lp) };
        ScreenToClient(hwnd, &pt);
        RECT rc; GetClientRect(hwnd, &rc);
        BOOL l = pt.x <= kResizeMargin;
        BOOL r = pt.x >= rc.right - kResizeMargin;
        BOOL t = pt.y <= kResizeMargin;
        BOOL b = pt.y >= rc.bottom - kResizeMargin;
        if (t && l) return HTTOPLEFT;    if (t && r) return HTTOPRIGHT;
        if (b && l) return HTBOTTOMLEFT; if (b && r) return HTBOTTOMRIGHT;
        if (t) return HTTOP;  if (b) return HTBOTTOM;
        if (l) return HTLEFT; if (r) return HTRIGHT;
    }
    if (msg == WM_EXITSIZEMOVE) {
        // 拖拽完成后：0×0 强制 WebView 释放旧缓冲 → 恢复到当前尺寸 → 重刷 Acrylic
        RECT rc;
        GetWindowRect(hwnd, &rc);
        int w = rc.right - rc.left;
        int h = rc.bottom - rc.top;
        SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
            SWP_NOZORDER | SWP_NOACTIVATE | SWP_NOCOPYBITS);
        SetWindowPos(hwnd, NULL, rc.left, rc.top, w, h,
            SWP_NOZORDER | SWP_NOACTIVATE);
        reapplyAcrylicAndFlush(hwnd);
    }
    return CallWindowProc(g_origBorderlessProc, hwnd, msg, wp, lp);
}

static void enableBorderlessResize(HWND hwnd) {
    if (g_origBorderlessProc) return;
    g_origBorderlessProc = (WNDPROC)SetWindowLongPtr(hwnd, GWLP_WNDPROC, (LONG_PTR)BorderlessWndProc);
}

static void startWindowDrag(HWND hwnd) {
	dbgLog("[native] startWindowDrag");

	// ★ 临时加回 WS_THICKFRAME 以启用 Windows Aero Snap
	// （无边框窗口移除了此样式，但 Snap 依赖它；拖拽结束后再移除）
	LONG style = GetWindowLong(hwnd, GWL_STYLE);
	BOOL needRestore = !(style & WS_THICKFRAME);
	if (needRestore) {
		SetWindowLong(hwnd, GWL_STYLE, style | WS_THICKFRAME);
		SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
			SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER);
	}

	if (g_isMaxed) {
		// 最大化了 → 先还原到配置文件尺寸（保持当前 X 位置，Y 置顶）
		RECT cur;
		GetWindowRect(hwnd, &cur);
		int w = (g_configWidth  > 0) ? g_configWidth  : (g_normalRect.right - g_normalRect.left);
		int h = (g_configHeight > 0) ? g_configHeight : (g_normalRect.bottom - g_normalRect.top);
		SetWindowPos(hwnd, NULL, cur.left, cur.top, w, h,
			SWP_NOZORDER | SWP_NOACTIVATE);
		g_isMaxed = FALSE;
	}
	ReleaseCapture();
	// SendMessage 进入模态拖拽循环，用户松开鼠标后返回
	SendMessage(hwnd, WM_NCLBUTTONDOWN, HT_CAPTION, 0);

	// 拖拽结束 → 移除临时加回的 WS_THICKFRAME（恢复无边框样式）
	if (needRestore) {
		SetWindowLong(hwnd, GWL_STYLE, style);
		SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
			SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER);
	}
}

// edge: 使用 HT_LEFT ~ HT_BOTTOMRIGHT 指定拖拽边缘
// 临时加回 WS_THICKFRAME 以启用 DefWindowProc 缩放，完成后移除避免黑条。
// 注意：不在此处做 0×0 前置，因为拖拽由 OS 实时接管，前置会导致窗口闪烁消失。
static void startWindowResize(HWND hwnd, int edge) {
	dbgLog("[native] startWindowResize");
	LONG style = GetWindowLong(hwnd, GWL_STYLE);
	SetWindowLong(hwnd, GWL_STYLE, style | WS_THICKFRAME);
	ReleaseCapture();
	SendMessage(hwnd, WM_NCLBUTTONDOWN, (WPARAM)edge, 0);
	// 缩放循环结束后移除
	SetWindowLong(hwnd, GWL_STYLE, style);
	SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
		SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER);
}
// 关闭窗口 — 先 PostMessage WM_CLOSE 优雅退出，再直接销毁确保关闭
static void closeWindow(HWND hwnd) {
	PostMessage(hwnd, WM_CLOSE, 0, 0);
}

// 窗口显示后/ resize 后重新声明 Acrylic 并同步 DWM
static void reapplyAcrylicAndFlush(HWND hwnd) {
    DWM_BLURBEHIND bb = {0};
    bb.dwFlags = DWM_BB_ENABLE | DWM_BB_BLURREGION;
    bb.fEnable = TRUE;
    bb.hRgnBlur = CreateRectRgn(0, 0, -1, -1);
    DwmEnableBlurBehindWindow(hwnd, &bb);
    DeleteObject(bb.hRgnBlur);
    DwmFlush();
    InvalidateRect(hwnd, NULL, TRUE);
    UpdateWindow(hwnd);
}

// 最大化/还原 — 与 setMaximized 共享 g_normalRect/g_isMaxed
// ★ 先缩到 0×0 强制 WebView 释放布局缓存，再设置目标尺寸，避免残影/黑边。
// ★ 还原时永远使用配置文件中的尺寸（g_configWidth × g_configHeight），
//    不保存用户拖拽后的尺寸，确保每次还原一致。
static void toggleMaximize(HWND hwnd) {
	dbgLog("[native] toggleMaximize");

	// 0×0 前置
	SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
		SWP_NOZORDER | SWP_NOACTIVATE | SWP_NOCOPYBITS);

	if (g_isMaxed) {
		// 还原：用配置文件尺寸居中到当前显示器
		if (g_configWidth > 0 && g_configHeight > 0) {
			RECT wa;
			if (SystemParametersInfoA(SPI_GETWORKAREA, 0, &wa, 0)) {
				int waW = wa.right - wa.left;
				int waH = wa.bottom - wa.top;
				int x = wa.left + (waW - g_configWidth) / 2;
				int y = wa.top  + (waH - g_configHeight) / 2;
				SetWindowPos(hwnd, NULL, x, y,
					g_configWidth, g_configHeight,
					SWP_NOZORDER | SWP_NOACTIVATE);
			}
		} else {
			// 回退：使用 g_normalRect
			SetWindowPos(hwnd, NULL,
				g_normalRect.left, g_normalRect.top,
				g_normalRect.right - g_normalRect.left,
				g_normalRect.bottom - g_normalRect.top,
				SWP_NOZORDER | SWP_NOACTIVATE);
		}
		g_isMaxed = FALSE;
	} else {
		// 最大化前保存当前尺寸（仅用于极端回退，不再覆盖 g_normalRect 的 size）
		GetWindowRect(hwnd, &g_normalRect);
		RECT wa;
		if (SystemParametersInfoA(SPI_GETWORKAREA, 0, &wa, 0)) {
			SetWindowPos(hwnd, NULL, wa.left, wa.top,
				wa.right - wa.left, wa.bottom - wa.top,
				SWP_NOZORDER | SWP_NOACTIVATE);
		}
		g_isMaxed = TRUE;
	}
}

// 全屏切换 — 用静态变量跟踪状态
// ★ 先缩到 0×0 强制 WebView 释放布局缓存，再设置目标尺寸，避免残影/黑边。
// ★ 退出全屏时永远恢复到配置文件尺寸，居中到当前显示器。
static void toggleFullscreen(HWND hwnd) {
	static BOOL isFullscreen = FALSE;
	static RECT savedRect;
	static LONG savedStyle, savedExStyle;

	// 0×0 前置
	SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
		SWP_NOZORDER | SWP_NOACTIVATE | SWP_NOCOPYBITS);

	if (!isFullscreen) {
		// 进入全屏：保存当前状态
		GetWindowRect(hwnd, &savedRect);
		savedStyle = GetWindowLong(hwnd, GWL_STYLE);
		savedExStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
		// 移除边框样式，覆盖整个显示器
		LONG newStyle = savedStyle & ~(WS_CAPTION | WS_THICKFRAME);
		SetWindowLong(hwnd, GWL_STYLE, newStyle);
		HMONITOR monitor = MonitorFromWindow(hwnd, MONITOR_DEFAULTTONEAREST);
		MONITORINFO info = {sizeof(MONITORINFO)};
		if (GetMonitorInfo(monitor, &info)) {
			SetWindowPos(hwnd, HWND_TOP,
				info.rcMonitor.left, info.rcMonitor.top,
				info.rcMonitor.right - info.rcMonitor.left,
				info.rcMonitor.bottom - info.rcMonitor.top,
				SWP_FRAMECHANGED);
		}
		isFullscreen = TRUE;
	} else {
		// 退出全屏：恢复样式，用配置文件尺寸居中到当前显示器
		SetWindowLong(hwnd, GWL_STYLE, savedStyle);
		SetWindowLong(hwnd, GWL_EXSTYLE, savedExStyle);

		if (g_configWidth > 0 && g_configHeight > 0) {
			RECT wa;
			if (SystemParametersInfoA(SPI_GETWORKAREA, 0, &wa, 0)) {
				int waW = wa.right - wa.left;
				int waH = wa.bottom - wa.top;
				int x = wa.left + (waW - g_configWidth) / 2;
				int y = wa.top  + (waH - g_configHeight) / 2;
				SetWindowPos(hwnd, NULL, x, y,
					g_configWidth, g_configHeight,
					SWP_FRAMECHANGED);
			}
		} else {
			// 回退：使用保存的尺寸
			SetWindowPos(hwnd, NULL,
				savedRect.left, savedRect.top,
				savedRect.right - savedRect.left,
				savedRect.bottom - savedRect.top,
				SWP_FRAMECHANGED);
		}
		isFullscreen = FALSE;
	}
}
*/
import "C"
import (
	"fmt"
	"os"
	"unsafe"
)

// dbgLog 写入 %TEMP%\wtd_go.log
func dbgLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if tmp := os.Getenv("TEMP"); tmp != "" {
		if f, err := os.OpenFile(tmp+"\\wtd_go.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintln(f, msg)
			f.Close()
		}
	}
}

// SetWebView2BackgroundColor 运行时设置 WebView2 控件背景色（ARGB）。
// ctrlPtr 来自 WebView2 ICoreWebView2Controller 指针。
func SetWebView2BackgroundColor(ctrlPtr unsafe.Pointer, a, r, g, b byte) {
	C.setWebView2BgColor(ctrlPtr, C.BYTE(a), C.BYTE(r), C.BYTE(g), C.BYTE(b))
}

// ReapplyAcrylic 窗口显示后/ resize 后重新声明 Acrylic 并同步 DWM。
func ReapplyAcrylic(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.reapplyAcrylicAndFlush((C.HWND)(winPtr))
}

// EnableBorderlessResize 为无边框窗口启用原生边缘 resize（子类化 WM_NCHITTEST）。
func EnableBorderlessResize(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.enableBorderlessResize((C.HWND)(winPtr))
}

// ResizeWebView2Controller 强制 WebView2 控件 bounds 匹配窗口客户区。
// 用于窗口尺寸变更后确保 WebView2 内容区域填满窗口。
func ResizeWebView2Controller(ctrlPtr unsafe.Pointer, winPtr unsafe.Pointer) {
	if ctrlPtr == nil || winPtr == nil {
		return
	}
	C.resizeWebView2Controller(ctrlPtr, (C.HWND)(winPtr))
}

// SetWindowIcon Windows 图标从 EXE 资源加载（非文件路径），此函数为空实现。
func SetWindowIcon(winPtr unsafe.Pointer, iconPath string) {}

// SetDefaultAppIcon Windows 图标从 EXE 资源加载，此函数为空实现。
func SetDefaultAppIcon(iconPath string) {}

// DisableWebKitHardwareAccel Windows 上通过 WebView2 环境设置，无需单独调用。
func DisableWebKitHardwareAccel(winPtr unsafe.Pointer) {}

// SetDefaultWindowSize 保存配置中的窗口尺寸（width × height）。
// 最大化/全屏恢复时永远使用此尺寸，而非用户拖拽后的尺寸。
func SetDefaultWindowSize(width, height int) {
	C.setConfigSize(C.int(width), C.int(height))
}

// ApplyPreShow 在窗口显示前调用。
// Windows: 仅设置 DWM 窗口属性（暗色标题栏/圆角/毛玻璃），这些属性不依赖窗口尺寸，
// 可安全在显示前设置。其余配置（边框/置顶/全屏/最大化/位置/透明度）统一在 Apply 中设置。
func ApplyPreShow(winPtr unsafe.Pointer, cfg WindowConfig) {
	hwnd := (C.HWND)(winPtr)
	if cfg.DarkTitleBar {
		C.setDarkTitleBar(hwnd)
	}
	if cfg.RoundCorners {
		C.setRoundCorners(hwnd)
	}
	if cfg.Acrylic {
		C.setAcrylic(hwnd)
	}
}

// Apply Windows 平台外观配置。
// 在 w.Dispatch() 回调中调用（确保 UI 线程）。
// 执行顺序很重要：先去边框 → 再最大化/全屏 → 最后置顶/透明度。
func Apply(winPtr unsafe.Pointer, cfg WindowConfig) {
	dbgLog("[go] Apply called: Borderless=%v Maximized=%v Fullscreen=%v AlwaysOnTop=%v Opacity=%.2f",
		cfg.Borderless, cfg.Maximized, cfg.Fullscreen, cfg.AlwaysOnTop, cfg.Opacity)
	hwnd := (C.HWND)(winPtr)

	// 1. Windows 专属效果（不影响窗口状态）
	if cfg.DarkTitleBar {
		C.setDarkTitleBar(hwnd)
	}
	if cfg.RoundCorners {
		C.setRoundCorners(hwnd)
	}
	if cfg.Acrylic {
		C.setAcrylic(hwnd)
	}

	// 2. 去边框（必须在最大化/全屏之前，否则 SetWindowPos 会撤销最大化）
	if cfg.Borderless {
		C.setBorderless(hwnd)
		C.enableBorderlessResize(hwnd) // 子类化 WM_NCHITTEST 支持原生边缘 resize
	}

	// 3. 窗口置顶（在最大化之前，避免 z-order 干扰尺寸）
	if cfg.AlwaysOnTop {
		C.setAlwaysOnTop(hwnd)
	}

	// 4. 最大化或全屏（最后执行，确保尺寸不被覆盖）
	if cfg.Fullscreen {
		C.setFullscreen(hwnd)
	} else if cfg.Maximized {
		C.setMaximized(hwnd)
	} else {
		// 非最大化/非全屏 → 设置窗口位置
		cPos := C.CString(cfg.WindowPosition)
		C.setWindowPosition(hwnd, cPos)
		C.free(unsafe.Pointer(cPos))
	}

	// 5. 透明度
	if cfg.Opacity < 1.0 {
		C.setOpacity(hwnd, C.double(cfg.Opacity))
	}

	// 6. 窗口图标（从 EXE 资源加载，ID=1）
	C.setWindowIcon(hwnd)
}

// ———— 窗口拖拽 / 调整大小（无边框窗口用） ————

// 调整方向常量，对应 HTxxx
const (
	ResizeLeft        = 10 // HTLEFT
	ResizeRight       = 11 // HTRIGHT
	ResizeTop         = 12 // HTTOP
	ResizeTopLeft     = 13 // HTTOPLEFT
	ResizeTopRight    = 14 // HTTOPRIGHT
	ResizeBottom      = 15 // HTBOTTOM
	ResizeBottomLeft  = 16 // HTBOTTOMLEFT
	ResizeBottomRight = 17 // HTBOTTOMRIGHT
)

// DragWindow 触发窗口拖拽（模拟在标题栏按下鼠标）。
// 需在 Dispatch 回调中调用。
func DragWindow(winPtr unsafe.Pointer) {
	C.startWindowDrag((C.HWND)(winPtr))
}

// ResizeWindow 从指定边缘/角落触发窗口缩放。
// edge: 使用 ResizeLeft ~ ResizeBottomRight 常量。
// 需在 Dispatch 回调中调用。
func ResizeWindow(winPtr unsafe.Pointer, edge int) {
	C.startWindowResize((C.HWND)(winPtr), C.int(edge))
}

// CloseWindow 关闭窗口。
func CloseWindow(winPtr unsafe.Pointer) {
	C.closeWindow((C.HWND)(winPtr))
}

// ToggleMaximize 最大化/还原切换。
func ToggleMaximize(winPtr unsafe.Pointer) {
	C.toggleMaximize((C.HWND)(winPtr))
}

// ToggleFullscreen 全屏切换。
func ToggleFullscreen(winPtr unsafe.Pointer) {
	C.toggleFullscreen((C.HWND)(winPtr))
}

// ToggleMinimize 最小化窗口到任务栏。
func ToggleMinimize(winPtr unsafe.Pointer) {
	hwnd := (C.HWND)(winPtr)
	// SW_MINIMIZE = 6，将窗口最小化到任务栏
	C.ShowWindow(hwnd, 6)
}

// HasDisplay 在 Windows 上始终返回 true（WebView2 无需 X11/Wayland）。
func HasDisplay() bool { return true }
