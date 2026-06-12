# WebToDesktop

将任意前端项目打包为桌面应用。Go + 系统原生 WebView，零运行时依赖。

## 演示项目

项目内置三个演示应用，可直接构建体验：

| 项目 | 目录 | 技术栈 | 演示功能 |
|------|------|--------|----------|
| `doc` | `demo/doc` | Vue 3 + Element Plus | 完整文档站点、桥接 API、凭证存储、窗口控制、**按键映射与快捷键拦截** |
| `serial` | `demo/serial` | Vue 3 | 串口扫描配置、双向数据通信、流式推送 |
| `xx` | `demo/xingxing` | 纯 HTML | **多效果锁屏动画**（太阳系/粒子星系/极光/数字雨）、按键映射禁用系统快捷键、密码保护关闭、`Ctrl+→/←` 切换效果 |

构建后运行即可体验核心功能。详情参见各项目目录下的 `index.html` 或 Vue 源码。

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
make run          # 构建并运行（生产模拟）
make dev          # 桌面窗口 + HMR 热更新

# 纯 HTML 项目
mkdir myapp && echo '<h1>Hello</h1>' > myapp/index.html
# 在 config.yaml 的 projects 中添加一行:
#   - name: "myapp", vue_dir: "myapp", description: "hello"
make dev          # 选 myapp → 自动构建运行
```

## 构建命令

| 命令 | 说明 |
|------|------|
| `make` | 交互式选择项目 + 平台 |
| `make run PROJECT=xxx` | 构建并运行（生产模拟） |
| `make dev PROJECT=xxx` | 桌面窗口 + HMR 热更新 |
| `make build-windows PROJECT=xxx` | Windows EXE |
| `make build-windows-console PROJECT=xxx` | Windows 控制台调试版 |
| `make build-linux-amd64 PROJECT=xxx` | Linux amd64 |
| `make build-linux-amd64-console PROJECT=xxx` | Linux amd64 控制台调试版 |
| `make build-linux-arm64 PROJECT=xxx` | Linux arm64 |
| `make build-linux-loong64 PROJECT=xxx` | Linux loong64（龙芯） |
| `make build-current-console PROJECT=xxx` | 当前平台控制台调试版 |
| `make list` | 列出所有可构建项目 |
| `make clean` | 清理构建产物 |

| 模式 | 右键菜单 | 控制台窗口 | 用途 |
|------|---------|-----------|------|
| `make run` / `build-*` | 禁用 | ❌ | 生产发布 |
| `make dev` / `build-*-console` | 启用 | ✅ | 开发调试 |

## 工作原理

构建工具自动完成：

1. 检测包管理器（pnpm/yarn/npm）和项目类型
2. 执行 `build` 脚本 → 复制产物到 embed 目录 → Gzip 压缩 → 编译 Go 二进制
3. 桌面窗口加载 embed 前端，`/api/*` 代理到远程服务器

## 支持的项目类型

| 类型 | 检测方式 | build 命令 | dev 模式 |
|------|---------|-----------|---------|
| Vite / Webpack / 构建工具 | `package.json` 有 `build`/`dev` 脚本 | `{pm} run build` | HMR 桌面窗口 |
| 纯 HTML | 无 `package.json`，有 `index.html` | 跳过，直接复制 | 构建后运行 |

> 包管理器从锁文件自动检测。无需配置。

### config.yaml（全局）

- `window.borderless: true` 无边框窗口
- `window.acrylic: false` Windows 毛玻璃
- `window.keyboard_shortcuts: true` 启用键盘快捷键拦截
- `window.key_mappings` 按键映射（由前端通过 `setKeyMapping` 动态管理，config.yaml 中留空）
- `security.aes_key` 务必修改
- `projects` 项目列表

### 安全：阻止浏览器直接访问

程序启动时在 `127.0.0.1` 上开启一个随机端口的 HTTP 服务，为 WebView 提供前端资源和 API 代理。为了防止同一台机器上的浏览器扫描端口后直接访问，每次启动时生成一个 32 字符的随机访问令牌：

1. 令牌注入到 WebView 的导航 URL：`http://127.0.0.1:PORT/?_wtd_=TOKEN`
2. 页面加载后注入的 JS 自动读取令牌并存储到 `window.__wtd_token__`
3. 所有 `fetch` / `XMLHttpRequest` 自动携带 `X-WTD-Token` 请求头
4. 服务器对每个请求校验令牌，无效则返回 `403 Forbidden`
5. 代理转发到远程服务器前自动剥离令牌头，防止泄露
6. 令牌每次启动随机，重启后旧令牌失效

> 开发模式下（`make dev`）不启用令牌校验，方便调试。

### 键盘快捷键拦截

启用 `keyboard_shortcuts: true` 后，程序使用 `WH_KEYBOARD_LL`（Windows）或 GTK `key-press-event`（Linux）拦截指定的键盘快捷键。

系统提供两层拦截机制：**按键映射**（粗粒度）和**组合快捷键注册**（细粒度）。

#### 按键映射（禁用整类系统快捷键）

通过前端 API 动态注册，将修饰键映射为自定义名称后，该键被钩子拦截，所有基于它的系统快捷键全部失效：

```javascript
await window.__lhpanda__('setKeyMapping', {key: 'Super_L', mappedName: 'Win'})
await window.__lhpanda__('listKeyMappings')
await window.__lhpanda__('clearKeyMappings')
```

> 被映射的按键以**原始键名**（如 `"Super_L"`）触发 `keyboard-shortcut` 事件，前端需自行维护名称转换。

#### 组合快捷键注册（精确控制）

注册特定组合键，仅拦截该组合：

```javascript
await window.__lhpanda__('registerShortcut', {keys: ['Ctrl+P', 'Shift+F1']})
await window.__lhpanda__('unregisterShortcut', {keys: ['Ctrl+P']})
await window.__lhpanda__('listShortcuts')
await window.__lhpanda__('resetShortcuts')
```

#### 默认拦截

| 快捷键 | 说明 |
|--------|------|
| `Ctrl+S` | 禁止保存页面 |

> 更多快捷键（`Alt+Tab`、`Alt+F4`、`Super_L` 等）需通过前端 API 动态注册。

#### 事件监听

被拦截的按键通过 `CustomEvent('keyboard-shortcut')` 推送到前端：

```javascript
const keyMappings = { Super_L: 'Win', Alt_L: 'Alt', Control_L: 'Ctrl' }
window.addEventListener('keyboard-shortcut', (e) => {
  const key = keyMappings[e.detail.key] || e.detail.key
  if (key === 'Alt+W') window.__lhpanda__('closeWindow', {})
})
```

#### 注意事项

- **按键映射**基于 `RegisterShortcut` 实现，与组合快捷键共用同一拦截列表
- 即使修饰键被映射，组合键仍可通过物理按键状态检测正常拦截（`GetAsyncKeyState` / `event->state`）
- **`Ctrl+Alt+Del`** 是 Windows 内核安全序列，任何用户态钩子均无法拦截
- `bridge` 方法调用需传 2 个参数：方法名 + 参数对象（无参数时传 `{}`），否则报 "arguments mismatch"
- 敏感 Windows API（`SetWindowsHookExA`、`GetAsyncKeyState`）在运行时动态解析，不出现在 PE 导入表中；钩子安装延迟 3~7 秒以降低杀毒软件误报

### .env.production（每个项目）

纯 HTML 项目自动生成。产物查找：`BUILD_OUTPUT_DIR` → `dist/` → 项目根。

### 按需裁剪模块

通过 `BUILD_TAGS` 选择编译的 bridge 模块：

```bash
# .env.production
BUILD_TAGS=minimal              # 仅核心（窗口控制+凭证+存储）
BUILD_TAGS=noserial             # 排除串口
BUILD_TAGS=nostream             # 排除流式推送
BUILD_TAGS=minimal,noserial    # 组合
```

| Tag | 排除的文件 |
|-----|-----------|
| `minimal` | 排除所有可选模块（串口、流式推送、凭证、存储） |
| `noserial` | serial.go + serial_bridge.go + go.bug.st/serial 依赖 |
| `nostream` | stream.go + streamdemo.go（及依赖它们的 serial） |
| `nocredentials` | credentials.go |
| `nostorage` | storage.go |

> 不配置则全量编译，向后兼容。

## 内置前端项目

| 项目 | 目录 | 说明 |
|------|------|------|
| `doc` | `demo/doc` | 完整文档站点（Vue 3），演示桥接 API、凭证存储、窗口控制、按键映射 |
| `serial` | `demo/serial` | 串口调试工具（Vue 3），演示流式数据推送、双向通信 |
| `xx` | `demo/xingxing` | 多效果锁屏动画（纯 HTML），演示按键映射、密码保护、效果切换 |
| `myapp` | `myapp/` | 纯 HTML 极简示例 |

## Go↔JS 流式数据推送

Go goroutine 通过 channel + `wv.Eval()` 主动推送数据到 JS，无需轮询。

```go
// Go 端
ch := bridge.NewStream(b, "my-topic", 64)
go func() {
    defer close(ch)
    for _, item := range dataSource() {
        ch <- item  // 自动推送到 JS CustomEvent('stream-data')
    }
}()
```

```javascript
// JS 端
import { useStream } from '../composables/useStream'
const { data, listen, stop } = useStream('my-topic')
await listen()  // data.value 实时更新
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
make dev PROJECT=debugtool          # 桌面窗口 + HMR 热更新（推荐）
make run PROJECT=demo               # 生产模拟
WTD_DEBUG=1 ./build/webtodesktop    # 崩溃时查看 wtd-debug.log
```

> `make dev` 自动启动 dev server，检测实际端口，桌面窗口导航到 dev server 地址。关闭窗口后自动清理 dev server 进程。

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
