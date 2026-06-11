# ============================================================
# WebToDesktop 构建工具
#
# 用法:
#   make                             交互式选择项目并构建（默认 Windows）
#   make build PROJECT=<name>        构建 Windows 版本（默认）
#   make build-linux PROJECT=<name>  构建 Linux（自动检测架构，BUILD_ARCH=arm64 可指定）
#   make build-linux-amd64  PROJECT=<name>  构建 Linux x86_64
#   make build-linux-arm64  PROJECT=<name>  构建 Linux ARM64
#   make build-windows PROJECT=<name> 构建 Windows EXE（同 make build）
#   make build-all PROJECT=<name>    打包所有平台 (Linux amd64 + arm64 + Windows)
#   make run   PROJECT=<name>        构建并运行（当前平台，默认调试模式）
#   make dev   PROJECT=<name>        启动 Vue 开发服务器
#   make list                        列出所有可构建项目
#   make clean                       清理构建产物
#
# 调试模式:
#   make run   → 默认开启调试模式（[WTD] 日志 + WebView 开发者工具）
#   make build → 默认生产模式（无调试输出）
#   关闭调试: WTD_DEBUG=0 make run
#
# 核心逻辑全部在 cmd/buildtool/ 中（Go 实现），Makefile 仅做参数转发。
# 也可直接使用: go run ./cmd/buildtool <命令>
# ============================================================

BUILD_TOOL := go run ./cmd/buildtool

# ———— 默认目标：交互式菜单 ————
.PHONY: all
all:
	@$(BUILD_TOOL)

# ———— 构建目标 ————
.PHONY: build build-all build-linux build-linux-amd64 build-linux-arm64 build-windows build-windows-console build-vue rebuild

# make build → 默认构建 Windows 版本
build:
	@$(BUILD_TOOL) windows $(PROJECT)

build-all:
	@$(BUILD_TOOL) all $(PROJECT)

build-linux build-linux-amd64 build-linux-arm64 build-linux-loong64 build-windows build-windows-console build-linux-amd64-console build-current-console:
	@$(BUILD_TOOL) $(subst build-,,$@) $(PROJECT)

build-vue:
	@$(BUILD_TOOL) build-vue $(PROJECT)

rebuild: build-vue
	@$(BUILD_TOOL) current $(PROJECT)

# ———— 运行 & 调试 ————
.PHONY: run run-windows dev

run:
	@$(BUILD_TOOL) run $(PROJECT)

run-windows:
	@$(BUILD_TOOL) run-win $(PROJECT)

dev:
	@$(BUILD_TOOL) dev $(PROJECT)

# ———— 工具 ————
.PHONY: list clean vet tidy help

list:
	@$(BUILD_TOOL) list

clean:
	@$(BUILD_TOOL) clean

vet:
	@$(BUILD_TOOL) vet

tidy:
	@$(BUILD_TOOL) tidy

help:
	@$(BUILD_TOOL) help
