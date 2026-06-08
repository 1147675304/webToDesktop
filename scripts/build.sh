#!/usr/bin/env bash
# ============================================================
# WebToDesktop 构建脚本
#
# 用法:
#   ./scripts/build.sh current  项目名    # 当前平台
#   ./scripts/build.sh linux    项目名    # Linux
#   ./scripts/build.sh windows  项目名    # Windows
#   ./scripts/build.sh windows-console 项目名  # Windows+控制台
#   ./scripts/build.sh list               # 列出项目
#   ./scripts/build.sh build-vue 项目名   # 仅构建 Vue
#
# 工作原理:
#   1. 从 config.yaml 读取项目列表
#   2. 读取前端项目 .env.production
#   3. 构建/复制 Vue 前端 → embed 目录
#   4. 从 favicon.ico 生成 EXE 图标资源
#   5. 通过 -ldflags -X 注入配置 → Go 二进制
# ============================================================

set -euo pipefail

# ———— 路径 ————
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_YAML="$PROJECT_ROOT/config.yaml"
OUTPUT_DIR="$PROJECT_ROOT/build"
APP_NAME="webtodesktop"
VUE_EMBED="$PROJECT_ROOT/vue/dist"

MINGW_CC="${MINGW_CC:-x86_64-w64-mingw32-gcc}"
MINGW_CXX="${MINGW_CXX:-x86_64-w64-mingw32-g++}"
MINGW_WINDRES="x86_64-w64-mingw32-windres"

# ———— 颜色 ————
C_GREEN='\033[0;32m'
C_YELLOW='\033[0;33m'
C_RED='\033[0;31m'
C_CYAN='\033[0;36m'
C_RESET='\033[0m'

# ———— 辅助函数 ————

# 从 config.yaml 解析项目配置
get_project_config() {
    local name="$1"
    local vue_dir
    vue_dir=$(grep -A 3 "name: \"$name\"" "$CONFIG_YAML" | grep 'vue_dir:' | head -1 | sed 's/.*vue_dir: *"\(.*\)".*/\1/')
    if [ -z "$vue_dir" ]; then
        echo -e "${C_RED}错误: 在 config.yaml 中找不到项目 '$name'${C_RESET}" >&2
        exit 1
    fi
    echo "$vue_dir"
}

# 列出所有项目
list_projects() {
    echo "可构建的前端项目:"
    echo ""
    local count=0
    grep '^  - name:' "$CONFIG_YAML" | sed 's/.*name: *"\(.*\)".*/\1/' | while IFS= read -r name; do
        count=$((count + 1))
        local desc
        desc=$(grep -A 2 "name: \"$name\"" "$CONFIG_YAML" | grep 'description:' | head -1 | sed 's/.*description: *"\(.*\)".*/\1/')
        printf "  ${C_GREEN}[%d]${C_RESET} %-20s — %s\n" "$count" "$name" "$desc"
    done
    echo ""
}

# 从 .env.production 读取环境变量，缺失则自动补全
read_env() {
    local env_file="$1"
    local remote_url proxy_prefixes desktop_sign_header

    remote_url=$(grep '^VITE_REMOTE_API_URL=' "$env_file" 2>/dev/null | cut -d'=' -f2- || true)
    if [ -z "$remote_url" ]; then
        echo -e "${C_YELLOW}╔══════════════════════════════════════════════╗${C_RESET}"
        echo -e "${C_YELLOW}║  检测到缺失配置项: VITE_REMOTE_API_URL        ║${C_RESET}"
        echo -e "${C_YELLOW}║  已自动添加默认值，请修改为实际远程地址:       ║${C_RESET}"
        echo -e "${C_YELLOW}║  → $env_file${C_RESET}"
        echo -e "${C_YELLOW}╚══════════════════════════════════════════════╝${C_RESET}"
        echo "" >> "$env_file"
        echo "# ★ 远程 API 服务器地址（由 tools/desktop 构建工具读取）" >> "$env_file"
        echo "VITE_REMOTE_API_URL=https://your-api-server.com" >> "$env_file"
        echo -e "${C_RED}请修改 $env_file 中的 VITE_REMOTE_API_URL 后重新构建${C_RESET}"
        exit 1
    fi

    proxy_prefixes=$(grep '^VITE_PROXY_PREFIXES=' "$env_file" 2>/dev/null | cut -d'=' -f2- || true)
    if [ -z "$proxy_prefixes" ]; then
        echo -e "${C_YELLOW}╔══════════════════════════════════════════════╗${C_RESET}"
        echo -e "${C_YELLOW}║  检测到缺失配置项: VITE_PROXY_PREFIXES        ║${C_RESET}"
        echo -e "${C_YELLOW}║  已自动添加默认值 (/api/, /storage/)         ║${C_RESET}"
        echo -e "${C_YELLOW}║  → $env_file${C_RESET}"
        echo -e "${C_YELLOW}╚══════════════════════════════════════════════╝${C_RESET}"
        echo "" >> "$env_file"
        echo "# ★ 代理到远程的路径前缀，逗号分隔（由 tools/desktop 构建工具读取）" >> "$env_file"
        echo "VITE_PROXY_PREFIXES=/api/,/storage/" >> "$env_file"
        proxy_prefixes="/api/,/storage/"
        echo ""
    fi

    # 桌面端签名请求头名（可选配置，默认 X-Desktop-Signature）
    desktop_sign_header=$(grep '^VITE_DESKTOP_SIGN_HEADER=' "$env_file" 2>/dev/null | cut -d'=' -f2- || true)
    if [ -z "$desktop_sign_header" ]; then
        desktop_sign_header="X-Desktop-Signature"
    fi

    echo "$remote_url"
    echo "$proxy_prefixes"
    echo "$desktop_sign_header"
}

# 生成 EXE 图标（PNG → ICO → syso）
# ★ 每次调用前强制清理旧资源，防止项目间图标串扰
generate_icon() {
    local favicon="$1"

    # 始终先删除旧的图标资源（即使本项目没有 favicon，也不应使用上一项目的残留）
    rm -f "$PROJECT_ROOT/rsrc.syso"

    if [ ! -f "$favicon" ]; then
        return 0  # 没有 favicon 则跳过
    fi

    echo -e "${C_YELLOW}>>> 生成 EXE 图标 ($favicon)...${C_RESET}"

    local ico_file="_app.ico"

    # Vue 的 favicon.ico 常常是 PNG 伪装的 → 转换为真正的 ICO
    if file "$favicon" | grep -q "PNG"; then
        echo "  检测到 PNG 格式，转换为 ICO..."
        if command -v convert &>/dev/null; then
            convert "$favicon" -define icon:auto-resize=256,128,64,48,32,16 "$ico_file"
        else
            echo -e "${C_YELLOW}  警告: ImageMagick convert 不可用，请安装: sudo apt install imagemagick${C_RESET}"
            return 0
        fi
    else
        ico_file="$favicon"
    fi

    if [ -f "$ico_file" ]; then
        echo "1 ICON \"$ico_file\"" > _icon.rc
        if $MINGW_WINDRES _icon.rc -o rsrc.syso 2>/dev/null; then
            echo "  图标资源: rsrc.syso"
        else
            echo -e "${C_YELLOW}  警告: windres 不可用，跳过图标生成${C_RESET}"
        fi
        rm -f _icon.rc
        if [ "$ico_file" = "_app.ico" ]; then
            rm -f "$ico_file"
        fi
    fi
}

# 复制 Vue 产物到 embed 目录
copy_vue_dist() {
    local dist_dir="$1"
    local project="$2"

    if [ ! -f "$dist_dir/index.html" ]; then
        echo -e "${C_RED}错误: $dist_dir 中未找到前端构建产物${C_RESET}"
        echo -e "${C_YELLOW}请先在该项目目录中执行: cd $dist_dir/.. && pnpm run build${C_RESET}"
        echo -e "${C_YELLOW}或使用: ./scripts/build.sh build-vue $project${C_RESET}"
        exit 1
    fi

    echo -e "${C_YELLOW}>>> 复制前端产物 ($project)...${C_RESET}"
    rm -rf "$VUE_EMBED"
    mkdir -p "$VUE_EMBED"
    cp -r "$dist_dir"/* "$VUE_EMBED/"
    echo ""
}

# Gzip 压缩 vue/dist 中所有文件（用 .gz 替换原文件）
# 必须在 generate_icon 之后调用（图标需要原始 favicon.ico）
gzip_vue_dist() {
    echo -e "${C_YELLOW}>>> Gzip 压缩前端资源 (-9)...${C_RESET}"
    find "$VUE_EMBED" -type f | while IFS= read -r f; do
        gzip -9 -c "$f" > "$f.gz" && rm "$f"
    done
    echo ""
}

# ———— 交互式辅助 ————

# 交互式选择项目，返回项目名
select_project() {
    list_projects >&2   # 列表输出到 stderr，避免污染返回值
    echo "" >&2

    read -p "请选择项目编号 (输入序号或名称): " choice
    if [ -z "$choice" ]; then
        echo -e "${C_RED}未选择项目，已取消${C_RESET}" >&2
        exit 1
    fi

    local found=""
    local count=0
    for name in $(grep '^  - name:' "$CONFIG_YAML" | sed 's/.*name: *"\(.*\)".*/\1/'); do
        count=$((count + 1))
        if [ "$choice" = "$count" ] || [ "$choice" = "$name" ]; then
            found="$name"
            break
        fi
    done

    if [ -z "$found" ]; then
        echo -e "${C_RED}错误: 无效的选择 '$choice'${C_RESET}" >&2
        exit 1
    fi

    echo "$found"
}

# 交互式选择平台，返回平台标识
select_platform() {
    echo -e "" >&2
    echo -e "请选择目标平台:" >&2
    echo -e "  ${C_GREEN}[1]${C_RESET} 当前平台" >&2
    echo -e "  ${C_GREEN}[2]${C_RESET} Linux" >&2
    echo -e "  ${C_GREEN}[3]${C_RESET} Windows" >&2
    echo -e "  ${C_GREEN}[4]${C_RESET} Windows + 控制台 (调试用)" >&2
    echo -e "" >&2

    read -p "请选择平台编号 [默认: 1]: " plat
    case "$plat" in
        2) echo "linux";;
        3) echo "windows";;
        4) echo "windows-console";;
        *) echo "current";;
    esac
}

# ———— 交互式入口（默认 make） ————

cmd_list() {
    list_projects
}

cmd_build_vue() {
    local project="$1"
    local vue_dir
    vue_dir=$(get_project_config "$project")

    echo -e "${C_GREEN}>>> 构建 Vue 前端 ($project)...${C_RESET}"
    (cd "$vue_dir" && pnpm run build) || { echo -e "${C_RED}Vue 构建失败${C_RESET}"; exit 1; }

    echo -e "${C_GREEN}>>> 复制产物到 embed 目录...${C_RESET}"
    rm -rf "$VUE_EMBED"
    mkdir -p "$VUE_EMBED"
    cp -r "$vue_dir/dist"/* "$VUE_EMBED/"
    gzip_vue_dist
    echo -e "${C_GREEN}Vue 构建完成 → vue/dist/${C_RESET}"
}

cmd_build() {
    local platform="$1"
    local project="$2"

    # 1. 解析项目配置
    local vue_dir
    vue_dir=$(get_project_config "$project")
    echo "项目目录: $vue_dir"

    # 2. 读取环境变量
    local env_file="$vue_dir/.env.production"
    if [ ! -f "$env_file" ]; then
        echo -e "${C_RED}错误: 找不到 $env_file${C_RESET}"
        exit 1
    fi

    # read_env 输出三行：remote_url、proxy_prefixes、desktop_sign_header
    local remote_url proxy_prefixes desktop_sign_header
    local env_output
    env_output=$(read_env "$env_file")
    remote_url=$(echo "$env_output" | sed -n '1p')
    proxy_prefixes=$(echo "$env_output" | sed -n '2p')
    desktop_sign_header=$(echo "$env_output" | sed -n '3p')

    echo "远程服务器: $remote_url"
    echo "代理前缀:   $proxy_prefixes"
    echo "签名请求头: $desktop_sign_header"
    echo ""

    # 3. 复制前端产物
    copy_vue_dist "$vue_dir/dist" "$project"

    # 4. 生成 EXE 图标（仅 Windows 平台需要，其他平台立即清理 rsrc.syso）
    generate_icon "$VUE_EMBED/favicon.ico"
    case "$platform" in
        windows|windows-console) ;;  # Windows 需要保留 rsrc.syso
        *) rm -f "$PROJECT_ROOT/rsrc.syso" ;;  # 非 Windows 立即删除，避免 Go 链接 COFF 失败
    esac
    echo ""

    # 4.5 Gzip 压缩所有前端文件（减小 EXE 体积 + 保护源码）
    gzip_vue_dist

    # 5. 构建 Go 二进制
    echo -e "${C_GREEN}>>> 构建 Go 二进制 ($platform)...${C_RESET}"
    mkdir -p "$OUTPUT_DIR"

    local ldflags="-s -w -X 'main.BuildRemoteURL=$remote_url' -X 'main.BuildProxyPrefixes=$proxy_prefixes' -X 'main.BuildProjectName=$project' -X 'main.BuildSignHeader=$desktop_sign_header'"

    case "$platform" in
        linux)
            echo "  目标: Linux (WebKitGTK)"
            CGO_ENABLED=1 go build -ldflags="$ldflags" -o "$OUTPUT_DIR/$APP_NAME-linux" .
            ;;
        windows)
            echo "  目标: Windows (Edge WebView2)"
            echo "  要求: sudo apt install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64"
            CGO_ENABLED=1 CGO_CXXFLAGS="-I$PROJECT_ROOT/include_win" \
                CC="$MINGW_CC" CXX="$MINGW_CXX" GOOS=windows GOARCH=amd64 \
                go build -ldflags="-H windowsgui $ldflags" -o "$OUTPUT_DIR/$APP_NAME.exe" .
            ;;
        windows-console)
            echo "  目标: Windows + 控制台 (调试用)"
            CGO_ENABLED=1 CGO_CXXFLAGS="-I$PROJECT_ROOT/include_win" \
                CC="$MINGW_CC" CXX="$MINGW_CXX" GOOS=windows GOARCH=amd64 \
                go build -ldflags="$ldflags" -o "$OUTPUT_DIR/$APP_NAME-console.exe" .
            ;;
        current)
            echo "  目标: 当前平台"
            CGO_ENABLED=1 go build -ldflags="$ldflags" -o "$OUTPUT_DIR/$APP_NAME" .
            ;;
        *)
            echo -e "${C_RED}错误: 未知平台 '$platform'${C_RESET}"
            exit 1
            ;;
    esac

    rm -f "$PROJECT_ROOT/rsrc.syso"

    echo ""
    echo -e "${C_GREEN}============================================${C_RESET}"
    echo -e "${C_GREEN}  构建完成!${C_RESET}"
    echo -e "${C_GREEN}============================================${C_RESET}"
    ls -lh "$OUTPUT_DIR/$APP_NAME"* 2>/dev/null || true
}

cmd_interactive() {
    local project
    project=$(select_project)
    local target
    target=$(select_platform)

    echo -e "${C_GREEN}已选择项目: $project  目标平台: $target${C_RESET}"
    echo ""

    cmd_build "$target" "$project"
}

cmd_clean() {
    rm -rf "$OUTPUT_DIR" "$PROJECT_ROOT/rsrc.syso" "$PROJECT_ROOT/_app.ico" "$PROJECT_ROOT/_icon.rc" "$VUE_EMBED"
    echo "已清理构建产物"
}

# ———— 运行 & 调试 ————

# 构建当前平台并运行（默认调试模式，输出 [WTD] 日志 + WebView 开发者工具）
# 不带参数 → 交互式选择项目；带参数 → 直接运行指定项目
# 若需关闭调试模式: WTD_DEBUG=0 make run
cmd_run() {
    local project="${1:-}"
    if [ -z "$project" ]; then
        project=$(select_project)
    fi
    cmd_build current "$project"

    # run 命令默认开启调试模式（可通过 WTD_DEBUG=0 显式关闭）
    if [ "${WTD_DEBUG:-}" != "0" ]; then
        export WTD_DEBUG=1
    fi

    echo ""
    echo -e "${C_GREEN}>>> 启动应用...${C_RESET}"
    exec "$OUTPUT_DIR/$APP_NAME"
}

# 启动 Vue 开发服务器（仅前端，不构建 Go）
# 不带参数 → 交互式选择项目；带参数 → 直接启动指定项目
cmd_dev() {
    local project="${1:-}"
    if [ -z "$project" ]; then
        project=$(select_project)
    fi

    local vue_dir
    vue_dir=$(get_project_config "$project")

    # 确保 .env.production 存在
    local env_file="$vue_dir/.env.production"
    if [ ! -f "$env_file" ]; then
        echo -e "${C_YELLOW}未找到 $env_file，创建默认配置...${C_RESET}"
        echo "" >> "$env_file"
        echo "VITE_REMOTE_API_URL=https://your-api-server.com" >> "$env_file"
        echo "VITE_PROXY_PREFIXES=/api/,/storage/" >> "$env_file"
        echo -e "${C_RED}请修改 $env_file 中的 VITE_REMOTE_API_URL 后重新运行${C_RESET}"
        exit 1
    fi

    local remote_url proxy_prefixes
    local env_output
    env_output=$(read_env "$env_file")
    remote_url=$(echo "$env_output" | head -1)
    proxy_prefixes=$(echo "$env_output" | tail -1)

    echo -e "${C_GREEN}>>> 启动 Vue 开发服务器 ($project)...${C_RESET}"
    echo "远程 API: $remote_url"
    echo "代理前缀: $proxy_prefixes"
    echo ""

    (cd "$vue_dir" && exec pnpm run dev)
}

# 构建 Windows 控制台版并尝试用 Wine 运行（仅 Linux 调试用）
# 不带参数 → 交互式选择项目；带参数 → 直接运行指定项目
cmd_run_windows() {
    local project="${1:-}"
    if [ -z "$project" ]; then
        project=$(select_project)
    fi
    cmd_build windows-console "$project"

    # run-win 命令默认开启调试模式（可通过 WTD_DEBUG=0 显式关闭）
    if [ "${WTD_DEBUG:-}" != "0" ]; then
        export WTD_DEBUG=1
    fi

    if command -v wine &>/dev/null; then
        echo ""
        echo -e "${C_GREEN}>>> 通过 Wine 启动 (调试模式)...${C_RESET}"
        exec wine "$OUTPUT_DIR/$APP_NAME-console.exe"
    else
        echo ""
        echo -e "${C_YELLOW}Wine 未安装，请手动将 EXE 复制到 Windows 运行:${C_RESET}"
        echo "  $OUTPUT_DIR/$APP_NAME-console.exe"
    fi
}

cmd_vet() {
    echo -e "${C_GREEN}>>> go vet...${C_RESET}"
    go vet ./...
}

cmd_tidy() {
    echo -e "${C_GREEN}>>> go mod tidy...${C_RESET}"
    go mod tidy
}

cmd_help() {
    echo "WebToDesktop 构建脚本"
    echo ""
    echo "用法:"
    echo "  ./scripts/build.sh                    交互式选择项目+平台"
    echo "  ./scripts/build.sh current  <项目名>   当前平台构建"
    echo "  ./scripts/build.sh linux    <项目名>   Linux 桌面版"
    echo "  ./scripts/build.sh windows  <项目名>   Windows EXE"
    echo "  ./scripts/build.sh windows-console <项目名>  Windows+控制台"
    echo "  ./scripts/build.sh run      <项目名>   构建并运行（当前平台）"
    echo "  ./scripts/build.sh run-win  <项目名>   构建 Windows 控制台版并 Wine 运行"
    echo "  ./scripts/build.sh dev      <项目名>   启动 Vue 开发服务器（仅前端）"
    echo "  ./scripts/build.sh build-vue <项目名>  仅构建 Vue 前端"
    echo "  ./scripts/build.sh list               列出所有项目"
    echo "  ./scripts/build.sh clean              清理构建产物"
    echo "  ./scripts/build.sh vet                代码静态检查"
    echo "  ./scripts/build.sh tidy               整理 Go 依赖"
    echo "  ./scripts/build.sh help               显示帮助"
}

# ———— 入口 ————

cd "$PROJECT_ROOT"

case "${1:-}" in
    list)
        cmd_list
        ;;
    build-vue)
        [ -z "${2:-}" ] && { echo -e "${C_RED}请指定项目名${C_RESET}"; exit 1; }
        cmd_build_vue "$2"
        ;;
    run)
        cmd_run "${2:-}"
        ;;
    run-win)
        cmd_run_windows "${2:-}"
        ;;
    dev)
        cmd_dev "${2:-}"
        ;;
    clean)
        cmd_clean
        ;;
    vet)
        cmd_vet
        ;;
    tidy)
        cmd_tidy
        ;;
    help|--help|-h)
        cmd_help
        ;;
    current|linux|windows|windows-console)
        [ -z "${2:-}" ] && { echo -e "${C_RED}请指定项目名${C_RESET}"; exit 1; }
        cmd_build "$1" "$2"
        ;;
    "")
        cmd_interactive
        ;;
    *)
        echo -e "${C_RED}错误: 未知命令 '$1'${C_RESET}"
        cmd_help
        exit 1
        ;;
esac
