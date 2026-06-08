// tools/desktop/pkg/webview.go
// 原生 WebView 窗口
package pkg

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/lhpanda/webtodesktop/native"
	webview "github.com/webview/webview_go"
)

// BridgeBinder 桥接绑定接口，由 bridge.Bridge 实现。
// 避免 pkg → pkg/bridge 循环导入。
type BridgeBinder interface {
	SetWebView(wv webview.WebView)
	Call(method string, params map[string]interface{}) (map[string]interface{}, error)
}

const injectCSS = `
/* 自定义滚动条 */
::-webkit-scrollbar { width: 0px; height: 0px; }
::-webkit-scrollbar-thumb { background: rgba(253, 0, 0, 0); border-radius: 4px; }
::-webkit-scrollbar-thumb:hover { background: rgb(255, 0, 0); }
::-webkit-scrollbar-track { background: transparent; }

/* 禁止文本选择（桌面应用体验），输入框除外 */
body { -webkit-user-select: none; user-select: none; }
input, textarea, [contenteditable] { -webkit-user-select: text; user-select: text; }
`

var debugMode bool

func init() {
	debugMode = os.Getenv("WTD_DEBUG") == "1"
}

func dbg(format string, args ...interface{}) {
	if debugMode {
		fmt.Printf("[WTD] "+format+"\n", args...)
	}
}

// buildInjectJS 构建注入到 WebView 的初始化 JS。
// debug 模式下包含 console.log 调试输出。
func buildInjectJS(css string) string {
	js := `(function injectCSS(){
		if (!document.head) { setTimeout(injectCSS, 10); return; }
		var s=document.createElement('style');
		s.textContent=%q;
		document.head.appendChild(s);`
	if !debugMode {
		js += `
		document.addEventListener('contextmenu', function(e){ e.preventDefault(); });`
	}
	js += `
		/* 选中文本自动复制到剪贴板 */
		document.addEventListener('mouseup', function(){
			var sel = window.getSelection().toString().trim();
			if (sel) {
				navigator.clipboard.writeText(sel).then(function(){`
	if debugMode {
		js += `
					console.log('[WTD] copied: ' + sel.substring(0, 50));`
	}
	js += `
				}).catch(function(){});
			}
		});`
	if debugMode {
		js += `
		console.log('[WTD] CSS injected');`
	}
	js += `
	})();`
	return fmt.Sprintf(js, css)
}

// RunApp 使用系统原生 WebView 创建独立窗口。
// br 是桥接绑定器（bridge.Bridge），关联 WebView 窗口后绑定到 JS。
func RunApp(addr string, server *http.Server, store *Store, projectName string, br BridgeBinder) {
	dbg("=== runApp start ===")
	dbg("addr=%s, width=%d, height=%d, borderless=%v",
		addr, AppCfg.Window.Width, AppCfg.Window.Height, AppCfg.Window.Borderless)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ★ 平台特定初始化（Windows: 设置 WebView2 透明背景等）
	platformInit()

	dbg("creating WebView...")
	w := webview.New(debugMode)
	dbg("WebView created")
	defer func() {
		dbg("destroying WebView...")
		w.Destroy()
		dbg("WebView destroyed")
	}()

	w.SetSize(1, 1, webview.HintNone)
	dbg("initial size set to 1x1")

	title := projectName
	if title == "" {
		title = AppCfg.Window.Title
	}
	if title == "" {
		title = "WebToDesktop"
	}
	w.SetTitle(title)
	dbg("title set: %s", title)

	winCfg := native.WindowConfig{
		Opacity:              AppCfg.Window.Opacity,
		Borderless:           AppCfg.Window.Borderless,
		AlwaysOnTop:          AppCfg.Window.AlwaysOnTop,
		Fullscreen:           AppCfg.Window.Fullscreen,
		Maximized:            AppCfg.Window.Maximized,
		WindowPosition:       AppCfg.Window.WindowPosition,
		DarkTitleBar:         AppCfg.Window.DarkTitleBar,
		RoundCorners:         AppCfg.Window.RoundCorners,
		Acrylic:              AppCfg.Window.Acrylic,
		WebViewBgTransparent: AppCfg.Window.Acrylic || AppCfg.Window.WebViewBgTransparent,
	}

	dbg("phase 1: ApplyPreShow...")
	native.ApplyPreShow(w.Window(), winCfg)

	dbg("setting size to %dx%d", AppCfg.Window.Width, AppCfg.Window.Height)
	w.SetSize(AppCfg.Window.Width, AppCfg.Window.Height, webview.HintNone)
	native.SetDefaultWindowSize(AppCfg.Window.Width, AppCfg.Window.Height)

	// ★ 平台特定窗口显示后处理（Windows: 重新应用 Acrylic 毛玻璃）
	platformOnShow(w.Window())

	if store != nil && br != nil {
		dbg("setting up bridge...")
		br.SetWebView(w) // ★ 关键：绑定 WebView 窗口句柄，窗口控制方法才能 Dispatch
		if err := w.Bind("__lhpanda__", br.Call); err != nil {
			dbg("ERROR: Bind failed: %v", err)
		} else {
			dbg("bridge ready: window.__lhpanda__")
		}
	}

	dbg("Init: injecting CSS + contextmenu block...")
	w.Init(buildInjectJS(injectCSS))

	dbg("Navigate: %s", addr)
	w.Navigate(addr)

	dbg("phase 2: Dispatch (post-show config)...")
	w.Dispatch(func() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					dbg("PANIC in Dispatch: %v\n%s", r, debug.Stack())
				}
			}()
			native.Apply(w.Window(), winCfg)
			platformOnShow(w.Window())
		}()
	})

	go func() {
		<-sigCh
		dbg("signal received, terminating...")
		w.Terminate()
	}()

	dbg("entering main loop (w.Run)...")
	w.Run()
	dbg("main loop exited")

	ShutdownServer(server, 3*time.Second)
	dbg("=== runApp end ===")
}
