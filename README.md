# WebToDesktop

将任意前端项目打包为独立桌面应用。基于 Go + 系统原生 WebView，零运行时依赖。

## 平台支持

| 平台 | WebView 后端 | 依赖 |
|------|-------------|------|
| Linux | GTK + WebKit2GTK | `sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev` |
| Windows | Edge WebView2 | 系统内置（Win10 1809+），无需额外安装 |
| macOS | Cocoa WKWebView | 系统内置 |

## 快速开始

```bash
# 1. 安装依赖（Linux）
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev

# 2. 进入 desktop 目录
cd desktop

# 3. 交互式构建
make

# 或指定项目 + 平台
make build PROJECT=<name>              # 当前平台
make build-linux PROJECT=<name>        # Linux
make build-windows PROJECT=<name>      # Windows (需 MinGW)
```

## 工作原理

```
┌──────────────────────────────────────────────────────────┐
│  桌面 EXE (main.go 编排)                                 │
│  ┌────────────────┐  ┌────────────┐  ┌───────────────┐ │
│  │ pkg/server.go  │  │ vue/ (embed)│  │ pkg/webview.go│ │
│  │ HTTP + 代理    │  │ 静态资源    │  │ 原生 WebView   │ │
│  │ /api/* ────────┼──┼─→ 远程 API │  │               │ │
│  │ /*     ────────┼──┼─→ SPA      │  │               │ │
│  └────────────────┘  └────────────┘  └───────────────┘ │
│                                                            │
│  pkg/bridge/ → Go↔JS 桥接 (window.__lhpanda__)            │
│  ├── bridge.go       调度器 + 方法注册                     │
│  ├── credentials.go  凭证管理 (AES-256-GCM)                │
│  ├── window.go       窗口控制 (拖拽/最大化/全屏/关闭)       │
│  └── config.go       窗口配置持久化 + 辅助函数             │
└──────────────────────────────────────────────────────────┘
```

## 构建命令

```bash
make                              # 交互式菜单（选项目 + 选平台）
make build PROJECT=<name>         # 构建指定项目（当前平台）
make build-linux PROJECT=<name>   # 构建 Linux 版
make build-windows PROJECT=<name> # 交叉编译 Windows EXE
make list                         # 列出所有可构建项目
make clean                        # 清理构建产物
```

## 项目结构

```
desktop/
├── main.go              # 唯一入口：embed 静态资源 + 配置 → 编排启动
├── config.yaml          # 构建配置（嵌入二进制）
├── Makefile             # 构建脚本入口 → scripts/build.sh
├── pkg/                 # Go 业务逻辑包
│   ├── config.go        # Config 类型 + InitConfig
│   ├── store.go         # AES-256-GCM 加密存储 + 凭证/窗口配置持久化
│   ├── server.go        # HTTP 服务器 + 智能代理 + gzip 支持
│   ├── webview.go       # 原生 WebView 窗口创建 (RunApp)
│   ├── webview_win.go   # Windows CGo 构建标记 (//go:build windows)
│   └── bridge/          # Go↔JS 桥接子包
│       ├── bridge.go    # Bridge 调度器 + 方法注册
│       ├── credentials.go ← 凭证管理 handlers
│       ├── window.go    ← 窗口控制 handlers
│       └── config.go    ← 窗口配置 handlers + 辅助函数
├── native/              # 平台原生窗口 API
│   ├── config.go        # WindowConfig + Apply/PreShow 接口
│   ├── windows.go       # Win32/DWM API
│   ├── linux.go         # GTK3 API
│   └── other.go         # 其他平台桩实现
├── scripts/
│   └── build.sh         # 构建逻辑（编译/gzip/图标）
├── demo/                # 前端演示项目
├── vue/                 # 构建时前端产出目录（go:embed，目录名可自定义）
└── include_win/         # Windows C++ 头文件补丁
```

## 配置体系

| 文件 | 用途 | 构建时 |
|------|------|--------|
| `config.yaml` | AES 密钥、窗口外观、项目列表 | embed 嵌入 |
| `.env.production` | 远程 API 地址、代理前缀 | ldflags 注入 |
| `vite.config.ts` | 前端构建配置（base: './'） | 运行时生效 |

## config.yaml 说明

```yaml
app:
  name: "WebToDesktop"      # 应用名（决定数据目录 ~/.config/<name>/）
  version: "1.0.0"

security:
  aes_key: "your-key"       # AES-256 加密密钥（SHA256 派生）

window:
  title: "后备标题"          # 若未通过 ldflags 注入项目名，使用此标题
  width: 1024
  height: 700
  fullscreen: false
  maximized: false
  borderless: true           # 无边框窗口（需前端适配）
  opacity: 0.95
  window_position: "1, 1"
  dark_title_bar: true       # Windows 暗色标题栏
  round_corners: true        # Windows 圆角
  acrylic: true              # Windows 毛玻璃效果

projects:
  - name: "my-project"       # make build PROJECT=my-project
    vue_dir: "../my-frontend" # 前端项目路径（相对于 desktop/）
    description: "项目描述"
```

## 前端适配

### 桥接模块
前端通过 `composables/useBridge.ts` 调用 Go 方法：

```typescript
import { useBridge } from '@/composables/useBridge'
import { useCredentials } from '@/composables/useCredentials'

const { callBridge, isDesktop } = useBridge()
const { credList, credForm, saveCred, loadCreds, deleteCred, clearCreds } = useCredentials()

// callBridge('methodName', { key: 'value' }) — 通用桥接调用
// isDesktop — 桌面端为 true（浏览器模式自动降级到 mock）
```

### 登录页集成

```typescript
import { useBridge } from '@/composables/useBridge'

const { callBridge } = useBridge()

// 保存凭据
if (rememberPwd.value) {
    await callBridge('saveCredentials', {
        username: form.username,
        password: form.password,
    })
}

// 读取已保存凭据
const result = await callBridge('getCredentials')
if (result.credentials?.length) {
    // 自动填充用户名
}
```

### 无边框窗口适配
若 `borderless: true`，前端 `App.vue` 需包含：

- **拖拽条** — `<div class="drag-bar" @mousedown="handleDragWindow">`（顶部 8px）
- **窗口控制** — 最大化/全屏/关闭按钮
- **调整手柄** — 左/右/下/左下/右下 resize 区域
- **标题栏拖拽** — Layout.vue 中 `el-header @mousedown.self` 支持

桌面端专属元素使用 `v-if="isDesktop"` 控制显示（`isDesktop` 来自 `useBridge()`）。

### 路由配置
`vite.config.js` 必须使用绝对路径：

```javascript
export default defineConfig({
    base: '/'
})
```

## 依赖

### 编译时
- Go 1.23+
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev`
- Windows 交叉编译: `gcc-mingw-w64-x86-64`, `g++-mingw-w64-x86-64`

### 运行时
- Linux: GTK3 + WebKit2GTK 运行时库
- Windows: Edge WebView2 Runtime（Win10/11 内置，Win7 需手动安装）

## 依赖与致谢

本项目基于以下优秀的开源项目构建：

| 依赖 | 用途 | 许可证 |
|---|---|---|
| [webview/webview_go](https://github.com/webview/webview_go) | Go 原生 WebView 绑定 | MIT |
| [WebKitGTK](https://webkitgtk.org/) | Linux WebView 渲染引擎 | LGPL |
| [Edge WebView2](https://developer.microsoft.com/microsoft-edge/webview2/) | Windows WebView 渲染引擎 | Proprietary |
| [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) | YAML 配置解析 | MIT |

> WebKitGTK (LGPL) 和 Edge WebView2 均为系统级运行时依赖，由用户环境提供，不与本项目二进制捆绑分发，不影响本项目的 MIT 授权。

## 捐献

如果这个项目对你有帮助，欢迎请作者喝杯咖啡 ☕

<p align="center">
  <img src="assets/donate.png" width="220" alt="赞赏码">
</p>

## 许可证

[MIT](LICENSE) © lhpanda
