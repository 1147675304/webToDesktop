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

// copyVueDist 复制 Vue 构建产物到 embed 目录
func copyVueDist(distDir, project string) error {
	if !fileExists(filepath.Join(distDir, "index.html")) {
		fatalf("错误: %s 中未找到前端构建产物", distDir)
		fmt.Printf(cYellow+"请先在该项目目录中执行: cd %s/.. && pnpm run build"+cReset+"\n", distDir)
		fmt.Printf(cYellow+"或使用: go run ./cmd/buildtool build-vue %s"+cReset+"\n", project)
		os.Exit(1)
	}

	fmt.Println(cYellow + ">>> 复制前端产物 (" + project + ")..." + cReset)
	os.RemoveAll(vueEmbed)
	os.MkdirAll(vueEmbed, 0755)

	entries, err := os.ReadDir(distDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(distDir, entry.Name())
		dst := filepath.Join(vueEmbed, entry.Name())
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("复制 %s 失败: %w", src, err)
		}
	}
	fmt.Println()
	return nil
}

// gzipVueDist 压缩 vue/dist 中所有文件（原文件替换为 .gz）
func gzipVueDist() {
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
