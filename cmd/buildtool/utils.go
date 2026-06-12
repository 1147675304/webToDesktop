// 简单命令与工具函数
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ———— 输出函数 ————

func warnf(format string, args ...any) {
	fmt.Printf(cYellow+format+cReset+"\n", args...)
}

func infof(format string, args ...any) {
	fmt.Printf(cGreen+format+cReset+"\n", args...)
}

func cyanf(format string, args ...any) {
	fmt.Printf(cCyan+format+cReset+"\n", args...)
}

// ———— 命令执行 ————

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = projectRoot
	return cmd.Run()
}

func runCmdInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	return cmd.Run()
}

func runCmdCombined(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = projectRoot
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// ———— 系统检测 ————

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ———— 辅助函数 ————

func envOr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func printBuildOutputs() {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		info, _ := entry.Info()
		fmt.Printf("  %s  %s\n", info.Mode(), filepath.Join(outputDir, entry.Name()))
	}
}

// ———— 简单命令 ————

func cmdList() {
	cfg, err := loadConfig()
	if err != nil {
		fatalf("错误: %v", err)
	}
	fmt.Println("可构建的前端项目:")
	fmt.Println()
	for i, p := range cfg.Projects {
		fmt.Printf("  "+cGreen+"[%d]"+cReset+" %-20s — %s\n", i+1, p.Name, p.Description)
	}
	fmt.Println()
}

func cmdClean() {
	os.RemoveAll(outputDir)
	os.Remove(filepath.Join(projectRoot, "rsrc.syso"))
	os.Remove(filepath.Join(projectRoot, "_app.ico"))
	os.Remove(filepath.Join(projectRoot, "_icon.rc"))
	os.RemoveAll(vueEmbed)
	fmt.Println("已清理构建产物")
}

func cmdVet() {
	infof(">>> go vet...")
	runCmd("go", "vet", "./...")
}

func cmdTidy() {
	infof(">>> go mod tidy...")
	runCmd("go", "mod", "tidy")
}

func cmdHelp() {
	fmt.Println("WebToDesktop 构建工具 (Go)")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run ./cmd/buildtool                    交互式选择项目+平台")
	fmt.Println("  go run ./cmd/buildtool current  <项目名>   当前平台构建")
	fmt.Println("  go run ./cmd/buildtool linux    <项目名>   Linux (自动检测架构)")
	fmt.Println("  go run ./cmd/buildtool linux-amd64   <项目名> Linux x86_64")
	fmt.Println("  go run ./cmd/buildtool linux-arm64   <项目名> Linux ARM64 (aarch64)")
	fmt.Println("  go run ./cmd/buildtool linux-loong64 <项目名> Linux loong64 (龙芯)")
	fmt.Println("  go run ./cmd/buildtool windows    <项目名>   Windows EXE")
	fmt.Println("  go run ./cmd/buildtool windows-console <项目名>  Windows+控制台")
	fmt.Println("  go run ./cmd/buildtool all      <项目名>   打包所有平台")
	fmt.Println("  go run ./cmd/buildtool run      <项目名>   构建并运行（当前平台）")
	fmt.Println("  go run ./cmd/buildtool run-win  <项目名>   构建 Windows 并 Wine 运行")
	fmt.Println("  go run ./cmd/buildtool dev      <项目名>   启动 Vue 开发服务器（仅前端）")
	fmt.Println("  go run ./cmd/buildtool build-vue <项目名>  仅构建 Vue 前端")
	fmt.Println("  go run ./cmd/buildtool list               列出所有项目")
	fmt.Println("  go run ./cmd/buildtool clean              清理构建产物")
	fmt.Println("  go run ./cmd/buildtool vet                代码静态检查")
	fmt.Println("  go run ./cmd/buildtool tidy               整理 Go 依赖")
	fmt.Println("  go run ./cmd/buildtool help               显示帮助")
}
