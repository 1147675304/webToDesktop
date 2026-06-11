// 交互式菜单与运行/调试命令
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ———— 交互式选择 ————

func selectProject() string {
	cmdList()
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请选择项目编号 (输入序号或名称): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "" {
		fatalf("未选择项目，已取消")
	}

	cfg, err := loadConfig()
	if err != nil {
		fatalf("错误: %v", err)
	}

	if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(cfg.Projects) {
		return cfg.Projects[n-1].Name
	}

	for _, p := range cfg.Projects {
		if p.Name == choice {
			return p.Name
		}
	}

	fatalf("错误: 无效的选择 '%s'", choice)
	return ""
}

func selectPlatform() string {
	arch := detectArch()
	fmt.Println()
	fmt.Println("请选择目标平台:")
	fmt.Println("  [1] 全部平台 (Linux amd64 + arm64 + loong64 + Windows)")
	fmt.Println("  [2] Windows")
	fmt.Println("  [3] Windows + 控制台 (调试)")
	fmt.Println("  [4] Linux amd64")
	fmt.Println("  [5] Linux amd64 + 控制台 (调试)")
	fmt.Println("  [6] Linux arm64")
	fmt.Println("  [7] Linux loong64")
	fmt.Println("  [8] 当前平台 (" + arch + ")")
	fmt.Println("  [9] 当前平台 + 控制台 (" + arch + ", 调试)")
	fmt.Println("  当前设备架构: " + cCyan + arch + cReset)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请选择平台编号 [默认: 2]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return "all"
	case "3":
		return "windows-console"
	case "4":
		return "linux-amd64"
	case "5":
		return "linux-amd64-console"
	case "6":
		return "linux-arm64"
	case "7":
		return "linux-loong64"
	case "8":
		return "current"
	case "9":
		return "current-console"
	default:
		return "windows"
	}
}

func cmdInteractive() {
	project := selectProject()
	target := selectPlatform()

	infof("已选择项目: %s  目标平台: %s", project, target)
	fmt.Println()

	if target == "all" {
		cmdAll(project)
	} else {
		if err := cmdBuild(target, project); err != nil {
			fatalf("构建失败: %v", err)
		}
	}
}

// ———— 运行 & 调试 ————

func cmdRun(project string) {
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)
	if err := cmdBuild("current", project); err != nil {
		fatalf("构建失败: %v", err)
	}

	if os.Getenv("WTD_DEBUG") != "0" {
		os.Setenv("WTD_DEBUG", "1")
	}

	fmt.Println()
	infof(">>> 启动应用...")
	cmd := exec.Command(filepath.Join(outputDir, appName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fatalf("运行失败: %v", err)
	}
}

func cmdRunWindows(project string) {
	if err := cmdBuild("windows-console", project); err != nil {
		fatalf("构建失败: %v", err)
	}

	if os.Getenv("WTD_DEBUG") != "0" {
		os.Setenv("WTD_DEBUG", "1")
	}

	if commandExists("wine") {
		fmt.Println()
		infof(">>> 通过 Wine 启动 (调试模式)...")
		cmd := exec.Command("wine", filepath.Join(outputDir, appName+"-console.exe"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fatalf("Wine 运行失败: %v", err)
		}
	} else {
		fmt.Println()
		warnf("Wine 未安装，请手动将 EXE 复制到 Windows 运行:")
		fmt.Println("  " + filepath.Join(outputDir, appName+"-console.exe"))
	}
}

func cmdDev(project string) {
	vueDir, err := findProject(project)
	if err != nil {
		fatalf("错误: %v", err)
	}

	envFile := filepath.Join(vueDir, ".env.production")
	if !fileExists(envFile) {
		// 纯前端项目（无 package.json）自动创建最小配置，无需手动编辑
		if !fileExists(filepath.Join(vueDir, "package.json")) {
			os.WriteFile(envFile, []byte("VITE_REMOTE_API_URL=about:blank\nVITE_PROXY_PREFIXES=\n"), 0644)
		} else {
			warnf("未找到 %s，创建默认配置...", envFile)
			os.WriteFile(envFile, []byte("\nVITE_REMOTE_API_URL=https://your-api-server.com\nVITE_PROXY_PREFIXES=/api/,/storage/\n"), 0644)
			fatalf("请修改 %s 中的 VITE_REMOTE_API_URL 后重新运行", envFile)
		}
	}

	remoteURL, proxyPrefixes, _, err := readEnv(envFile)
	if err != nil {
		fatalf("错误: %v", err)
	}

	// 检测项目类型与包管理器
	pm := detectPM(vueDir)
	pkgJSON := filepath.Join(vueDir, "package.json")
	hasDevServer := fileExists(pkgJSON) && hasScript(vueDir, "dev")

	if !hasDevServer {
		// 无构建工具的项目（纯 HTML 等）：直接构建并运行，无 HMR
		infof(">>> 未检测到 dev server，直接构建运行...")
		if err := copyDist(vueDir, project); err != nil {
			fatalf("复制前端产物失败: %v", err)
		}
		gzipDist()
		os.RemoveAll(outputDir)
		os.MkdirAll(outputDir, 0755)
		ldflags := fmt.Sprintf("-s -w -X 'main.BuildRemoteURL=%s' -X 'main.BuildProxyPrefixes=%s' -X 'main.BuildProjectName=%s' -X 'main.BuildDisableContextmenu=false'",
			remoteURL, proxyPrefixes, project)
		if err := goBuild("current", ldflags, ""); err != nil {
			fatalf("构建失败: %v", err)
		}
		fmt.Println()
		infof(">>> 启动桌面窗口...")
		cmd := exec.Command(filepath.Join(outputDir, appName))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fatalf("运行失败: %v", err)
		}
		return
	}

	// 清理上次可能残留的 dev server 进程（按端口，不限工具）
	killPortRange(5173, 5180)

	// 1. 先构建一次前端
	infof(">>> 首次构建前端 (%s)...", project)
	buildCmd := exec.Command(pm, "run", "build")
	buildCmd.Dir = vueDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fatalf("前端构建失败: %v", err)
	}

	// 2. 后台启动 dev server（HMR），捕获 stdout 以检测实际端口
	infof(">>> 启动 dev server（热更新）...")
	viteCmd := exec.Command(pm, "run", "dev")
	viteCmd.Dir = vueDir
	viteCmd.Stderr = os.Stderr
	// Setpgid 使子进程归入新进程组，cleanup 时 Kill(-pid) 可杀掉整组
	viteCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// 用管道捕获 stdout 以解析实际端口
	stdoutPipe, err := viteCmd.StdoutPipe()
	if err != nil {
		fatalf("Vite stdout 管道创建失败: %v", err)
	}
	if err := viteCmd.Start(); err != nil {
		fatalf("Vite 启动失败: %v", err)
	}

	// 确保退出时清理 dev server 进程树（defer + signal 双保险）
	cleanup := func() {
		if viteCmd.Process != nil {
			pid := viteCmd.Process.Pid
			// 方式1: 杀掉进程组
			syscall.Kill(-pid, syscall.SIGKILL)
			// 方式2: 杀 pnpm 父进程
			viteCmd.Process.Kill()
			viteCmd.Process.Wait()
			// 方式3: 兜底杀端口上的残留进程
			exec.Command("pkill", "-f", "vite").Run()
		}
	}
	defer cleanup()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() { <-sigCh; cleanup(); os.Exit(0) }()

	// 启动后台 goroutine 持续输出日志 + 检测就绪端口
	devReady := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		// 匹配多种 dev server 的就绪输出格式
		patterns := []*regexp.Regexp{
			regexp.MustCompile(`Local:\s+(https?://[^\s]+)`),       // Vite
			regexp.MustCompile(`(https?://localhost:\d+)`),         // 通用 localhost URL
			regexp.MustCompile(`listening on\s+(https?://[^\s]+)`), // Node/Express
		}
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			for _, re := range patterns {
				if m := re.FindStringSubmatch(line); m != nil {
					select {
					case devReady <- m[1]:
					default:
					}
					return
				}
			}
		}
	}()

	// 等待 dev server 就绪：先等 stdout 输出 URL，超时则回退到端口探测
	var viteURL string
	select {
	case url := <-devReady:
		viteURL = url
	case <-time.After(8 * time.Second):
		// 8 秒无输出匹配，回退到端口探测
		viteURL = detectDevPort(5173, 5180)
		if viteURL == "" {
			cleanup()
			fatalf("Dev server 启动超时，请检查 pnpm dev 是否正常")
		}
	}
	infof(">>> Dev server 就绪: %s", viteURL)

	// 3. 构建 Go 桌面应用，远程 URL 指向 dev server
	infof(">>> 构建桌面应用...")
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)

	ldflags := fmt.Sprintf("-s -w -X 'main.BuildRemoteURL=%s' -X 'main.BuildProxyPrefixes=%s' -X 'main.BuildProjectName=%s' -X 'main.BuildDisableContextmenu=false' -X 'main.BuildDevURL=%s'",
		remoteURL, proxyPrefixes, project, viteURL)
	if err := goBuild("current", ldflags, ""); err != nil {
		cleanup()
		fatalf("构建失败: %v", err)
	}

	// 4. 启动桌面窗口
	if os.Getenv("WTD_DEBUG") != "0" {
		os.Setenv("WTD_DEBUG", "1")
	}

	fmt.Println()
	infof(">>> 启动桌面窗口（HMR 已启用，修改前端代码自动刷新）...")
	cmd := exec.Command(filepath.Join(outputDir, appName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		cleanup()
		fatalf("运行失败: %v", err)
	}
}

// detectDevPort TCP 端口探测回退方案：从 start 到 end 扫描 localhost 端口，
// 返回第一个有 HTTP 服务监听的 URL，若全无则返回空字符串。
func detectDevPort(start, end int) string {
	for port := start; port <= end; port++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 50*time.Millisecond)
		if err == nil {
			conn.Close()
			return fmt.Sprintf("http://localhost:%d", port)
		}
	}
	return ""
}

// detectPM 根据锁文件检测包管理器（pnpm / yarn / bun / npm）
func detectPM(dir string) string {
	if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) { return "pnpm" }
	if fileExists(filepath.Join(dir, "yarn.lock")) { return "yarn" }
	if fileExists(filepath.Join(dir, "bun.lockb")) || fileExists(filepath.Join(dir, "bun.lock")) { return "bun" }
	return "npm"
}

// hasScript 读取 package.json 检查是否定义了指定脚本（纯 JSON 解析，不执行命令）。
func hasScript(dir, script string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), fmt.Sprintf(`"%s":`, script))
}

// killPortRange 按端口范围 kill 监听进程（不限工具）。
func killPortRange(start, end int) {
	for port := start; port <= end; port++ {
		out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
		if err == nil && len(out) > 0 {
			for _, s := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if pid, err := strconv.Atoi(s); err == nil && pid > 1 {
					syscall.Kill(pid, syscall.SIGKILL)
				}
			}
		}
	}
}
