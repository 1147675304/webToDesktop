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
#ifndef WS_EX_TRANSPARENT
#define WS_EX_TRANSPARENT 0x00000020L
#endif
#ifndef WS_EX_LAYERED
#define WS_EX_LAYERED 0x00080000L
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

// ———— 不透光区域 + 穿透状态 ————
#define MAX_OPAQUE_REGIONS 128
static RECT g_opaqueRegions[MAX_OPAQUE_REGIONS];
static int g_opaqueRegionCount = 0;
static BOOL g_passthroughEnabled = FALSE;

// 检查客户区坐标 pt 是否在任一不透光区域内
static BOOL isPointInOpaqueRegionC(POINT pt) {
    for (int i = 0; i < g_opaqueRegionCount; i++) {
        if (PtInRect(&g_opaqueRegions[i], pt)) {
            return TRUE;
        }
    }
    return FALSE;
}

// ———— 系统托盘图标 ————
#define WM_TRAYICON  (WM_APP + 100)
#define IDM_TRAY_SHOW  3001
#define IDM_TRAY_EXIT  3002

static NOTIFYICONDATAW g_nid;
static BOOL g_trayCreated = FALSE;
static HMENU g_trayMenu = NULL;
static WNDPROC g_origTrayProc = NULL;  // 托盘专用 WndProc 的原始句柄

static void createTrayIcon(HWND hwnd) {
    if (g_trayCreated) return;
    HICON hIcon = (HICON)LoadImage(GetModuleHandle(NULL), MAKEINTRESOURCE(1),
        IMAGE_ICON, GetSystemMetrics(SM_CXSMICON), GetSystemMetrics(SM_CYSMICON), 0);
    if (!hIcon) {
        hIcon = LoadIcon(GetModuleHandle(NULL), MAKEINTRESOURCE(1));
    }
    ZeroMemory(&g_nid, sizeof(g_nid));
    g_nid.cbSize = sizeof(NOTIFYICONDATAW);
    g_nid.hWnd = hwnd;
    g_nid.uID = 1;
    g_nid.uFlags = NIF_ICON | NIF_MESSAGE | NIF_TIP;
    g_nid.uCallbackMessage = WM_TRAYICON;
    g_nid.hIcon = hIcon ? hIcon : LoadIcon(NULL, IDI_APPLICATION);
    wcscpy(g_nid.szTip, L"WebToDesktop");
    Shell_NotifyIconW(NIM_ADD, &g_nid);
    g_trayCreated = TRUE;
    dbgLog("[native] createTrayIcon: OK");
}

static void removeTrayIcon(void) {
    if (!g_trayCreated) return;
    Shell_NotifyIconW(NIM_DELETE, &g_nid);
    if (g_trayMenu) { DestroyMenu(g_trayMenu); g_trayMenu = NULL; }
    g_trayCreated = FALSE;
    dbgLog("[native] removeTrayIcon: OK");
}

// 隐藏窗口的任务栏按钮（WS_EX_TOOLWINDOW），托盘模式下使用
static void hideFromTaskbar(HWND hwnd) {
    LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
    SetWindowLong(hwnd, GWL_EXSTYLE, exStyle | WS_EX_TOOLWINDOW);
    SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
        SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_NOACTIVATE);
}

// 恢复窗口的任务栏按钮（移除 WS_EX_TOOLWINDOW）
static void showInTaskbar(HWND hwnd) {
    LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
    SetWindowLong(hwnd, GWL_EXSTYLE, exStyle & ~WS_EX_TOOLWINDOW);
    SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
        SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_NOACTIVATE);
}

// 完整禁用托盘模式：移除图标 + 恢复任务栏 + 还原 WndProc
static void disableSystemTray(HWND hwnd) {
    // 1. 移除托盘图标
    if (g_trayCreated) {
        Shell_NotifyIconW(NIM_DELETE, &g_nid);
        if (g_trayMenu) { DestroyMenu(g_trayMenu); g_trayMenu = NULL; }
        g_trayCreated = FALSE;
    }
    // 2. 恢复任务栏按钮
    showInTaskbar(hwnd);
    // 3. 如果安装了专用托盘 WndProc，还原原始 WndProc
    if (g_origTrayProc) {
        SetWindowLongPtr(hwnd, GWLP_WNDPROC, (LONG_PTR)g_origTrayProc);
        g_origTrayProc = NULL;
        dbgLog("[native] disableSystemTray: TrayWndProc removed");
    }
    dbgLog("[native] disableSystemTray: OK");
}

static void createTrayMenu(void) {
    if (g_trayMenu) return;
    g_trayMenu = CreatePopupMenu();
    AppendMenuW(g_trayMenu, MF_STRING, IDM_TRAY_SHOW, L"显示窗口");
    AppendMenuW(g_trayMenu, MF_SEPARATOR, 0, NULL);
    AppendMenuW(g_trayMenu, MF_STRING, IDM_TRAY_EXIT, L"退出");
}

static void showTrayMenu(HWND hwnd) {
    createTrayMenu();
    POINT pt;
    GetCursorPos(&pt);
    SetForegroundWindow(hwnd);
    TrackPopupMenu(g_trayMenu, TPM_RIGHTBUTTON | TPM_BOTTOMALIGN, pt.x, pt.y, 0, hwnd, NULL);
}

static LRESULT CALLBACK BorderlessWndProc(HWND hwnd, UINT msg, WPARAM wp, LPARAM lp) {
    // ★ 托盘消息（兼容无边框 + 托盘同时启用的场景）
    if (msg == WM_TRAYICON) {
        if (lp == WM_RBUTTONUP) {
            showTrayMenu(hwnd);
        } else if (lp == WM_LBUTTONDBLCLK) {
            ShowWindow(hwnd, SW_SHOW);
            SetForegroundWindow(hwnd);
        }
        return 0;
    }
    if (msg == WM_COMMAND) {
        switch (LOWORD(wp)) {
            case IDM_TRAY_SHOW:
                ShowWindow(hwnd, SW_SHOW);
                SetForegroundWindow(hwnd);
                break;
            case IDM_TRAY_EXIT:
                removeTrayIcon();
                PostQuitMessage(0);
                break;
        }
        return 0;
    }
    if (msg == WM_CLOSE && g_trayCreated) {
        ShowWindow(hwnd, SW_HIDE);
        return 0;
    }
    if (msg == WM_DESTROY) {
        removeTrayIcon();
    }
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

// ———— 系统托盘 WndProc（轻量，仅拦截托盘消息和 WM_CLOSE） ————

static LRESULT CALLBACK TrayWndProc(HWND hwnd, UINT msg, WPARAM wp, LPARAM lp) {
    if (msg == WM_TRAYICON) {
        if (lp == WM_RBUTTONUP) {
            showTrayMenu(hwnd);
        } else if (lp == WM_LBUTTONDBLCLK) {
            ShowWindow(hwnd, SW_SHOW);
            SetForegroundWindow(hwnd);
        }
        return 0;
    }
    if (msg == WM_COMMAND) {
        switch (LOWORD(wp)) {
            case IDM_TRAY_SHOW:
                ShowWindow(hwnd, SW_SHOW);
                SetForegroundWindow(hwnd);
                break;
            case IDM_TRAY_EXIT:
                removeTrayIcon();
                PostQuitMessage(0);
                break;
        }
        return 0;
    }
    // 托盘模式：关闭 → 隐藏到托盘
    if (msg == WM_CLOSE && g_trayCreated) {
        ShowWindow(hwnd, SW_HIDE);
        return 0;
    }
    if (msg == WM_DESTROY) {
        removeTrayIcon();
    }
    return CallWindowProc(g_origTrayProc, hwnd, msg, wp, lp);
}

static void enableSystemTray(HWND hwnd) {
    if (g_origBorderlessProc) {
        // 已经有无边框 WndProc，不要覆盖它
        // 托盘消息已在 BorderlessWndProc 中处理（通过 enableBorderlessResize 安装）
        dbgLog("[native] enableSystemTray: BorderlessWndProc already active, skip TrayWndProc install");
        return;
    }
    if (g_origTrayProc) return; // 已安装
    g_origTrayProc = (WNDPROC)SetWindowLongPtr(hwnd, GWLP_WNDPROC, (LONG_PTR)TrayWndProc);
    dbgLog("[native] enableSystemTray: TrayWndProc installed");
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

// ———— 键盘快捷键拦截 ————
// 通过 C 层静态数组管理，Go 侧通过 syncBlockedKeysToC 同步

// C 层屏蔽键列表（由 Go 端 syncBlockedKeysToC 维护）
#define MAX_BLOCKED_KEYS 128
static char g_blockedKeys[MAX_BLOCKED_KEYS][128];
static int g_blockedCount = 0;

// 清空 C 层屏蔽键列表
static void clearBlockedKeysC(void) {
    g_blockedCount = 0;
}

// 向 C 层列表添加一条屏蔽键
static void addBlockedKeyC(const char* key) {
    if (g_blockedCount >= MAX_BLOCKED_KEYS) return;
    strncpy(g_blockedKeys[g_blockedCount], key, 127);
    g_blockedKeys[g_blockedCount][127] = '\0';
    g_blockedCount++;
}

// 查询 C 层列表，返回 1=应拦截
static int isKeyBlockedC(const char* key) {
    for (int i = 0; i < g_blockedCount; i++) {
        if (strcmp(g_blockedKeys[i], key) == 0) {
            return 1;
        }
    }
    return 0;
}

static HHOOK g_keyboardHook = NULL;

// ———— 动态 API 解析（避免导入表特征） ————
// SetWindowsHookEx / CallNextHookEx / UnhookWindowsHookEx / GetAsyncKeyState
// 在使用键盘钩子的程序中常被杀毒软件静态标记，改用运行时 GetProcAddress 解析
// 可使这些 API 不出现在 PE 导入表中，降低静态检测命中率

typedef HHOOK (WINAPI *fnSetWindowsHookEx_t)(int, HOOKPROC, HINSTANCE, DWORD);
typedef LRESULT (WINAPI *fnCallNextHookEx_t)(HHOOK, int, WPARAM, LPARAM);
typedef BOOL (WINAPI *fnUnhookWindowsHookEx_t)(HHOOK);
typedef SHORT (WINAPI *fnGetAsyncKeyState_t)(int);

static fnSetWindowsHookEx_t dyn_SetWindowsHookExA = NULL;
static fnCallNextHookEx_t dyn_CallNextHookEx = NULL;
static fnUnhookWindowsHookEx_t dyn_UnhookWindowsHookEx = NULL;
static fnGetAsyncKeyState_t dyn_GetAsyncKeyState = NULL;

// 运行时构造 API 名称字符串（避免明文字符串出现在二进制中）
static void buildApiName(char *buf, int id) {
    switch (id) {
        case 1: // "SetWindowsHookExA"
            buf[0]=0x53;buf[1]=0x65;buf[2]=0x74;buf[3]=0x57;buf[4]=0x69;
            buf[5]=0x6E;buf[6]=0x64;buf[7]=0x6F;buf[8]=0x77;buf[9]=0x73;
            buf[10]=0x48;buf[11]=0x6F;buf[12]=0x6F;buf[13]=0x6B;buf[14]=0x45;
            buf[15]=0x78;buf[16]=0x41;buf[17]=0; break;
        case 2: // "CallNextHookEx"
            buf[0]=0x43;buf[1]=0x61;buf[2]=0x6C;buf[3]=0x6C;buf[4]=0x4E;
            buf[5]=0x65;buf[6]=0x78;buf[7]=0x74;buf[8]=0x48;buf[9]=0x6F;
            buf[10]=0x6F;buf[11]=0x6B;buf[12]=0x45;buf[13]=0x78;buf[14]=0; break;
        case 3: // "UnhookWindowsHookEx"
            buf[0]=0x55;buf[1]=0x6E;buf[2]=0x68;buf[3]=0x6F;buf[4]=0x6F;
            buf[5]=0x6B;buf[6]=0x57;buf[7]=0x69;buf[8]=0x6E;buf[9]=0x64;
            buf[10]=0x6F;buf[11]=0x77;buf[12]=0x73;buf[13]=0x48;buf[14]=0x6F;
            buf[15]=0x6F;buf[16]=0x6B;buf[17]=0x45;buf[18]=0x78;buf[19]=0; break;
        case 4: // "GetAsyncKeyState"
            buf[0]=0x47;buf[1]=0x65;buf[2]=0x74;buf[3]=0x41;buf[4]=0x73;
            buf[5]=0x79;buf[6]=0x6E;buf[7]=0x63;buf[8]=0x4B;buf[9]=0x65;
            buf[10]=0x79;buf[11]=0x53;buf[12]=0x74;buf[13]=0x61;buf[14]=0x74;
            buf[15]=0x65;buf[16]=0; break;
    }
}

static int initDynamicApis(void) {
    HMODULE hUser32 = GetModuleHandleA("user32.dll");
    if (!hUser32) return -1;

    char nameBuf[64];
    buildApiName(nameBuf, 1); dyn_SetWindowsHookExA = (fnSetWindowsHookEx_t)GetProcAddress(hUser32, nameBuf);
    buildApiName(nameBuf, 2); dyn_CallNextHookEx = (fnCallNextHookEx_t)GetProcAddress(hUser32, nameBuf);
    buildApiName(nameBuf, 3); dyn_UnhookWindowsHookEx = (fnUnhookWindowsHookEx_t)GetProcAddress(hUser32, nameBuf);
    buildApiName(nameBuf, 4); dyn_GetAsyncKeyState = (fnGetAsyncKeyState_t)GetProcAddress(hUser32, nameBuf);

    if (!dyn_SetWindowsHookExA || !dyn_CallNextHookEx || !dyn_UnhookWindowsHookEx || !dyn_GetAsyncKeyState)
        return -1;
    return 0;
}

// ———— 键盘事件通知（供前端监听） ————
static volatile LONG g_kbEventCounter = 0;
static char g_kbEventKey[128] = {0};

static LONG getKbEventCounterC(void) { return g_kbEventCounter; }

static void popKbEventC(char* buf, int bufsize) {
    strcpy_s(buf, bufsize, g_kbEventKey);
    g_kbEventKey[0] = '\0';
    InterlockedExchange(&g_kbEventCounter, 0);
}

static void pushKbEvent(const char* key) {
    strcpy_s(g_kbEventKey, sizeof(g_kbEventKey), key);
    InterlockedIncrement(&g_kbEventCounter);
}

// 构建按键描述字符串并查询 C 层屏蔽列表
// 格式: "Alt+Tab", "Ctrl+Shift+S", "Super_L"
// 返回 1=应拦截, 0=放行
static int checkDynamicRegistry(DWORD vk, DWORD flags) {
    char desc[128] = {0};

    // 检测修饰键
    if (flags & LLKHF_ALTDOWN) {
        if (vk != VK_MENU) { strcat_s(desc, sizeof(desc), "Alt+"); }
    }
    if (dyn_GetAsyncKeyState(VK_CONTROL) & 0x8000) {
        if (vk != VK_CONTROL) { strcat_s(desc, sizeof(desc), "Ctrl+"); }
    }
    if (dyn_GetAsyncKeyState(VK_SHIFT) & 0x8000) {
        if (vk != VK_SHIFT) { strcat_s(desc, sizeof(desc), "Shift+"); }
    }
    // 注意：Super 键 (VK_LWIN/VK_RWIN) 不作为修饰符前缀，而是直接作为键名处理

    // 将虚拟键码转为名称
    switch (vk) {
        case VK_LWIN:       strcat_s(desc, sizeof(desc), "Super_L"); break;
        case VK_RWIN:       strcat_s(desc, sizeof(desc), "Super_R"); break;
        case VK_MENU:       strcat_s(desc, sizeof(desc), "Alt_L"); break;
        case VK_CONTROL:    strcat_s(desc, sizeof(desc), "Control_L"); break;
        case VK_SHIFT:      strcat_s(desc, sizeof(desc), "Shift_L"); break;
        case VK_TAB:        strcat_s(desc, sizeof(desc), "Tab"); break;
        case VK_F4:         strcat_s(desc, sizeof(desc), "F4"); break;
        case VK_ESCAPE:     strcat_s(desc, sizeof(desc), "Esc"); break;
        case VK_SPACE:      strcat_s(desc, sizeof(desc), "Space"); break;
        case VK_RETURN:     strcat_s(desc, sizeof(desc), "Enter"); break;
        case VK_BACK:       strcat_s(desc, sizeof(desc), "Backspace"); break;
        case VK_DELETE:     strcat_s(desc, sizeof(desc), "Delete"); break;
        case VK_INSERT:     strcat_s(desc, sizeof(desc), "Insert"); break;
        case VK_HOME:       strcat_s(desc, sizeof(desc), "Home"); break;
        case VK_END:        strcat_s(desc, sizeof(desc), "End"); break;
        case VK_PRIOR:      strcat_s(desc, sizeof(desc), "PageUp"); break;
        case VK_NEXT:       strcat_s(desc, sizeof(desc), "PageDown"); break;
        case VK_LEFT:       strcat_s(desc, sizeof(desc), "Left"); break;
        case VK_RIGHT:      strcat_s(desc, sizeof(desc), "Right"); break;
        case VK_UP:         strcat_s(desc, sizeof(desc), "Up"); break;
        case VK_DOWN:       strcat_s(desc, sizeof(desc), "Down"); break;
        case VK_F1: case VK_F2: case VK_F3: case VK_F5: case VK_F6: case VK_F7:
        case VK_F8: case VK_F9: case VK_F10: case VK_F11: case VK_F12:
        {
            char tmp[16];
            sprintf_s(tmp, sizeof(tmp), "F%d", vk - VK_F1 + 1);
            strcat_s(desc, sizeof(desc), tmp);
            break;
        }
        default: {
            // 字母 A-Z
            if (vk >= 'A' && vk <= 'Z') {
                char tmp[2] = { (char)vk, 0 };
                strcat_s(desc, sizeof(desc), tmp);
            }
            // 数字 0-9
            else if (vk >= '0' && vk <= '9') {
                char tmp[2] = { (char)vk, 0 };
                strcat_s(desc, sizeof(desc), tmp);
            }
            // 其他: 用 VK_ 编号
            else {
                char tmp[32];
                sprintf_s(tmp, sizeof(tmp), "VK_%d", (int)vk);
                strcat_s(desc, sizeof(desc), tmp);
            }
            break;
        }
    }

    // 查询 C 层屏蔽列表（纯 C 检查，不回调 Go）
    if (isKeyBlockedC(desc)) {
        pushKbEvent(desc);
        return 1;
    }
    return 0;
}

// 低层键盘钩子回调 — 拦截快捷键
static LRESULT CALLBACK KeyboardHookProc(int nCode, WPARAM wParam, LPARAM lParam) {
    if (nCode >= 0) {
        KBDLLHOOKSTRUCT *pKb = (KBDLLHOOKSTRUCT*)lParam;
        DWORD vk = pKb->vkCode;

        if (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN) {
            // ★ 动态注册中心（前端通过 bridge 注册的快捷键 + 默认拦截列表）
            if (checkDynamicRegistry(vk, pKb->flags)) {
                return 1;
            }
        }
    }
    return dyn_CallNextHookEx(NULL, nCode, wParam, lParam);
}

// 安装低层键盘钩子
static int enableKeyboardHook(void) {
    if (g_keyboardHook != NULL) {
        return 0;
    }
    if (initDynamicApis() != 0) {
        OutputDebugStringA("[native] initDynamicApis failed");
        return -1;
    }
    g_keyboardHook = dyn_SetWindowsHookExA(WH_KEYBOARD_LL, KeyboardHookProc, NULL, 0);
    if (g_keyboardHook == NULL) {
        return -1;
    }
    return 0;
}

// 卸载低层键盘钩子
static void disableKeyboardHook(void) {
    if (g_keyboardHook != NULL && dyn_UnhookWindowsHookEx != NULL) {
        dyn_UnhookWindowsHookEx(g_keyboardHook);
        g_keyboardHook = NULL;
    }
}

// ———— 输入穿透（精准按区域穿透到下层窗口） ————
//
// 架构：
//   前端 JS 通过 bridge.setOpaqueRegions() 上报不透光区域的矩形列表，
//   C 层在 WM_NCHITTEST 中判断命中坐标是否在不透光区域内：
//     - 在区域内 → 返回 HTCLIENT（本窗口正常捕获）
//     - 不在区域内 → 返回 HTTRANSPARENT（穿透到下层窗口）
//
// 两种模式：
//   模式 A（全局穿透）：opaqueRegionCount == 0 且 inputPassthrough 启用
//     → 使用 WS_EX_TRANSPARENT 全局穿透（适合纯视觉叠加层）
//   模式 B（精准穿透）：opaqueRegionCount > 0
//     → 移除 WS_EX_TRANSPARENT，通过 WM_NCHITTEST 返回 HTTRANSPARENT 精准控制
//     （适合有交互元素的页面，只有透明区域穿透）

// 清空不透光区域列表
static void clearOpaqueRegionsC(void) {
    g_opaqueRegionCount = 0;
}

// 添加一个不透光矩形（坐标相对于窗口客户区原点）
static void addOpaqueRegionC(int x, int y, int w, int h) {
    if (g_opaqueRegionCount >= MAX_OPAQUE_REGIONS) return;
    if (w <= 0 || h <= 0) return;
    g_opaqueRegions[g_opaqueRegionCount].left   = x;
    g_opaqueRegions[g_opaqueRegionCount].top    = y;
    g_opaqueRegions[g_opaqueRegionCount].right  = x + w;
    g_opaqueRegions[g_opaqueRegionCount].bottom = y + h;
    g_opaqueRegionCount++;
}

// 设置穿透模式开关
static void setPassthroughEnabledC(BOOL enabled) {
    g_passthroughEnabled = enabled;
    if (!enabled) {
        clearOpaqueRegionsC();
    }
}

static BOOL isPassthroughEnabledC(void) { return g_passthroughEnabled; }
static int getOpaqueRegionCountC(void) { return g_opaqueRegionCount; }

// 启用全局穿透（WS_EX_TRANSPARENT，适合纯视觉叠加层）
static void enableGlobalPassthrough(HWND hwnd) {
    LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
    SetWindowLong(hwnd, GWL_EXSTYLE, exStyle | WS_EX_LAYERED | WS_EX_TRANSPARENT);
    SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
        SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_NOACTIVATE);
}

// 禁用全局穿透（移除 WS_EX_TRANSPARENT，保留 WS_EX_LAYERED）
static void disableGlobalPassthrough(HWND hwnd) {
    LONG exStyle = GetWindowLong(hwnd, GWL_EXSTYLE);
    SetWindowLong(hwnd, GWL_EXSTYLE, exStyle & ~WS_EX_TRANSPARENT);
    SetWindowPos(hwnd, NULL, 0, 0, 0, 0,
        SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_NOACTIVATE);
}

static int isGlobalPassthroughC(HWND hwnd) {
    return (GetWindowLong(hwnd, GWL_EXSTYLE) & WS_EX_TRANSPARENT) ? 1 : 0;
}

// ———— 穿透：WM_NCHITTEST 自动记录并推送鼠标位置 ————
// WM_NCHITTEST 被 WS_EX_TRANSPARENT 屏蔽，改用 Go goroutine 保底轮询。
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
	dbgLog("[go] Apply called: Borderless=%v Maximized=%v Fullscreen=%v AlwaysOnTop=%v KeyboardShortcuts=%v Opacity=%.2f",
		cfg.Borderless, cfg.Maximized, cfg.Fullscreen, cfg.AlwaysOnTop, cfg.KeyboardShortcuts, cfg.Opacity)
	fmt.Fprintf(os.Stderr, "[WTD] Apply running: KeyboardShortcuts=%v\n", cfg.KeyboardShortcuts)
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

	// 6. 键盘快捷键拦截（按键映射提前注册，钩子安装延迟到 webview.go 中执行）
	if cfg.KeyboardShortcuts {
		if len(cfg.KeyMappings) > 0 {
			InitKeyMappings(cfg.KeyMappings)
		}
	}

	// 7. 窗口图标（从 EXE 资源加载，ID=1）
	C.setWindowIcon(hwnd)

	// 8. 系统托盘（必须在窗口显示后创建）
	if cfg.SystemTray {
		if cfg.TrayHideTaskbar {
			C.hideFromTaskbar(hwnd)
		}
		C.enableSystemTray(hwnd)
		C.createTrayIcon(hwnd)
	} else {
		C.disableSystemTray(hwnd)
	}

	// 9. 输入穿透
	if cfg.InputPassthrough {
		C.setPassthroughEnabledC(1)
		C.enableGlobalPassthrough(hwnd)
	} else {
		C.setPassthroughEnabledC(0)
		C.disableGlobalPassthrough(hwnd)
		C.clearOpaqueRegionsC()
	}
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

// InitSystemTray 初始化系统托盘图标（安装 TrayWndProc + 创建托盘图标）。
// 需在 Dispatch 回调中调用。
func InitSystemTray(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	hwnd := (C.HWND)(winPtr)
	C.enableSystemTray(hwnd)
	C.createTrayIcon(hwnd)
}

// RemoveSystemTray 移除系统托盘图标。
func RemoveSystemTray(winPtr unsafe.Pointer) {
	C.removeTrayIcon()
}

// ShowWindowRestore 显示窗口并置前（用于托盘"显示窗口"）。
func ShowWindowRestore(winPtr unsafe.Pointer) {
	hwnd := (C.HWND)(winPtr)
	C.ShowWindow(hwnd, C.SW_SHOW)
	C.SetForegroundWindow(hwnd)
}

// EnableInputPassthrough 启用全局输入穿透（WS_EX_TRANSPARENT，适合纯视觉叠加层）。
// 注意：此方法使整个窗口穿透（包括不透明区域）。如需精准穿透，
// 请使用 SetOpaqueRegions 设置不透光区域列表。
// 需在 Dispatch 回调中调用。
func EnableInputPassthrough(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.setPassthroughEnabledC(1)
	C.enableGlobalPassthrough((C.HWND)(winPtr))
}

// DisableInputPassthrough 禁用输入穿透（恢复窗口正常鼠标事件处理）。
// 需在 Dispatch 回调中调用。
func DisableInputPassthrough(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.setPassthroughEnabledC(0)
	C.disableGlobalPassthrough((C.HWND)(winPtr))
}

// IsInputPassthrough 查询当前是否启用输入穿透（全局穿透或精准穿透均返回 true）。
func IsInputPassthrough(winPtr unsafe.Pointer) bool {
	if winPtr == nil {
		return false
	}
	if C.isPassthroughEnabledC() != 0 {
		return true
	}
	return C.isGlobalPassthroughC((C.HWND)(winPtr)) != 0
}

// OpaqueRegion 不透光矩形（相对于窗口客户区原点）。
type OpaqueRegion struct {
	X, Y, W, H int
}

// SetOpaqueRegions 设置不透光区域列表（精准穿透模式）。
// JS 侧通过检测像素透明度或 CSS 标记，将不透光区域上报到原生层。
// 设置后 WM_NCHITTEST 只在 opaque 区域内返回 HTCLIENT，其余区域返回 HTTRANSPARENT。
//
// 用法：
//  1. 关闭全局穿透：DisableInputPassthrough(winPtr)
//  2. 设置不透光区域：SetOpaqueRegions(winPtr, regions)
//
// 此时 WndProc（BorderlessWndProc 或 PassthroughWndProc）会根据区域列表
// 决定哪些坐标穿透、哪些捕获。
// 需在 Dispatch 回调中调用。
func SetOpaqueRegions(winPtr unsafe.Pointer, regions []OpaqueRegion) {
	if winPtr == nil {
		return
	}
	C.clearOpaqueRegionsC()
	for _, r := range regions {
		C.addOpaqueRegionC(C.int(r.X), C.int(r.Y), C.int(r.W), C.int(r.H))
	}
}

// ClearOpaqueRegions 清空不透光区域列表。
// 清空后所有区域穿透（因为 opaqueRegionCount==0，WndProc 返回 HTTRANSPARENT）。
// 需在 Dispatch 回调中调用。
func ClearOpaqueRegions(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.clearOpaqueRegionsC()
}

// ———— 鼠标位置轮询 ————

// PollCursorPos 保底轮询：GetCursorPos（WM_NCHITTEST 不触发时的后备）
func PollCursorPos(winPtr unsafe.Pointer) (x, y int32) {
	if winPtr == nil {
		return -1, -1
	}
	var pt C.POINT
	C.GetCursorPos(&pt)
	C.ScreenToClient((C.HWND)(winPtr), &pt)
	return int32(pt.x), int32(pt.y)
}

// HasDisplay 在 Windows 上始终返回 true（WebView2 无需 X11/Wayland）。
func HasDisplay() bool { return true }

// syncBlockedKeysToC 将 Go 注册表中的快捷键同步到 C 层静态数组。
// keys 为 nil 时清空 C 层数组。
func syncBlockedKeysToC(keys []string) {
	C.clearBlockedKeysC()
	for _, k := range keys {
		cKey := C.CString(k)
		C.addBlockedKeyC(cKey)
		C.free(unsafe.Pointer(cKey))
	}
}

// syncKeyMappingsToC — 按键映射已移除，保留为空桩避免编译错误
func syncKeyMappingsToC(mappings map[string]string) {}

// EnableKeyboardHook 安装低层键盘钩子拦截注册的快捷键。
// 必须在 UI 线程（具有消息循环）上调用，典型调用位置为 w.Dispatch 回调中。
// defaults: 构建时配置的默认拦截快捷键列表。
func EnableKeyboardHook(winPtr unsafe.Pointer, defaults []string) {
	if winPtr == nil {
		return
	}
	// 初始化默认快捷键并同步到 C 层
	InitDefaultBlockedShortcuts(defaults)

	// 安装钩子并检查结果
	ret := C.enableKeyboardHook()
	if ret != 0 {
		dbgLog("[go] enableKeyboardHook failed (SetWindowsHookEx returned NULL)")
		fmt.Fprintf(os.Stderr, "[WTD] ERROR: enableKeyboardHook failed\n")
	} else {
		dbgLog("[go] enableKeyboardHook installed successfully")
		fmt.Fprintf(os.Stderr, "[WTD] enableKeyboardHook installed successfully\n")
	}
}

// DisableKeyboardHook 卸载低层键盘钩子，恢复系统快捷键。
func DisableKeyboardHook(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.disableKeyboardHook()
	ClearShortcuts()
}

// PollKbEvent 查询是否有被拦截的键盘事件。
// 返回事件描述字符串，无事件时返回空字符串。
// 每次调用仅返回最近一次事件（消费后重置）。
func PollKbEvent() string {
	if C.getKbEventCounterC() == 0 {
		return ""
	}
	var buf [128]C.char
	C.popKbEventC(&buf[0], 128)
	return C.GoString(&buf[0])
}
