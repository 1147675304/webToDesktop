//go:build linux

package pkg

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed icon.png
var iconPNG []byte

// AppIconPath 临时 PNG 图标路径，由 init() 从嵌入数据写入。
var AppIconPath string

func init() {
	f, err := os.CreateTemp("", "wtd-icon-*.png")
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Write(iconPNG); err != nil {
		os.Remove(f.Name())
		return
	}
	abs, _ := filepath.Abs(f.Name())
	if abs != "" {
		AppIconPath = abs
	} else {
		AppIconPath = f.Name()
	}
}
