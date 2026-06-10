# WebToDesktop

将任意前端项目打包为桌面应用。Go + 系统原生 WebView，零运行时依赖。

## 平台支持

| 平台 | 后端 | 编译依赖 |
|------|------|----------|
| Linux (amd64/arm64/loong64) | GTK + WebKit2GTK | `apt install libgtk-3-dev libwebkit2gtk-4.0-dev` |
| Windows (amd64) | Edge WebView2 | `apt install gcc-mingw-w64-x86-64` |

## 国产系统 & 龙芯适配

| 系统 | 架构 | 适配说明 |
|------|------|----------|
| 麒麟 V10 | loong64 / amd64 | 正常运行 |
| 统信 UOS | amd64 / arm64 | 正常运行 |
| 龙芯 (loong64) | 3A5000+ 等 | 自动禁用 WebKit GPU 加速 |

> 上述适配在程序启动时自动检测并应用，无需手动配置。

## 快速开始

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev

make              # 交互式构建（默认 Windows）
make run          # 构建并运行（调试模式）
make build-all PROJECT=demo  # 打包所有平台
```

## 工作原理

构建工具自动完成：读取 config.yaml → 构建前端 → Gzip 压缩 → 生成图标 → 编译 Go 二进制。

```
HTTP Server ← embed Vue 产物 → WebView 窗口
/api/* → 代理到远程服务器，/* → SPA 前端
window.__lhpanda__ ← Go↔JS 桥接
```

## 配置

### config.yaml（嵌入二进制）

- `window.borderless: true` 无边框窗口，需前端适配拖拽条 + 窗口按钮
- `window.webview_bg_transparent: false` 窗口背景透明（需前端适配透明背景）
- `window.acrylic: false` Windows 毛玻璃效果
- `security.aes_key` AES-256 密钥，务必修改

### .env.production（每个项目）

```bash
VITE_REMOTE_API_URL=https://api.example.com
VITE_PROXY_PREFIXES=/api/,/storage/
DESKTOP_ICON_LINUX=favicon.png
DESKTOP_ICON_WINDOWS=favicon.ico
```

## 前端桥接

```typescript
const { callBridge, isDesktop } = useBridge()

// 调用 Go 方法
await callBridge('saveCredentials', { username, password })
const { credentials } = await callBridge('getCredentials')

// 窗口控制
await callBridge('dragWindow')
await callBridge('toggleMaximize')
await callBridge('closeWindow')
```

桌面端用 `v-if="isDesktop"` 控制专属 UI。

## 键值对持久化存储

桌面端已透明接管浏览器原生 `localStorage` API，所有读写操作自动路由到 AES-256-GCM 加密存储：

```javascript
// ★ 代码完全不变，桌面端自动加密持久化
localStorage.setItem('theme', 'dark')
const theme = localStorage.getItem('theme')
```

也可使用 Vue composable 获取响应式绑定：

```javascript
import { useStorage } from '../composables/useStorage.js'
const storage = useStorage()
const themeRef = storage.useRef('theme', 'default-dark')
themeRef.value = 'light'  // 自动持久化 + 响应式更新
```

Vite 配置: `base: '/'`, `build.target: 'es2015'`（兼容旧版 WebKitGTK）。

## 调试

```bash
make run                        # 自动开启调试（WTD_DEBUG=1）
WTD_DEBUG=0 make run            # 关闭调试
WTD_DEBUG=1 ./build/webtodesktop  # 崩溃时查看 wtd-debug.log
```

## 依赖与致谢

| 依赖 | 许可证 |
|------|--------|
| [webview/webview_go](https://github.com/webview/webview_go) | MIT |
| [WebKitGTK](https://webkitgtk.org/) | LGPL |
| [Edge WebView2](https://developer.microsoft.com/microsoft-edge/webview2/) | Proprietary |

> WebKitGTK 和 WebView2 为系统运行时，不与本项目捆绑分发。

## 捐献

<p align="center"><img src="assets/donate.png" width="220"></p>

## 许可证

[MIT](LICENSE) © lhpanda
