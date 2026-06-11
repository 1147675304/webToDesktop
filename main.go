// tools/desktop/main.go
// 桌面版 EXE 主入口
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/lhpanda/webtodesktop/pkg"
	"github.com/lhpanda/webtodesktop/pkg/bridge"
)

//go:embed all:vue/dist
var embeddedVue embed.FS

//go:embed config.yaml
var configData []byte

// ———— 构建时注入变量（通过 -ldflags -X 设置） ————

var BuildRemoteURL string
var BuildProxyPrefixes string
var BuildProjectName string
var BuildSignHeader string         // 桌面端签名请求头名称（默认 X-Desktop-Signature）
var BuildDisableContextmenu string // "true" 则禁用右键菜单
var BuildDevURL string              // 开发模式 Vite URL，非空时 webview 直接导航到此

func main() {
	// 解析配置
	if err := pkg.InitConfig(configData); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}

	// 嵌入资源
	staticFS, err := fs.Sub(embeddedVue, "vue/dist")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载嵌入资源失败: %v\n", err)
		os.Exit(1)
	}

	remoteURL := strings.TrimSpace(BuildRemoteURL)
	proxyPrefixes := parseProxyPrefixes(BuildProxyPrefixes)

	if remoteURL == "" {
		fmt.Fprintln(os.Stderr, "错误: 未指定远程服务器地址")
		os.Exit(1)
	}
	if len(proxyPrefixes) == 0 {
		proxyPrefixes = []string{"/api/", "/storage/"}
	}

	fmt.Printf("%s v%s 启动中...\n", pkg.AppCfg.App.Name, pkg.AppCfg.App.Version)
	fmt.Printf("远程服务器: %s\n", remoteURL)
	fmt.Printf("代理前缀: %v\n", proxyPrefixes)

	store, err := pkg.NewStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化本地存储失败: %v（凭证功能不可用）\n", err)
	}

	if store != nil {
		if saved, err := store.LoadWindowConfig(); err == nil && saved != nil {
			fmt.Println("加载已保存的窗口配置...")
			pkg.UpdateAppWindowConfig(saved)
		}
	}

	addr, server, err := pkg.StartServer(staticFS, remoteURL, proxyPrefixes, store, BuildSignHeader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "启动服务失败: %v\n", err)
		os.Exit(1)
	}

	// 创建桥接调度器
	var br pkg.BridgeBinder
	if store != nil {
		br = bridge.New(store)
	}

	pkg.DisableContextmenu = BuildDisableContextmenu == "true"

	pkg.RunApp(addr, server, store, BuildProjectName, br, BuildDevURL)
}

func parseProxyPrefixes(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if !strings.HasSuffix(p, "/") {
			p = p + "/"
		}
		result = append(result, p)
	}
	return result
}
