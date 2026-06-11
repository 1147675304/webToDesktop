// 构建工具入口、路径初始化、命令调度
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 颜色常量
const (
	cGreen  = "\033[0;32m"
	cYellow = "\033[0;33m"
	cRed    = "\033[0;31m"
	cCyan   = "\033[0;36m"
	cReset  = "\033[0m"
)

// 全局路径
var (
	projectRoot string
	configYAML  string
	outputDir   string
	appName     = "webtodesktop"
	vueEmbed    string
)

func initPaths() {
	exe, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		projectRoot = cwd
	} else {
		projectRoot = filepath.Dir(exe)
	}

	if strings.HasSuffix(projectRoot, "cmd/buildtool") || strings.HasSuffix(projectRoot, "cmd/buildtool/") {
		projectRoot = filepath.Dir(filepath.Dir(projectRoot))
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "config.yaml")); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		if _, err := os.Stat(filepath.Join(cwd, "config.yaml")); err == nil {
			projectRoot = cwd
		}
	}

	configYAML = filepath.Join(projectRoot, "config.yaml")
	outputDir = filepath.Join(projectRoot, "build")
	vueEmbed = filepath.Join(projectRoot, "vue", "dist")
}

func main() {
	initPaths()

	if err := os.Chdir(projectRoot); err != nil {
		fatalf("无法切换到项目根目录: %v", err)
	}

	args := os.Args[1:]

	if len(args) == 0 {
		cmdInteractive()
		return
	}

	cmd := args[0]
	project := ""
	if len(args) >= 2 {
		project = args[1]
	}

	switch cmd {
	case "list":
		cmdList()
	case "build-vue":
		if project == "" {
			fatalf("请指定项目名")
		}
		cmdBuildVue(project)
	case "run":
		if project == "" {
			project = selectProject()
		}
		cmdRun(project)
	case "run-win":
		if project == "" {
			project = selectProject()
		}
		cmdRunWindows(project)
	case "dev":
		if project == "" {
			project = selectProject()
		}
		cmdDev(project)
	case "clean":
		cmdClean()
	case "vet":
		cmdVet()
	case "tidy":
		cmdTidy()
	case "help", "--help", "-h":
		cmdHelp()
	case "current", "current-console", "linux", "linux-amd64", "linux-amd64-console", "linux-arm64", "linux-loong64", "windows", "windows-console":
		if project == "" {
			fatalf("请指定项目名")
		}
		if err := cmdBuild(cmd, project); err != nil {
			fatalf("构建失败: %v", err)
		}
	case "all":
		if project == "" {
			project = selectProject()
		}
		cmdAll(project)
	default:
		fatalf("错误: 未知命令 '%s'", cmd)
		cmdHelp()
	}
}

// fatalf 打印错误并退出
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, cRed+format+cReset+"\n", args...)
	os.Exit(1)
}
