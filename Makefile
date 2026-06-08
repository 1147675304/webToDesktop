# ============================================================
# WebToDesktop 构建工具
#
# 用法:
#   make                             交互式选择项目并构建
#   make build PROJECT=<name>        构建指定项目（当前平台）
#   make build-windows PROJECT=<name> 交叉编译 Windows EXE
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
# 核心逻辑全部在 scripts/build.sh 中，Makefile 仅做参数转发。
# ============================================================

BUILD_SCRIPT := ./scripts/build.sh

# ———— 默认目标：交互式菜单 ————
.PHONY: all
all:
	@$(BUILD_SCRIPT)

# ———— 构建目标 ————
.PHONY: build build-linux build-windows build-windows-console build-vue rebuild

build build-linux build-windows build-windows-console:
	@$(BUILD_SCRIPT) $(subst build-,,$@) $(PROJECT)

build-vue:
	@$(BUILD_SCRIPT) build-vue $(PROJECT)

rebuild: build-vue
	@$(BUILD_SCRIPT) current $(PROJECT)

# ———— 运行 & 调试 ————
.PHONY: run run-windows dev

run:
	@$(BUILD_SCRIPT) run $(PROJECT)

run-windows:
	@$(BUILD_SCRIPT) run-win $(PROJECT)

dev:
	@$(BUILD_SCRIPT) dev $(PROJECT)

# ———— 工具 ————
.PHONY: list clean vet tidy help

list:
	@$(BUILD_SCRIPT) list

clean:
	@$(BUILD_SCRIPT) clean

vet:
	@$(BUILD_SCRIPT) vet

tidy:
	@$(BUILD_SCRIPT) tidy

help:
	@$(BUILD_SCRIPT) help
