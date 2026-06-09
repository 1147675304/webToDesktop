// 交互式菜单与运行/调试命令
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	fmt.Println("  " + cGreen + "[1]" + cReset + " 全部平台 (Linux amd64 + arm64 + loong64 + Windows)")
	fmt.Println("  " + cGreen + "[2]" + cReset + " Windows (默认)")
	fmt.Println("  " + cGreen + "[3]" + cReset + " Windows + 控制台 (调试用)")
	fmt.Println("  " + cGreen + "[4]" + cReset + " Linux (amd64 / x86_64)")
	fmt.Println("  " + cGreen + "[5]" + cReset + " Linux (arm64 / aarch64)")
	fmt.Println("  " + cGreen + "[6]" + cReset + " Linux (loong64 / 龙芯)")
	fmt.Println("  " + cGreen + "[7]" + cReset + " 当前平台 (" + arch + ")")
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
		return "linux-arm64"
	case "6":
		return "linux-loong64"
	case "7":
		return "current"
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
		warnf("未找到 %s，创建默认配置...", envFile)
		f, _ := os.Create(envFile)
		if f != nil {
			f.WriteString("\n")
			f.WriteString("VITE_REMOTE_API_URL=https://your-api-server.com\n")
			f.WriteString("VITE_PROXY_PREFIXES=/api/,/storage/\n")
			f.Close()
		}
		fatalf("请修改 %s 中的 VITE_REMOTE_API_URL 后重新运行", envFile)
	}

	remoteURL, proxyPrefixes, _, err := readEnv(envFile)
	if err != nil {
		fatalf("错误: %v", err)
	}

	infof(">>> 启动 Vue 开发服务器 (%s)...", project)
	fmt.Println("远程 API:", remoteURL)
	fmt.Println("代理前缀:", proxyPrefixes)
	fmt.Println()

	cmd := exec.Command("pnpm", "run", "dev")
	cmd.Dir = vueDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fatalf("Vue 开发服务器启动失败: %v", err)
	}
}
