// 资源处理：图标生成、前端产物复制、Gzip 压缩
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// generateIcon 生成 Windows EXE 图标资源（PNG/ICO → rsrc.syso）
func generateIcon(favicon string) {
	os.Remove(filepath.Join(projectRoot, "rsrc.syso"))

	if !fileExists(favicon) {
		return
	}

	fmt.Println(cYellow + ">>> 生成 EXE 图标 (" + favicon + ")..." + cReset)

	icoFile := favicon
	out, _ := runCmdCombined("file", favicon)
	if strings.Contains(out, "PNG") || strings.Contains(out, "SVG") || strings.Contains(out, "JPEG") {
		fmt.Println("  检测到非 ICO 格式，转换为 ICO...")
		if commandExists("convert") {
			icoFile = filepath.Join(projectRoot, "_app.ico")
			runCmd("convert", favicon, "-define", "icon:auto-resize=256,128,64,48,32,16", icoFile)
		} else {
			warnf("  警告: ImageMagick convert 不可用，跳过 Windows 图标")
			warnf("  安装: sudo apt install imagemagick")
			return
		}
	}

	if fileExists(icoFile) {
		rcFile := filepath.Join(projectRoot, "_icon.rc")
		os.WriteFile(rcFile, []byte("1 ICON \""+icoFile+"\"\n"), 0644)
		if runCmd("x86_64-w64-mingw32-windres", rcFile, "-o", "rsrc.syso") == nil {
			fmt.Println("  图标资源: rsrc.syso")
		} else {
			warnf("  警告: windres 不可用，跳过 Windows 图标")
		}
		os.Remove(rcFile)
		if strings.HasSuffix(icoFile, "_app.ico") {
			os.Remove(icoFile)
		}
	}
}

// copyDist 复制前端构建产物到 embed 目录。
//
// 按以下优先级查找产物源目录:
//   1. .env.production 中 BUILD_OUTPUT_DIR 指定的目录
//   2. {projectDir}/dist （Vite/Webpack 等构建工具的默认输出）
//   3. {projectDir} 本身（纯 HTML 项目，无构建步骤）
func copyDist(projectDir, project string) error {
	// 优先级 1: 读取 .env.production 配置
	envFile := filepath.Join(projectDir, ".env.production")
	envBytes, _ := os.ReadFile(envFile)
	envStr := string(envBytes)
	if outDir := extractEnvValue(envStr, "BUILD_OUTPUT_DIR"); outDir != "" {
		distDir := filepath.Join(projectDir, outDir)
		if fileExists(filepath.Join(distDir, "index.html")) {
			return doCopyDist(distDir, project)
		}
	}

	// 优先级 2: 标准 dist/ 目录
	distDir := filepath.Join(projectDir, "dist")
	if fileExists(filepath.Join(distDir, "index.html")) {
		return doCopyDist(distDir, project)
	}

	// 优先级 3: 项目根目录本身（纯 HTML 等项目）
	if fileExists(filepath.Join(projectDir, "index.html")) {
		fmt.Println(cYellow + ">>> 检测到纯前端项目（无构建步骤），直接复制项目文件..." + cReset)
		return doCopyDist(projectDir, project)
	}

	fatalf("错误: %s 中未找到前端构建产物（dist/index.html 或 index.html）", projectDir)
	fmt.Printf(cYellow+"请先在该项目目录中执行: cd %s && pnpm run build"+cReset+"\n", projectDir)
	fmt.Printf(cYellow+"或使用: go run ./cmd/buildtool build-vue %s"+cReset+"\n", project)
	os.Exit(1)
	return nil
}

// doCopyDist 执行实际的文件复制操作。
func doCopyDist(srcDir, project string) error {
	fmt.Println(cYellow + ">>> 复制前端产物 (" + project + ")..." + cReset)
	os.RemoveAll(vueEmbed)
	os.MkdirAll(vueEmbed, 0755)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", srcDir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		// 跳过常见的非产物目录/文件
		if name == "node_modules" || name == ".git" || name == ".env.production" || name == "package.json" || name == "pnpm-lock.yaml" || name == "tsconfig.json" || name == "vite.config.ts" || strings.HasPrefix(name, "src") {
			continue
		}
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(vueEmbed, name)
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("复制 %s 失败: %w", src, err)
		}
	}
	fmt.Println()
	return nil
}

// gzipDist 压缩 embed 目录中所有文件为 .gz（原文件替换）
func gzipDist() {
	fmt.Println(cYellow + ">>> Gzip 压缩前端资源 (-9)..." + cReset)
	filepath.Walk(vueEmbed, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		cmd := exec.Command("gzip", "-9", "-c")
		cmd.Stdin = strings.NewReader(string(data))
		out, err := cmd.Output()
		if err != nil {
			return nil
		}
		os.WriteFile(path+".gz", out, 0644)
		os.Remove(path)
		return nil
	})
	fmt.Println()
}

// ———— 文件复制工具 ————

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	info, _ := os.Stat(src)
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
