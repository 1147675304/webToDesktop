// tools/desktop/pkg/webview.go
// 原生 WebView 窗口
package pkg

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/lhpanda/webtodesktop/native"
	webview "github.com/webview/webview_go"
)

// isWSL 检测当前是否运行在 WSL（Windows Subsystem for Linux）环境中。
func isWSL() bool {
	// 方式1: 检查 WSL 专有文件
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		return true
	}
	// 方式2: 检查 /proc/version 是否包含 "microsoft" 或 "WSL"
	if data, err := os.ReadFile("/proc/version"); err == nil {
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl") {
			return true
		}
	}
	// 方式3: 检查 WSL 环境变量
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	return false
}

// isLoongArch 检测当前是否运行在龙芯 (loong64) 架构。
func isLoongArch() bool {
	return runtime.GOARCH == "loong64"
}

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
var DisableContextmenu bool // 由 main 包根据构建参数设置

// BuildDebug 由构建工具通过 ldflags 注入（console 构建目标为 "true"）。
// 作为 WTD_DEBUG 环境变量的编译时回退。
var BuildDebug string

func init() {
	debugMode = os.Getenv("WTD_DEBUG") == "1" || BuildDebug == "true"
}

func dbg(format string, args ...interface{}) {
	msg := fmt.Sprintf("[WTD] "+format+"\n", args...)
	if debugMode {
		fmt.Print(msg)
	}
	// 始终写入日志文件，避免会话崩溃时丢失日志
	writeLog(msg)
}

// logFile 调试日志文件路径
var logFile string

func initLogFile() {
	if logFile != "" || !debugMode {
		return
	}
	// 仅调试模式写入日志文件
	logFile = filepath.Join(".", "wtd-debug.log")
	os.WriteFile(logFile, nil, 0644)
	abs, _ := filepath.Abs(logFile)
	logFile = abs
}

func writeLog(msg string) {
	if logFile == "" {
		return
	}
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(msg)
	f.Sync() // 立即刷盘，确保崩溃前写入
}

// buildInjectJS 构建注入到 WebView 的初始化 JS。
// 包含 CSS 注入、右键菜单控制、浏览器快捷键禁用、选中复制、以及访问令牌管理。
// accessToken 为空字符串时不注入令牌逻辑。
func buildInjectJS(css string, disableCtxMenu bool, disableBrowserShortcuts bool, accessToken string) string {
	js := `(function injectCSS(){
		if (!document.head) { setTimeout(injectCSS, 10); return; }
		var s=document.createElement('style');
		s.textContent=%q;
		document.head.appendChild(s);`
	if disableCtxMenu && !debugMode {
		js += `
		document.addEventListener('contextmenu', function(e){ e.preventDefault(); });`
	}
	if disableBrowserShortcuts {
		js += `
		/* 禁用 WebView 浏览器内置快捷键（不影响系统其他应用） */
		document.addEventListener('keydown', function(e){
			if (e.ctrlKey && !e.altKey && !e.metaKey) {
				switch (e.key) {
					case 's': e.preventDefault(); break; /* Ctrl+S 保存网页 */
					case 'p': e.preventDefault(); break; /* Ctrl+P 打印 */
					case 'u': e.preventDefault(); break; /* Ctrl+U 查看源码 */
				}
			}
			if (e.key === 'F12') { e.preventDefault(); } /* 开发者工具 */
		});`
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

	// ———— 访问令牌管理 ————
	// 将令牌存储在全局变量中，所有 XHR/fetch 自动携带令牌请求头
	if accessToken != "" {
		js += `
(function(){
	var token = ` + "`" + accessToken + "`" + `;

	// 拦截 fetch，自动添加 X-WTD-Token 请求头
	var origFetch = window.fetch;
	window.fetch = function(url, opts) {
		opts = opts || {};
		opts.headers = opts.headers || {};
		if (opts.headers instanceof Headers) {
			opts.headers.set('X-WTD-Token', token);
		} else {
			opts.headers['X-WTD-Token'] = token;
		}
		return origFetch.call(window, url, opts);
	};

	// 拦截 XMLHttpRequest，自动添加 X-WTD-Token 请求头
	var origOpen = XMLHttpRequest.prototype.open;
	XMLHttpRequest.prototype.open = function(method, url) {
		this.addEventListener('readystatechange', function() {
			if (this.readyState === 1) {
				this.setRequestHeader('X-WTD-Token', token);
			}
		});
		return origOpen.apply(this, arguments);
	};

	// 存储令牌到 window，供前端代码直接使用
	window.__wtd_token__ = token;
})();`
	}

	return fmt.Sprintf(js, css)
}

// RunApp 使用系统原生 WebView 创建独立窗口。
// br 是桥接绑定器（bridge.Bridge），关联 WebView 窗口后绑定到 JS。
// accessToken 是随机令牌，用于防止浏览器直接访问本地服务。
func RunApp(addr, accessToken string, server *http.Server, store *Store, projectName string, br BridgeBinder, devURL string, externalWebDir string, isConsole bool) {
	initLogFile()
	defer func() {
		if logFile != "" {
			dbg("日志文件: %s", logFile)
		}
	}()
	dbg("=== runApp start ===")
	dbg("addr=%s, width=%d, height=%d, borderless=%v",
		addr, AppCfg.Window.Width, AppCfg.Window.Height, AppCfg.Window.Borderless)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ★ 平台特定初始化（Windows: 设置 WebView2 透明背景等）
	platformInit()

	// ★ 龙芯 GPU 硬加速与 WebKit 不兼容，仅禁用 WebView 内容渲染的 GPU 加速
	disableGPU := isLoongArch()

	// 检查桌面显示服务器是否可用（Linux: DISPLAY/WAYLAND_DISPLAY）
	if !native.HasDisplay() {
		if isWSL() {
			// WSL 环境：自动配置 DISPLAY=:0（兼容 WSLg 和常见 X Server）
			fmt.Fprintln(os.Stderr, "[WTD] 检测到 WSL 环境，自动设置 DISPLAY=:0...")
			os.Setenv("DISPLAY", ":0")
			if !native.HasDisplay() {
				// WSLg 可能使用 /mnt/wslg 下的 Wayland socket
				fmt.Fprintln(os.Stderr, "[WTD] 尝试 WSLg Wayland 配置...")
				os.Setenv("WAYLAND_DISPLAY", "wayland-0")
				os.Setenv("XDG_RUNTIME_DIR", "/mnt/wslg/runtime-dir")
				if !native.HasDisplay() {
					fmt.Fprintln(os.Stderr, "\n错误: 未检测到可用的桌面显示服务器。")
					fmt.Fprintln(os.Stderr, "  WSL 环境下请确保已安装 WSLg（Windows 11 默认包含），")
					fmt.Fprintln(os.Stderr, "  或安装 VcXsrv/Xming 等第三方 X Server 并运行: export DISPLAY=$(hostname).local:0")
					fmt.Fprintln(os.Stderr, "  Windows 下更推荐直接使用原生 exe 版本。")
					os.Exit(1)
				}
			}
		} else {
			fmt.Fprintln(os.Stderr, "\n错误: 未检测到可用的桌面显示服务器。")
			fmt.Fprintln(os.Stderr, "  Linux 桌面环境需要运行 X11 或 Wayland 显示服务器。")
			fmt.Fprintln(os.Stderr, "  当前环境可能为无图形界面的服务器或 SSH 会话。")
			os.Exit(1)
		}
	}

	dbg("creating WebView...")
	dbg("STEP-1: webview.New")
	w := webview.New(debugMode)
	dbg("STEP-1: ok, WebView created")
	defer func() {
		dbg("destroying WebView...")
		w.Destroy()
		dbg("WebView destroyed")
	}()

	dbg("STEP-2: SetSize(1,1)")
	w.SetSize(1, 1, webview.HintNone)
	dbg("STEP-2: ok")

	title := projectName
	if title == "" {
		title = AppCfg.Window.Title
	}
	if title == "" {
		title = "WebToDesktop"
	}
	dbg("STEP-3: SetTitle")
	w.SetTitle(title)
	dbg("STEP-3: ok, title=%s", title)

	// ★ 龙芯+麒麟：禁用 WebKit GPU 硬加速（仅影响 WebView 内容渲染）
	if disableGPU {
		native.DisableWebKitHardwareAccel(w.Window())
		dbg("WebKit hardware acceleration disabled for compatibility")
	}

	winCfg := native.WindowConfig{
		Opacity:                 AppCfg.Window.Opacity,
		Borderless:              AppCfg.Window.Borderless,
		AlwaysOnTop:             AppCfg.Window.AlwaysOnTop,
		Fullscreen:              AppCfg.Window.Fullscreen,
		Maximized:               AppCfg.Window.Maximized,
		WindowPosition:          AppCfg.Window.WindowPosition,
		DarkTitleBar:            AppCfg.Window.DarkTitleBar,
		RoundCorners:            AppCfg.Window.RoundCorners,
		Acrylic:                 AppCfg.Window.Acrylic,
		WebViewBgTransparent:    (AppCfg.Window.Acrylic && runtime.GOOS == "windows") || AppCfg.Window.WebViewBgTransparent,
		InputPassthrough:        AppCfg.Window.InputPassthrough,
		SystemTray:              AppCfg.Window.SystemTray,
		TrayHideTaskbar:         AppCfg.Window.TrayHideTaskbar,
		KeyboardShortcuts:       AppCfg.Window.KeyboardShortcuts,
		DefaultBlockedShortcuts: AppCfg.Window.DefaultBlockedShortcuts,
		KeyMappings:             AppCfg.Window.KeyMappings,
	}

	dbg("STEP-4: ApplyPreShow (borderless=%v, opactiy=%.1f, transparent=%v, keeptop=%v)",
		winCfg.Borderless, winCfg.Opacity, winCfg.WebViewBgTransparent, winCfg.AlwaysOnTop)
	native.ApplyPreShow(w.Window(), winCfg)
	dbg("STEP-4: ok")

	dbg("STEP-5: SetSize %dx%d", AppCfg.Window.Width, AppCfg.Window.Height)
	w.SetSize(AppCfg.Window.Width, AppCfg.Window.Height, webview.HintNone)
	native.SetDefaultWindowSize(AppCfg.Window.Width, AppCfg.Window.Height)
	dbg("STEP-5: ok")

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

	dbg("STEP-6: Init (inject CSS + access token)")
	w.Init(buildInjectJS(injectCSS, DisableContextmenu, AppCfg.Window.DisableBrowserShortcuts, accessToken))
	dbg("STEP-6: ok")

	dbg("STEP-7: Navigate")
	navigateURL := addr
	if devURL != "" {
		navigateURL = devURL
		dbg("  dev mode, navigating to Vite: %s", devURL)
	} else {
		// 在 URL 中嵌入访问令牌，仅 WebView 知道完整 URL
		navigateURL = addr + "/?_wtd_=" + accessToken
		dbg("  navigating to local server: %s", addr)
	}
	w.Navigate(navigateURL)
	dbg("STEP-7: ok")

	dbg("STEP-8: Dispatch (Apply post-show config)")
	w.Dispatch(func() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					dbg("PANIC in Dispatch: %v\n%s", r, debug.Stack())
				}
			}()
			native.Apply(w.Window(), winCfg)
			if AppIconPath != "" {
				native.SetWindowIcon(w.Window(), AppIconPath)
			}
			platformOnShow(w.Window())
		}()
	})
	dbg("STEP-8: dispatched")

	// ★ 延迟安装键盘钩子，避开杀毒软件启动扫描（3~7 秒后）
	if AppCfg.Window.KeyboardShortcuts {
		go func() {
			delay := time.Duration(3+rand.Intn(5)) * time.Second
			dbg("keyboard hook will install after %v", delay)
			time.Sleep(delay)
			w.Dispatch(func() {
				native.EnableKeyboardHook(w.Window(), AppCfg.Window.DefaultBlockedShortcuts)
			})
		}()
	}

	// ★ 穿透：JS 每帧通过 bridge.getCursorPos 拉取鼠标坐标

	go func() {
		<-sigCh
		dbg("signal received, terminating...")
		w.Terminate()
	}()

	// ★ 键盘事件轮询：将拦截到的快捷键以 CustomEvent 推送到前端
	if AppCfg.Window.KeyboardShortcuts {
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				key := native.PollKbEvent()
				if key != "" {
					js := fmt.Sprintf("window.dispatchEvent(new CustomEvent('keyboard-shortcut',{detail:{key:%q}}))", key)
					w.Dispatch(func() { w.Eval(js) })
				}
			}
		}()
	}

	// ★ 外挂 HTML + 调试模式：文件变更监控 → 自动刷新页面
	if externalWebDir != "" && isConsole {
		go func() {
			pollInterval := 1 * time.Second
			lastMod := time.Now()
			for {
				time.Sleep(pollInterval)
				changed := false
				filepath.Walk(externalWebDir, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					if info.ModTime().After(lastMod) {
						changed = true
						return filepath.SkipDir
					}
					return nil
				})
				if changed {
					dbg("[WTD] web/ 文件变更，自动刷新页面")
					w.Dispatch(func() { w.Eval("location.reload()") })
					lastMod = time.Now()
				}
			}
		}()
	}

	dbg("STEP-9: w.Run (main loop)")
	w.Run()
	dbg("STEP-9: main loop exited")

	ShutdownServer(server, 3*time.Second)
	dbg("=== runApp end ===")
	if debugMode {
		fmt.Fprintf(os.Stderr, "\n[WTD] 调试日志: %s\n", logFile)
	}
}
