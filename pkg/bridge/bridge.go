// Package bridge 实现 Go ↔ JS 统一桥接调度器。
//
// 通过 webview.Bind 将 Bridge.Call 暴露为 JS 全局函数 window.__lhpanda__(method, params)，
// 前端调用时自动路由到对应的 Go handler，返回 Promise<result>。
//
// 架构：
//
//	JS:  window.__lhpanda__('saveCredentials', {username, password})
//	       → webview JSON 序列化
//	Go:  Bridge.Call(method, params)
//	       → 查表 methods[method] → handler(params)
//	       → 返回 {success: true, data: {...}}
//	JS:  .then(result => result.data)
//
// 按功能模块拆分文件：
//   - bridge.go          核心调度器（Bridge 结构体、自动注册机制、Call）
//   - credentials.go     凭证管理（save/get/delete/clearCredentials）
//   - window.go          窗口控制（drag/resize/close/toggle*）
//   - config.go          窗口配置持久化（get/saveWindowConfig + 辅助函数）
//
// ★ 新增方法零配置：只需写 func (b *Bridge) handleXxx(params) (...) 即可，
//
//	registerBuiltins() 通过反射自动发现所有 handle* 方法并注册。
//	方法名 → JS 方法名规则: handleGetAppInfo → "getAppInfo"（去掉 handle，首字母小写）
package bridge

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"

	"github.com/lhpanda/webtodesktop/pkg"
	webview "github.com/webview/webview_go"
)

// HandlerFunc 是注册到 Bridge 的方法签名。
//
// 参数 params 由 JS 传入的对象经 JSON 反序列化而来，类型为 map[string]interface{}。
// JS 的 number → Go float64，JS 的 boolean → Go bool，JS 的 string → Go string。
//
// 返回值 (data, error)：
//   - data 非 nil 且 error 为 nil：Call 包装为 {success: true, data: data}
//   - error 非 nil：Call 透传错误给 JS 的 .catch()
type HandlerFunc func(params map[string]interface{}) (interface{}, error)

// Bridge 是 Go↔JS 桥接的核心调度器。
//
// 每个桌面应用创建一个 Bridge 实例，通过 Register() 注册 handler，
// 通过 webview.Bind("__lhpanda__", bridge.Call) 暴露给前端。
//
// 字段说明：
//   - store   用于凭证管理和窗口配置持久化，可能为 nil（降级运行）
//   - methods handler 注册表，key 为 JS 调用的方法名字符串
//   - wv      关联的原生 WebView 实例，用于 Dispatch() 到 UI 线程执行窗口操作
//   - winPtr  缓存的原生窗口句柄（HWND/GtkWindow），避免每次从 wv 获取
type Bridge struct {
	store                *pkg.Store
	methods              map[string]HandlerFunc
	wv                   webview.WebView
	winPtr               unsafe.Pointer
	lastPassthroughState bool
	lastPassthroughValid bool // 首次调用强制生效
}

// New 创建 Bridge 实例并自动注册所有 handle* 方法。
//
// 通过反射扫描 *Bridge 的所有方法，将符合 handleXxx 命名规范的方法
// 自动注册为 JS 方法 "xxx"（首字母小写）。
//
// 新增方法只需在任意文件中添加:
//
//	func (b *Bridge) handleMyMethod(params map[string]interface{}) (interface{}, error) { ... }
//
// 无需任何额外注册代码。
//
// store 为 nil 时，凭证和配置相关功能降级（调用时返回错误），窗口控制仍可用。
func New(store *pkg.Store) *Bridge {
	b := &Bridge{
		store:   store,
		methods: make(map[string]HandlerFunc),
	}
	b.registerBuiltins()
	return b
}

// SetWebView 关联原生 WebView 实例，启用窗口控制功能。
//
// 必须在 WebView 创建后、绑定到 JS 前调用。
// 调用后 window control 类 handler（drag/resize/close 等）才能通过 Dispatch() 操作窗口。
func (b *Bridge) SetWebView(wv webview.WebView) {
	b.wv = wv
	b.winPtr = wv.Window()
}

// Register 注册一个具名 handler。
//
// 外部可调用此方法扩展桥接功能：
//
//	bridge.Register("myCustomMethod", func(params map[string]interface{}) (interface{}, error) {
//	    return map[string]interface{}{"ok": true}, nil
//	})
func (b *Bridge) Register(name string, handler HandlerFunc) {
	b.methods[name] = handler
}

// Call 是暴露给 JS 的统一调度入口。
//
// 通过 webview.Bind("__lhpanda__", bridge.Call) 绑定后，
// JS 侧调用 window.__lhpanda__('methodName', {key: 'value'}) 即触发此方法。
//
// 执行流程：
//  1. 查表 methods[method] → 未找到返回 error
//  2. 调用 handler(params) → 获取 (result, error)
//  3. 包装为 {success: true, data: result} 或透传 error
//
// 注意：webview.Bind 要求返回 (map[string]interface{}, error) 签名，
// webview 库会自动将返回值 JSON 序列化后 resolve/reject JS Promise。
func (b *Bridge) Call(method string, params map[string]interface{}) (map[string]interface{}, error) {
	handler, ok := b.methods[method]
	if !ok {
		return nil, fmt.Errorf("未知方法: %s", method)
	}
	result, err := handler(params)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": true,
		"data":    result,
	}, nil
}

// registerBuiltins 通过反射自动发现并注册所有 handle* 方法。
//
// 命名规则: Go 方法名 handleXxxYyy → JS 方法名 "xxxYyy"（去掉 handle 前缀，首字母小写）。
// 例如: handleGetAppInfo → "getAppInfo", handleToggleMaximize → "toggleMaximize"。
//
// 只注册签名匹配的方法: func(*Bridge, map[string]interface{}) (interface{}, error)。
// 不符合签名的方法（如 handleGetAppInfo 的旧签名）会被静默跳过。
//
// ★ 新增方法时无需修改此函数或添加 init()，写 func (b *Bridge) handleXxx(...) 即可。
func (b *Bridge) registerBuiltins() {
	t := reflect.TypeOf(b)
	v := reflect.ValueOf(b)

	fmt.Printf("[bridge] scanning %d exported methods on %v:\n", t.NumMethod(), t)

	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		name := method.Name

		// 只处理 Handle 前缀的方法
		if !strings.HasPrefix(name, "Handle") || len(name) <= 6 {
			fmt.Printf("[bridge]   skip %s (not Handle*)\n", name)
			continue
		}

		// 验证签名: func(*Bridge, map[string]interface{}) (interface{}, error)
		mt := method.Type
		if mt.NumIn() != 2 || mt.In(0) != t || mt.In(1) != typeMapIface {
			fmt.Printf("[bridge]   skip %s (bad in: NumIn=%d In0=%v In1=%v)\n",
				name, mt.NumIn(), mt.In(0), mt.In(1))
			continue
		}
		if mt.NumOut() != 2 || mt.Out(0) != typeInterface || mt.Out(1) != typeError {
			fmt.Printf("[bridge]   skip %s (bad out: NumOut=%d Out0=%v Out1=%v)\n",
				name, mt.NumOut(), mt.Out(0), mt.Out(1))
			continue
		}

		// handleGetAppInfo → "getAppInfo"
		jsName := strings.ToLower(name[6:7]) + name[7:]

		// 通过闭包捕获方法索引，避免每次调用都做反射查找
		idx := i
		b.methods[jsName] = func(params map[string]interface{}) (interface{}, error) {
			in := []reflect.Value{reflect.ValueOf(params)}
			out := v.Method(idx).Call(in)
			var err error
			if !out[1].IsNil() {
				err = out[1].Interface().(error)
			}
			return out[0].Interface(), err
		}

		fmt.Printf("[bridge] auto-registered: %s → handle%s\n", jsName, name[6:])
	}
}

// 反射中常用的类型，避免重复调用 reflect.TypeOf
var (
	typeMapIface  = reflect.TypeOf((map[string]interface{})(nil))
	typeInterface = reflect.TypeOf((*interface{})(nil)).Elem()
	typeError     = reflect.TypeOf((*error)(nil)).Elem()
)

// HandleGetAppInfo 返回当前应用的运行环境信息。
//
// JS 调用: window.__lhpanda__('getAppInfo')
// 返回: {success: true, data: {platform: "linux", arch: "amd64", version: "1.0.0"}}
func (b *Bridge) HandleGetAppInfo(params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"platform": runtime.GOOS,
		"arch":     runtime.GOARCH,
		"version":  "1.0.0",
	}, nil
}

// HandleListMethods 返回所有已注册的 JS 方法名列表。
//
// JS 调用: window.__lhpanda__('listMethods')
// 返回: {methods: ["getAppInfo", "saveCredentials", "dragWindow", ...]}
//
// 用途: 前端调试时发现可用方法，或动态生成 UI 控制面板。
func (b *Bridge) HandleListMethods(params map[string]interface{}) (interface{}, error) {
	names := make([]string, 0, len(b.methods))
	for name := range b.methods {
		names = append(names, name)
	}
	return map[string]interface{}{"methods": names}, nil
}
