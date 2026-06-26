package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
var BuildDevURL string             // 开发模式 Vite URL，非空时 webview 直接导航到此
var BuildExternalHTML string       // "true" = 外挂 HTML 模式（从 web/ 目录读取）
var BuildConsole string            // "true" = 控制台（调试）版

func isConsoleBuild() bool {
	return BuildConsole == "true" || BuildDevURL != ""
}

func main() {
	// 解析配置
	if err := pkg.InitConfig(configData); err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(1)
	}

	// 静态资源源：外挂 HTML 模式从可执行文件同目录的 web/ 读取
	var staticFS fs.FS
	var externalWebDir string // 外挂 HTML 的 web/ 目录路径（用于文件监控）
	if BuildExternalHTML == "true" {
		exe, _ := os.Executable()
		externalWebDir = filepath.Join(filepath.Dir(exe), "web")
		if info, err := os.Stat(externalWebDir); err == nil && info.IsDir() {
			staticFS = os.DirFS(externalWebDir)
			fmt.Printf("[WTD] 外挂 HTML 模式: %s\n", externalWebDir)
		} else {
			fmt.Fprintf(os.Stderr, "[WTD] 外挂 HTML 模式启用但未找到 web/ 目录: %s\n", externalWebDir)
			fmt.Fprintf(os.Stderr, "[WTD] 尝试使用嵌入资源作为回退...\n")
			staticFS, _ = fs.Sub(embeddedVue, "vue/dist")
			externalWebDir = "" // 回退到嵌入模式，不监控
		}
	} else {
		var err error
		staticFS, err = fs.Sub(embeddedVue, "vue/dist")
		if err != nil {
			fmt.Fprintf(os.Stderr, "加载嵌入资源失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 外挂 HTML 模式：从 web/.env.production 读取运行时配置
	if BuildExternalHTML == "true" && externalWebDir != "" {
		envFile := filepath.Join(externalWebDir, ".env.production")
		if data, err := os.ReadFile(envFile); err == nil {
			envMap := parseEnvFile(string(data))
			if v := envMap["VITE_REMOTE_API_URL"]; v != "" {
				BuildRemoteURL = v
			}
			if v := envMap["VITE_PROXY_PREFIXES"]; v != "" {
				BuildProxyPrefixes = v
			}
			if v := envMap["VITE_DISABLE_CONTEXTMENU"]; v != "" {
				BuildDisableContextmenu = v
			}
			if v := envMap["VITE_DESKTOP_SIGN_HEADER"]; v != "" {
				BuildSignHeader = v
			}
			fmt.Printf("[WTD] 从 %s 加载运行时配置\n", envFile)
		} else {
			fmt.Printf("[WTD] 未找到 %s，使用构建时配置\n", envFile)
		}
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

	disableCtxMenu := BuildDisableContextmenu == "true"
	_ = disableCtxMenu // 后续用于控制右键菜单

	fmt.Printf("%s v%s 启动中...\n", pkg.AppCfg.App.Name, pkg.AppCfg.App.Version)
	fmt.Printf("远程服务器: %s\n", remoteURL)
	fmt.Printf("代理前缀: %v\n", proxyPrefixes)

	store, err := pkg.NewStore(BuildProjectName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化本地存储失败: %v（凭证功能不可用）\n", err)
	}

	if store != nil {
		if saved, err := store.LoadWindowConfig(); err == nil && saved != nil {
			fmt.Println("加载已保存的窗口配置...")
			pkg.UpdateAppWindowConfig(saved)
		}
	}

	addr, accessToken, server, err := pkg.StartServer(staticFS, remoteURL, proxyPrefixes, store, BuildSignHeader)
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

	pkg.RunApp(addr, accessToken, server, store, BuildProjectName, br, BuildDevURL, externalWebDir, isConsoleBuild())
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

// parseEnvFile 解析 .env 格式文件，返回 key→value 映射。
// 支持：VITE_KEY=value、# 注释、空行、引号去除。
func parseEnvFile(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		if key != "" {
			result[key] = val
		}
	}
	return result
}
