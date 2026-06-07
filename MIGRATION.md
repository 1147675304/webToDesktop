# 背景透明实现方案

> 从 git 原始代码出发，实现窗口毛玻璃 + WebView2 控件背景透明的完整方案。

## 涉及文件（9个）

| # | 文件 | 行号 | 改动 |
|---|------|------|------|
| 1 | `config.yaml` | L44 | 新增 `webview_bg_transparent` |
| 2 | `config.go` | L49 | `WindowConfig` 新增字段 |
| 3 | `native/config.go` | L12 | `WindowConfig` 新增字段 |
| 4 | `webview_cc.cc` | L110 | 新增 `wtd_webview_get_native_handle` |
| 5 | `webview_cgo.go` | L29, L105 | 新增 C 声明 + Go 方法 |
| 6 | `webview.go` | L40 | 修改初始化流程 |
| 7 | `native/windows.go` | 整体 | 新增 C 函数 + Go 函数 |
| 8 | `native/linux.go` | 末尾 | 新增空桩 |
| 9 | `native/other.go` | 末尾 | 新增空桩 |

---

## 1. `config.yaml` — 窗口样式段

原代码：
```yaml
  opacity: 1       # 窗口不透明度（0.0=全透明，1.0=不透明）
                        # 前端无需适配
```

改为（在 `opacity` 后加一行）：
```yaml
  opacity: 1       # 窗口整体不透明度（0.0=全透明，1.0=不透明）
                        # 前端无需适配
  webview_bg_transparent: false  # WebView2 控件背景透明（Windows 专用）
                        # true : 控件背景透明，毛玻璃可透过网页透明区域显示
                        # false: 控件背景不透明，网页正常渲染不受影响
                        # 启用 Acrylic 时自动视为 true
```

---

## 2. `config.go` — WindowConfig 结构体

找到 `Opacity` 字段，在它下面加一行：

```go
Opacity              float64 `yaml:"opacity"`               // 窗口整体透明度
WebViewBgTransparent bool    `yaml:"webview_bg_transparent"` // 新增：WebView2 控件背景透明
```

---

## 3. `native/config.go` — WindowConfig 结构体

找到 `Opacity` 字段，在它下面加一行：

```go
Opacity              float64  // 窗口整体透明度
WebViewBgTransparent bool     // 新增：WebView2 控件背景透明（Windows 专用）
```

---

## 4. `webview_cc.cc` — 新增 C 包装函数

在 `extern "C"` 块末尾（`} // extern "C"` 之前），添加：

```c
void* wtd_webview_get_native_handle(webview_t w, int kind) {
    return webview_get_native_handle(w, (webview_native_handle_kind_t)kind);
}
```

完整上下文：
```c
    webview_error_t wtd_webview_dispatch(webview_t w, unsigned long long index) {
        dispatch_ctx_t* ctx = (dispatch_ctx_t*)malloc(sizeof(dispatch_ctx_t));
        ctx->index = index;
        return webview_dispatch(w, dispatch_callback_c, ctx);
    }

    // +++ 新增 +++
    void* wtd_webview_get_native_handle(webview_t w, int kind) {
        return webview_get_native_handle(w, (webview_native_handle_kind_t)kind);
    }

} // extern "C"
```

---

## 5. `webview_cgo.go`

### 5a. C 声明区（在 `extern webview_error_t wtd_webview_dispatch...` 后加一行）

```c
extern void*          wtd_webview_get_native_handle(webview_t w, int kind);
```

### 5b. Go 类型和方法（在 `const` 块之后，`func boolToInt` 之前）

```go
// NativeHandleKind 原生句柄类型
type NativeHandleKind int

const (
	NativeHandleKindWindow             NativeHandleKind = 0 // WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW
	NativeHandleKindWidget             NativeHandleKind = 1 // WEBVIEW_NATIVE_HANDLE_KIND_UI_WIDGET
	NativeHandleKindBrowserController  NativeHandleKind = 2 // WEBVIEW_NATIVE_HANDLE_KIND_BROWSER_CONTROLLER
)

func (wv *WebView) GetNativeHandle(kind NativeHandleKind) unsafe.Pointer {
	return C.wtd_webview_get_native_handle(wv.w, C.int(kind))
}
```

---

## 6. `webview.go` — 修改初始化流程

### 6a. 环境变量设置（改条件判断）

原代码：
```go
// 设置 WebView2 默认背景为透明（需在创建任何 WebView2 实例之前）
os.Setenv("WEBVIEW2_DEFAULT_BACKGROUND_COLOR", "00000000")
```

改为条件设置：
```go
// 步骤①：设置 WebView2 环境变量（仅初始化时生效）
needTransparentBg := AppCfg.Window.Acrylic || AppCfg.Window.WebViewBgTransparent
if needTransparentBg {
    os.Setenv("WEBVIEW2_DEFAULT_BACKGROUND_COLOR", "00000000")
}
```

### 6b. 窗口创建后（在 `w := NewWebView(true)` 之后、`title` 之前）

```go
w := NewWebView(true)
defer w.Destroy()

// 步骤②：运行时强制设置 WebView2 背景色
if needTransparentBg {
    ctrl := w.GetNativeHandle(NativeHandleKindBrowserController)
    if ctrl != nil {
        native.SetWebView2BackgroundColor(ctrl, 0, 0, 0, 0) // ARGB 全透明
    }
}
```

### 6c. `SetSize` 段（替换原来的单行 SetSize）

原代码：
```go
w.SetSize(AppCfg.Window.Width, AppCfg.Window.Height, HintNone)
```

改为：
```go
// ③ 先 1x1 显示触发 DWM 合成，再全尺寸确保毛玻璃全面覆盖
w.SetSize(1, 1, HintNone)
if needTransparentBg {
    native.ReapplyAcrylic(w.Window())
}
w.SetSize(AppCfg.Window.Width, AppCfg.Window.Height, HintNone)
```

### 6d. `winCfg` 构造

```go
winCfg := native.WindowConfig{
    // ... 现有字段 ...
    WebViewBgTransparent: AppCfg.Window.Acrylic || AppCfg.Window.WebViewBgTransparent, // 新增
    // ... 现有字段 ...
}
```

---

## 7. `native/windows.go` — 主战场

### 7a. C 层 — WebView2 背景色 COM 接口

在 `/*` C 块内，`#include <string.h>` 之后添加：

```c
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
```

### 7b. C 层 — resize 后重设背景

在 `closeWindow` 函数定义之后添加：

```c
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
```

### 7c. C 层 — 无边框窗口边缘 resize 子类化

在 `HTxxx` 常量定义之后，`startWindowDrag` 之前添加：

```c
// ———— 无边框窗口边缘 resize：子类化 WndProc 拦截 WM_NCHITTEST ————
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
        reapplyAcrylicAndFlush(hwnd);
    }
    return CallWindowProc(g_origBorderlessProc, hwnd, msg, wp, lp);
}

static void enableBorderlessResize(HWND hwnd) {
    if (g_origBorderlessProc) return;
    g_origBorderlessProc = (WNDPROC)SetWindowLongPtr(hwnd, GWLP_WNDPROC, (LONG_PTR)BorderlessWndProc);
}
```

### 7d. Go 层 — 新增导出函数

在 `import` 块之后，`ApplyPreShow` 之前添加：

```go
func SetWebView2BackgroundColor(ctrlPtr unsafe.Pointer, a, r, g, b byte) {
    C.setWebView2BgColor(ctrlPtr, C.BYTE(a), C.BYTE(r), C.BYTE(g), C.BYTE(b))
}

func ReapplyAcrylic(winPtr unsafe.Pointer) {
    if winPtr == nil { return }
    C.reapplyAcrylicAndFlush((C.HWND)(winPtr))
}

func EnableBorderlessResize(winPtr unsafe.Pointer) {
    if winPtr == nil { return }
    C.enableBorderlessResize((C.HWND)(winPtr))
}
```

### 7e. Go 层 — 修改 `Apply` 函数

在 `if cfg.Borderless { C.setBorderless(hwnd) }` 块内加一行：

```go
if cfg.Borderless {
    C.setBorderless(hwnd)
    C.enableBorderlessResize(hwnd)  // 新增：子类化 WM_NCHITTEST
}
```

---

## 8. `native/linux.go` — 空桩

在文件末尾添加：

```go
func SetWebView2BackgroundColor(ctrlPtr unsafe.Pointer, a, r, g, b byte) {}
func ReapplyAcrylic(winPtr unsafe.Pointer) {}
func EnableBorderlessResize(winPtr unsafe.Pointer) {}
```

---

## 9. `native/other.go` — 空桩

在文件末尾添加：

```go
func SetWebView2BackgroundColor(ctrlPtr unsafe.Pointer, a, r, g, b byte) {}
func ReapplyAcrylic(winPtr unsafe.Pointer) {}
func EnableBorderlessResize(winPtr unsafe.Pointer) {}
```

---

## 附录：背景透明原理

```
┌─ 网页层 ──────────────────────────────────────┐
│  body { background: transparent }              │  ← 前端设置 CSS 透明
│  透明区域向下看 →                               │
├─ WebView2 控件层 ─────────────────────────────┤
│  put_DefaultBackgroundColor({A=0,R=0,G=0,B=0}) │  ← COM 运行时设置
│  环境变量 WEBVIEW2_DEFAULT_BG_COLOR=00000000   │  ← 初始化时兜底
│  透明区域向下看 →                               │
├─ DWM 窗口层 ──────────────────────────────────┤
│  DwmEnableBlurBehindWindow(全区域模糊)          │  ← Win10+毛玻璃 API
│  透明区域向下看 →                               │
├─ 桌面层 ──────────────────────────────────────┤
│  用户桌面背景（被模糊后显示）                     │
└───────────────────────────────────────────────┘
```

### 各层职责

| 层 | API | 作用 | 调用时机 |
|----|-----|------|---------|
| WebView2 | `put_DefaultBackgroundColor(A=0)` | 控件背景透明 | 窗口创建后立即 |
| WebView2 | `WEBVIEW2_DEFAULT_BG_COLOR` env var | 环境变量兜底 | 窗口创建前 |
| DWM | `DwmEnableBlurBehindWindow` | 窗口毛玻璃模糊 | ApplyPreShow / resize后 |
| DWM | `DwmFlush` | 同步 DWM 合成 | 每次设完毛玻璃后 |
| 窗口 | `WM_EXITSIZEMOVE` + `reapplyAcrylicAndFlush` | resize 后重设背景 | 每次 resize 结束 |

---

## 构建

```bash
make build-windows PROJECT=huangjin
```

配置文件 `config.yaml` 中启用：
```yaml
window:
  acrylic: true                    # DWM 毛玻璃
  webview_bg_transparent: true     # WebView2 控件透明
  borderless: true                 # 无边框（自动启用原生边缘 resize）
```
