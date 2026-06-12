// 构建命令：build / build-vue / all
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func cmdBuildVue(project string) {
	vueDir, err := findProject(project)
	if err != nil {
		fatalf("错误: %v", err)
	}

	pm := detectPM(vueDir)
	infof(">>> 构建前端 (%s, %s)...", project, pm)
	if err := runCmdInDir(vueDir, pm, "run", "build"); err != nil {
		fatalf("Vue 构建失败")
	}

	infof(">>> 复制产物到 embed 目录...")
	os.RemoveAll(vueEmbed)
	os.MkdirAll(vueEmbed, 0755)

	distDir := filepath.Join(vueDir, "dist")
	entries, _ := os.ReadDir(distDir)
	for _, entry := range entries {
		copyPath(filepath.Join(distDir, entry.Name()), filepath.Join(vueEmbed, entry.Name()))
	}
	gzipDist()
	infof("构建完成 → vue/dist/")
}

// cmdBuild 构建单个平台
func cmdBuild(platform, project string) error {
	// 构建时传入项目名，用于输出文件名和持久化隔离
	// 1. 解析项目配置
	vueDir, err := findProject(project)
	if err != nil {
		return err
	}
	fmt.Println("项目目录:", vueDir)

	// 2. 读取环境变量
	envFile := filepath.Join(vueDir, ".env.production")
	remoteURL, proxyPrefixes, signHeader, err := readEnv(envFile)
	if err != nil {
		return err
	}
	// 读取 disableContextmenu（从 .env.production，默认不禁用）
	envBytes, _ := os.ReadFile(envFile)
	disableCtxMenu := extractEnvValue(string(envBytes), "VITE_DISABLE_CONTEXTMENU")
	if disableCtxMenu == "" {
		// console 模式自动不禁用，桌面模式默认禁用
		if strings.Contains(platform, "console") {
			disableCtxMenu = "false"
		} else {
			disableCtxMenu = "true"
		}
	}
	fmt.Println("远程服务器:", remoteURL)
	fmt.Println("代理前缀:  ", proxyPrefixes)
	fmt.Println("签名请求头:", signHeader)
	fmt.Println("禁用右键:  ", disableCtxMenu)

	// 读取 build tags（可选模块裁剪）
	buildTags := extractEnvValue(string(envBytes), "BUILD_TAGS")
	if buildTags != "" {
		fmt.Println("构建标签:  ", buildTags)
	}

	// 查询是否启用外挂 HTML
	externalHTML := isExternalHTML(project)

	// 3. 复制前端产物
	if err := copyDist(vueDir, project); err != nil {
		return err
	}

	// 4. 桌面图标
	envData, _ := os.ReadFile(envFile)
	envStr := string(envData)
	copyDesktopIcons(envStr)

	// 5. 生成 EXE 图标（外挂和嵌入模式都需要）
	winIconPath := extractEnvValue(envStr, "DESKTOP_ICON_WINDOWS")
	if winIconPath == "" {
		winIconPath = filepath.Join(vueEmbed, "favicon.ico")
	} else {
		winIconPath = filepath.Join(vueEmbed, winIconPath)
	}
	if !fileExists(winIconPath) {
		pngDefault := filepath.Join(projectRoot, "pkg", "icon.png")
		if fileExists(pngDefault) {
			winIconPath = pngDefault
		}
	}
	generateIcon(winIconPath)
	if platform != "windows" && platform != "windows-console" {
		os.Remove(filepath.Join(projectRoot, "rsrc.syso"))
	}
	fmt.Println()

	if externalHTML {
		// 外挂 HTML 模式：复制到构建输出目录的 web/ 子目录，跳过 gzip
		webDir := filepath.Join(outputDir, "web")
		os.RemoveAll(webDir)
		os.MkdirAll(webDir, 0755)
		entries, _ := os.ReadDir(vueEmbed)
		for _, entry := range entries {
			copyPath(filepath.Join(vueEmbed, entry.Name()), filepath.Join(webDir, entry.Name()))
		}
		// vue/dist/ 留一个占位文件使 //go:embed 不报错
		os.MkdirAll(vueEmbed, 0755)
		os.WriteFile(filepath.Join(vueEmbed, ".gitkeep"), []byte{}, 0644)
		fmt.Printf(">>> 外挂 HTML → %s/\n", webDir)
		fmt.Println(">>> 外挂模式：跳过 gzip 压缩")
	} else {
		// 6. Gzip 压缩（仅嵌入模式）
		gzipDist()
	}

	// 7. 构建 Go 二进制
	infof(">>> 构建 Go 二进制 (%s)...", platform)

	isConsole := strings.Contains(platform, "console")
	ldflags := fmt.Sprintf("-s -w -X 'main.BuildRemoteURL=%s' -X 'main.BuildProxyPrefixes=%s' -X 'main.BuildProjectName=%s' -X 'main.BuildSignHeader=%s' -X 'main.BuildDisableContextmenu=%s' -X 'main.BuildExternalHTML=%s' -X 'main.BuildConsole=%s'",
		remoteURL, proxyPrefixes, project, signHeader, disableCtxMenu, fmt.Sprint(externalHTML), fmt.Sprint(isConsole))

	if err := goBuild(platform, ldflags, buildTags, project); err != nil {
		return err
	}

	os.Remove(filepath.Join(projectRoot, "rsrc.syso"))

	fmt.Println()
	infof("============================================")
	infof("  构建完成!")
	infof("============================================")
	printBuildOutputs()
	return nil
}

// cmdAll 打包所有平台
func cmdAll(project string) {
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)

	fmt.Println(cGreen + "╔══════════════════════════════════════════════╗" + cReset)
	fmt.Println(cGreen + "║  打包所有平台: " + project + cReset)
	fmt.Println(cGreen + "╚══════════════════════════════════════════════╝" + cReset)
	fmt.Println()

	platforms := []string{"linux-amd64", "linux-amd64-console", "linux-arm64", "linux-loong64", "windows", "windows-console"}
	var failed []string

	for _, plat := range platforms {
		fmt.Println()
		cyanf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyanf("  开始构建: %s", plat)
		cyanf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		if err := cmdBuild(plat, project); err != nil {
			fmt.Println(cRed + "✗ " + plat + " 构建失败: " + err.Error() + cReset)
			failed = append(failed, plat)
		} else {
			infof("✓ %s 构建成功", plat)
		}
	}

	fmt.Println()
	infof("============================================")
	infof("  全部打包完成!")
	infof("============================================")
	printBuildOutputs()

	successCount := len(platforms) - len(failed)
	if len(failed) > 0 {
		fmt.Println()
		warnf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("  成功: "+cGreen+"%d"+cReset+" / 失败: "+cRed+"%d"+cReset+"\n", successCount, len(failed))
		warnf("  跳过平台: %v", failed)
		warnf("  原因: CGO + GTK/WebKit 无法跨架构交叉编译")
		warnf("  请在对应架构的机器上本地编译这些平台")
		warnf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if successCount == 0 {
			os.Exit(1)
		}
	}
}

// copyDesktopIcons 从 env 配置复制桌面图标。
// 找不到配置的图标时，静默使用 pkg/icon.png 默认图标。
func copyDesktopIcons(envStr string) {
	iconLinux := extractEnvValue(envStr, "DESKTOP_ICON_LINUX")
	iconWin := extractEnvValue(envStr, "DESKTOP_ICON_WINDOWS")

	if iconLinux != "" {
		srcIcon := filepath.Join(vueEmbed, iconLinux)
		if fileExists(srcIcon) {
			copyFile(srcIcon, filepath.Join(projectRoot, "pkg", "icon.png"))
			fmt.Println("桌面图标(Linux): ", iconLinux, "→ pkg/icon.png")
		}
		// 找不到则保留默认 pkg/icon.png
	}

	if iconWin != "" {
		srcIcon := filepath.Join(vueEmbed, iconWin)
		if fileExists(srcIcon) {
			fmt.Println("桌面图标(Win): ", iconWin)
		}
	}
	fmt.Println()
}

// goBuild 执行 go build
func goBuild(platform, baseLdflags, buildTags, project string) error {
	cfg := buildCfg(platform, baseLdflags, buildTags, project)
	fmt.Println(cfg.desc)

	if cfg.preSetup != nil {
		if err := cfg.preSetup(); err != nil {
			return err
		}
	}

	args := []string{"build"}
	if cfg.ldflags != "" {
		args = append(args, "-ldflags="+cfg.ldflags)
	}
	if cfg.buildTags != "" {
		args = append(args, "-tags="+cfg.buildTags)
	}
	if fileExists("vendor") {
		args = append(args, "-mod=vendor")
	}
	args = append(args, "-o", cfg.output, ".")

	cmd := exec.Command("go", args...)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), cfg.env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type buildConfig struct {
	desc      string
	output    string
	ldflags   string
	buildTags string
	env       []string
	preSetup  func() error
}

func buildCfg(platform, baseLdflags, buildTags, project string) buildConfig {
	name := project
	if name == "" {
		name = appName
	}
	switch platform {
	case "linux":
		arch := getTargetArch()
		return buildConfig{
			desc:      fmt.Sprintf("  目标: Linux %s (WebKitGTK)", arch),
			output:    filepath.Join(outputDir, name+"-linux-"+arch),
			ldflags:   baseLdflags,
			buildTags: buildTags,
			env:       addPkgConfigPath("CGO_ENABLED=1", "GOARCH="+arch),
			preSetup:  func() error { return setupLinuxCrossCC(arch) },
		}
	case "linux-amd64":
		return buildConfig{
			desc:      "  目标: Linux amd64 (WebKitGTK)",
			output:    filepath.Join(outputDir, name+"-linux-amd64"),
			ldflags:   baseLdflags,
			buildTags: buildTags,
			env:       addPkgConfigPath("CGO_ENABLED=1", "GOARCH=amd64"),
			preSetup:  func() error { return setupLinuxCrossCC("amd64") },
		}
	case "linux-amd64-console":
		return buildConfig{
			desc:      "  目标: Linux amd64 + 控制台 (调试用)",
			output:    filepath.Join(outputDir, name+"-linux-amd64-console"),
			ldflags:   baseLdflags + " -X 'github.com/lhpanda/webtodesktop/pkg.BuildDebug=true'",
			buildTags: buildTags,
			env:       addPkgConfigPath("CGO_ENABLED=1", "GOARCH=amd64"),
			preSetup:  func() error { return setupLinuxCrossCC("amd64") },
		}
	case "linux-arm64":
		return buildConfig{
			desc:      "  目标: Linux arm64 (WebKitGTK)",
			output:    filepath.Join(outputDir, name+"-linux-arm64"),
			ldflags:   baseLdflags,
			buildTags: buildTags,
			env:       addPkgConfigPath("CGO_ENABLED=1", "GOARCH=arm64"),
			preSetup:  func() error { return setupLinuxCrossCC("arm64") },
		}
	case "linux-loong64":
		return buildConfig{
			desc:      "  目标: Linux loong64 / 龙芯 (WebKitGTK)",
			output:    filepath.Join(outputDir, name+"-linux-loong64"),
			ldflags:   baseLdflags,
			buildTags: buildTags,
			env:       addPkgConfigPath("CGO_ENABLED=1", "GOARCH=loong64"),
			preSetup:  func() error { return setupLinuxCrossCC("loong64") },
		}
	case "windows":
		return buildConfig{
			desc:      "  目标: Windows (Edge WebView2)\n  要求: sudo apt install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64",
			output:    filepath.Join(outputDir, name+".exe"),
			ldflags:   "-H windowsgui " + baseLdflags,
			buildTags: buildTags,
			env: []string{
				"CGO_ENABLED=1",
				"CGO_CXXFLAGS=-I" + filepath.Join(projectRoot, "include_win"),
				"CC=" + envOr("MINGW_CC", "x86_64-w64-mingw32-gcc"),
				"CXX=" + envOr("MINGW_CXX", "x86_64-w64-mingw32-g++"),
				"GOOS=windows", "GOARCH=amd64",
			},
		}
	case "windows-console":
		return buildConfig{
			desc:      "  目标: Windows + 控制台 (调试用)",
			output:    filepath.Join(outputDir, name+"-console.exe"),
			ldflags:   baseLdflags + " -X 'github.com/lhpanda/webtodesktop/pkg.BuildDebug=true'",
			buildTags: buildTags,
			env: []string{
				"CGO_ENABLED=1",
				"CGO_CXXFLAGS=-I" + filepath.Join(projectRoot, "include_win"),
				"CC=" + envOr("MINGW_CC", "x86_64-w64-mingw32-gcc"),
				"CXX=" + envOr("MINGW_CXX", "x86_64-w64-mingw32-g++"),
				"GOOS=windows", "GOARCH=amd64",
			},
		}
	default: // current / current-console
		cfg := buildConfig{
			desc:      "  目标: 当前平台",
			output:    filepath.Join(outputDir, name),
			ldflags:   baseLdflags,
			buildTags: buildTags,
			env:       addPkgConfigPath("CGO_ENABLED=1"),
		}
		if platform == "current-console" {
			cfg.desc = "  目标: 当前平台 + 控制台 (调试用)"
			cfg.ldflags += " -X 'github.com/lhpanda/webtodesktop/pkg.BuildDebug=true'"
		}
		return cfg
	}
}

// pkgConfigExists 检查指定的 pkg-config 包是否存在。
func pkgConfigExists(name string) bool {
	cmd := exec.Command("pkg-config", "--exists", name)
	return cmd.Run() == nil
}

// pkgConfigLibs 获取指定 pkg-config 包的链接参数。
func pkgConfigLibs(name string) string {
	out, err := exec.Command("pkg-config", "--libs", name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// pkgConfigCflags 获取指定 pkg-config 包的编译参数。
func pkgConfigCflags(name string) string {
	out, err := exec.Command("pkg-config", "--cflags", name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ensurePkgConfigCompat 自动检测并生成兼容性 .pc 文件。
// 当请求的包名（如 webkit2gtk-4.0）不存在时，自动查找同一家族的高版本
// （如 webkit2gtk-4.1）并生成兼容 shim。
// 无需手动维护版本映射表，系统升级后自动适配。
func ensurePkgConfigCompat(pkgConfigDir string) {
	// 获取系统上已安装的所有 pkg-config 包
	out, err := exec.Command("pkg-config", "--list-all").Output()
	if err != nil {
		return
	}
	installed := strings.Split(strings.TrimSpace(string(out)), "\n")

	// 检查当前项目的所有 CGO 依赖
	deps, err := scanCGODeps()
	if err != nil {
		deps = []string{"webkit2gtk-4.0", "javascriptcoregtk-4.0"} // 回退
	}

	for _, req := range deps {
		if pkgConfigExists(req) {
			continue // 已安装，跳过
		}

		// 提取包名基础部分：webkit2gtk-4.0 → webkit2gtk, 4, 0
		base, major, minor := parsePkgName(req)
		if base == "" || major < 0 {
			continue
		}

		// 在所有已安装包中查找同家族的最高版本
		alt := findBestCompat(base, major, minor, installed)
		if alt == "" {
			continue
		}

		// 生成兼容 .pc 文件
		libs := pkgConfigLibs(alt)
		cflags := pkgConfigCflags(alt)
		pcContent := fmt.Sprintf(`prefix=/usr
exec_prefix=${prefix}
libdir=/usr/lib/x86_64-linux-gnu
includedir=${prefix}/include

Name: %s (compat)
Description: Auto-generated compatibility shim for %s → %s
Version: 0.0.0
Libs: %s
Cflags: %s
`, req, req, alt, libs, cflags)

		pcPath := filepath.Join(pkgConfigDir, req+".pc")
		if err := os.WriteFile(pcPath, []byte(pcContent), 0644); err != nil {
			warnf("  无法生成 %s 兼容文件: %v", pcPath, err)
		} else {
			infof("  自动生成 %s → %s 兼容配置", req, alt)
		}
	}
}

// parsePkgName 解析 pkg-config 包名，提取基础名称和主次版本号。
// 如 "webkit2gtk-4.0" → base="webkit2gtk", major=4, minor=0
func parsePkgName(name string) (base string, major int, minor int) {
	// 从末尾反向查找版本分隔符
	lastDash := strings.LastIndex(name, "-")
	if lastDash < 0 {
		return name, -1, -1
	}
	verStr := name[lastDash+1:]
	parts := strings.SplitN(verStr, ".", 2)
	if len(parts) != 2 {
		return "", -1, -1
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return "", -1, -1
	}
	return name[:lastDash], major, minor
}

// findBestCompat 在已安装包列表中查找同家族的最高兼容版本。
func findBestCompat(base string, reqMajor, reqMinor int, installed []string) string {
	var best string
	bestMajor, bestMinor := reqMajor, reqMinor

	for _, pkg := range strings.Fields(strings.Join(installed, "\n")) {
		// pkg-config --list-all 输出格式: "name description"
		name := strings.SplitN(pkg, " ", 2)[0]
		b, m, n := parsePkgName(name)
		if b != base || m < 0 {
			continue
		}
		// 只选比请求版本更高或相等的
		if m > bestMajor || (m == bestMajor && n > bestMinor) ||
			(m == reqMajor && n >= reqMinor) {
			best = name
			bestMajor = m
			bestMinor = n
		}
	}
	return best
}

// scanCGODeps 扫描项目 Go 文件中的 #cgo pkg-config 依赖。
func scanCGODeps() ([]string, error) {
	var deps []string
	seen := map[string]bool{}

	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// 匹配 #cgo pkg-config: name1 name2 ...
		re := regexp.MustCompile(`#cgo\s+.*?\bpkg-config:\s*(.+)`)
		for _, match := range re.FindAllStringSubmatch(string(data), -1) {
			for _, name := range strings.Fields(match[1]) {
				if !seen[name] {
					seen[name] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for name := range seen {
		deps = append(deps, name)
	}
	return deps, nil
}

// addPkgConfigPath 将项目 pkgconfig 目录添加到 PKG_CONFIG_PATH 环境变量中，
// 并自动检测 webkit2gtk 版本兼容性。
func addPkgConfigPath(envs ...string) []string {
	pkgConfigPath := filepath.Join(projectRoot, "pkgconfig")
	os.MkdirAll(pkgConfigPath, 0755)
	ensurePkgConfigCompat(pkgConfigPath)

	env := append([]string{}, envs...)
	existing := os.Getenv("PKG_CONFIG_PATH")
	if existing != "" {
		env = append(env, "PKG_CONFIG_PATH="+pkgConfigPath+":"+existing)
	} else {
		env = append(env, "PKG_CONFIG_PATH="+pkgConfigPath)
	}
	return env
}
