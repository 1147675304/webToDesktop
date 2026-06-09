// 架构检测与 Linux CGO 交叉编译环境设置
package main

import (
	"fmt"
	"os"
	"runtime"
)

func detectArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "loong64":
		return "loong64"
	default:
		warnf("警告: 未知架构 '%s'，默认使用 amd64", runtime.GOARCH)
		return "amd64"
	}
}

func validateArch(arch string) error {
	switch arch {
	case "amd64", "arm64", "loong64":
		return nil
	default:
		return fmt.Errorf("不支持的架构 '%s'，支持: amd64, arm64, loong64", arch)
	}
}

func getTargetArch() string {
	if arch := os.Getenv("BUILD_ARCH"); arch != "" {
		if err := validateArch(arch); err != nil {
			fatalf("错误: %v", err)
		}
		return arch
	}
	return detectArch()
}

// setupLinuxCrossCC 设置 Linux CGO 交叉编译环境（CC / CXX / PKG_CONFIG）
func setupLinuxCrossCC(targetArch string) error {
	hostArch := detectArch()
	if targetArch == hostArch {
		return nil
	}

	var crossPrefix string
	switch hostArch + "→" + targetArch {
	case "amd64→arm64":
		crossPrefix = "aarch64-linux-gnu"
	case "arm64→amd64":
		crossPrefix = "x86_64-linux-gnu"
	case "loong64→arm64":
		crossPrefix = "aarch64-linux-gnu"
	case "loong64→amd64":
		crossPrefix = "x86_64-linux-gnu"
	}

	if crossPrefix == "" {
		if targetArch == "loong64" {
			printBox(cYellow,
				"LoongArch (loong64) 交叉编译",
				"",
				"当前架构: "+hostArch,
				"目标架构: loong64 (龙芯)",
				"",
				"由于缺少 loong64 交叉编译工具链，",
				"请在龙芯机器上本地编译此平台。",
				"",
				"命令: go run ./cmd/buildtool linux-loong64 <项目名>",
			)
			return fmt.Errorf("缺少 loong64 交叉编译工具链，请在龙芯机器上本地编译")
		}
		// 未知的交叉编译组合，提示用户
		printBox(cYellow,
			"不支持的交叉编译组合",
			"",
			"当前架构: "+hostArch,
			"目标架构: "+targetArch,
			"",
			"请在目标架构机器上本地编译此平台。",
		)
		return fmt.Errorf("不支持的交叉编译组合 %s→%s，请在目标架构机器上本地编译", hostArch, targetArch)
	}

	cc := crossPrefix + "-gcc"
	cxx := crossPrefix + "-g++"
	pkgConfig := crossPrefix + "-pkg-config"

	if !commandExists(cc) {
		printBox(cYellow,
			"缺少 "+targetArch+" 交叉编译工具链",
			"",
			"请安装:",
			aptInstallHint(targetArch),
			"",
			"建议: 在 "+targetArch+" 机器上本地编译更可靠",
		)
		return fmt.Errorf("缺少交叉编译工具链")
	}

	if !commandExists(pkgConfig) {
		printBox(cYellow,
			"⚠ CGO 交叉编译 "+targetArch+" 需要完整 sysroot",
			"",
			"当前缺少 "+pkgConfig,
			"以及 "+targetArch+" 的 GTK/WebKit 开发库",
			"",
			"由于此项目依赖 GTK3 + WebKit2GTK (CGO)，",
			"跨架构交叉编译在实际中不可行。",
			"",
			"★ 推荐方案: 在 "+targetArch+" 机器上本地编译",
			"★ CI/CD: 使用 GitHub Actions ARM64 runner",
		)
		return fmt.Errorf("缺少目标架构 sysroot")
	}

	os.Setenv("CC", cc)
	os.Setenv("CXX", cxx)
	if commandExists(pkgConfig) {
		os.Setenv("PKG_CONFIG", pkgConfig)
	}
	warnf("  交叉编译器: %s", cc)
	return nil
}

func aptInstallHint(arch string) string {
	switch arch {
	case "arm64":
		return "sudo apt install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu"
	case "amd64":
		return "sudo apt install gcc-x86-64-linux-gnu g++-x86-64-linux-gnu"
	case "loong64":
		return "sudo apt install gcc-loongarch64-linux-gnu g++-loongarch64-linux-gnu"
	}
	return ""
}

func printBox(color string, lines ...string) {
	fmt.Println(color + "╔══════════════════════════════════════════════╗" + cReset)
	for _, l := range lines {
		fmt.Printf(color+"║  %-44s  ║"+cReset+"\n", l)
	}
	fmt.Println(color + "╚══════════════════════════════════════════════╝" + cReset)
}
