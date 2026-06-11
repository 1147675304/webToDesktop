<template>
  <div>
    <h2><el-icon><Box /></el-icon> 构建与部署</h2>
    <p class="desc">支持 Vite / Webpack / 纯 HTML 等多种前端项目类型，自动检测包管理器，一键打包为桌面应用。</p>

    <h3>构建命令</h3>
    <table class="api-table">
      <thead><tr><th>命令</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td><code>make</code></td><td>交互式选择项目 + 平台（含全部平台一键构建）</td></tr>
        <tr><td><code>make dev PROJECT=xxx</code></td><td>桌面窗口 + HMR 热更新（开发首选）</td></tr>
        <tr><td><code>make run PROJECT=xxx</code></td><td>构建并运行（生产模拟）</td></tr>
        <tr><td><code>make build-windows PROJECT=xxx</code></td><td>Windows EXE</td></tr>
        <tr><td><code>make build-windows-console PROJECT=xxx</code></td><td>Windows 控制台调试版（含右键）</td></tr>
        <tr><td><code>make build-linux-amd64 PROJECT=xxx</code></td><td>Linux amd64</td></tr>
        <tr><td><code>make build-linux-amd64-console PROJECT=xxx</code></td><td>Linux amd64 控制台调试版</td></tr>
        <tr><td><code>make build-current-console PROJECT=xxx</code></td><td>当前平台控制台调试版</td></tr>
        <tr><td><code>make list</code></td><td>列出所有可构建项目</td></tr>
        <tr><td><code>make clean</code></td><td>清理构建产物</td></tr>
      </tbody>
    </table>

    <h3>支持的项目类型</h3>
    <table class="api-table">
      <thead><tr><th>类型</th><th>检测方式</th><th>行为</th></tr></thead>
      <tbody>
        <tr><td>Vite / Webpack 等</td><td><code>package.json</code> 有 <code>build</code> 脚本</td><td>执行 <code>{pm} run build</code> → 复制 <code>dist/</code></td></tr>
        <tr><td>纯 HTML</td><td>无 <code>package.json</code>，有 <code>index.html</code></td><td>跳过构建，直接复制项目文件</td></tr>
      </tbody>
    </table>

    <p class="hint">包管理器从锁文件自动检测：<code>pnpm-lock.yaml</code> → pnpm，<code>yarn.lock</code> → yarn，<code>package-lock.json</code> → npm。产物目录优先级：<code>BUILD_OUTPUT_DIR</code> → <code>dist/</code> → 项目根目录。</p>

    <h3>Go 项目结构</h3>
    <CodeBlock lang="text">webtodesktop/
├── main.go                    ← 入口，嵌入资源 + 编排启动
├── go.mod / go.sum            ← Go 模块依赖
├── config.yaml                ← 构建配置（嵌入二进制）
├── Makefile                   ← 构建脚本
├── cmd/buildtool/             ← 构建工具（make 调用的入口）
│   ├── main.go                ← 命令分发
│   ├── build.go               ← 构建逻辑（交叉编译 / gzip / 图标）
│   ├── run.go                 ← 运行 / dev / 交互菜单
│   ├── config.go              ← 配置解析（config.yaml / .env）
│   └── assets.go              ← 图标生成 / 产物复制 / gzip
├── pkg/                       ← Go 运行时包
│   ├── config.go              ← Config 类型 + InitConfig
│   ├── store.go               ← AES 加密存储
│   ├── server.go              ← HTTP 服务器 + 智能代理
│   ├── webview.go             ← 原生 WebView 窗口
│   ├── webview_linux.go       ← Linux 平台 WebView
│   ├── webview_win.go         ← Windows 平台 WebView
│   └── bridge/                ← Go↔JS 桥接
│       ├── bridge.go          ← 核心调度器 + 反射自动注册
│       ├── credentials.go     ← 凭证管理
│       ├── storage.go         ← 键值对存储
│       ├── window.go          ← 窗口控制
│       ├── config.go          ← 窗口配置 + getString 等辅助
│       ├── stream.go          ← 流式推送基础设施
│       └── serial.go / serial_bridge.go  ← 串口 + 网络桥接
├── native/                    ← 平台原生 API
│   ├── linux.go               ← GTK3
│   ├── windows.go             ← Win32
│   └── other.go               ← 其他平台桩
├── demo/                      ← Vue 3 演示项目
├── debugtool/                 ← 串口调试工具
├── myapp/                     ← 纯 HTML 示例
└── vue/dist/                  ← 构建时前端嵌入目录</CodeBlock>

    <h3>添加新项目</h3>
    <p class="hint">在 <code>config.yaml</code> 中添加一行即可：</p>
    <CodeBlock lang="yaml">projects:
  - name: "my-app"
    vue_dir: "../my-app/vue"
    description: "我的应用"

  # 纯 HTML 项目也支持
  - name: "hello"
    vue_dir: "hello"
    description: "纯 HTML 示例"</CodeBlock>

    <p class="hint">纯 HTML 项目只需一个 <code>index.html</code> 文件，<code>.env.production</code> 自动生成。Vue/React 等项目则需配置 <code>.env.production</code>：</p>
    <CodeBlock lang="bash"># 远程 API 地址
VITE_REMOTE_API_URL=https://api.example.com
# 代理前缀（逗号分隔）
VITE_PROXY_PREFIXES=/api/,/storage/
# 桌面图标（从 dist/ 目录查找）
# Linux 使用 PNG 格式（建议 ≥ 64x64）
DESKTOP_ICON_LINUX=favicon.png
# Windows 使用 ICO 格式（建议含多尺寸 16~256）
DESKTOP_ICON_WINDOWS=favicon.ico

# 按需裁剪 bridge 模块
BUILD_TAGS=minimal             # 仅核心模块
# BUILD_TAGS=noserial          # 排除串口
# BUILD_TAGS=minimal,noserial  # 组合</CodeBlock>

    <h3>模块裁剪</h3>
    <p class="hint">如果项目不需要某些 bridge 功能，通过 <code>BUILD_TAGS</code> 排除，减小二进制体积。</p>
    <table class="api-table">
      <thead><tr><th>Tag</th><th>排除的模块</th><th>节省的依赖</th></tr></thead>
      <tbody>
        <tr><td><code>minimal</code></td><td>全部可选模块</td><td>~500KB</td></tr>
        <tr><td><code>noserial</code></td><td>串口 + 网络桥接</td><td>go.bug.st/serial (~500KB)</td></tr>
        <tr><td><code>nostream</code></td><td>流式推送 + 串口</td><td>—</td></tr>
        <tr><td><code>nocredentials</code></td><td>凭证管理</td><td>—</td></tr>
        <tr><td><code>nostorage</code></td><td>键值对存储</td><td>—</td></tr>
      </tbody>
    </table>

    <h3>窗口配置 (config.yaml)</h3>
    <table class="api-table">
      <thead><tr><th>配置项</th><th>类型</th><th>默认</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td><code>title</code></td><td>string</td><td>—</td><td>窗口标题</td></tr>
        <tr><td><code>width</code></td><td>int</td><td>1024</td><td>窗口宽度 (px)</td></tr>
        <tr><td><code>height</code></td><td>int</td><td>768</td><td>窗口高度 (px)</td></tr>
        <tr><td><code>fullscreen</code></td><td>bool</td><td>false</td><td>启动时全屏</td></tr>
        <tr><td><code>maximized</code></td><td>bool</td><td>false</td><td>启动时最大化</td></tr>
        <tr><td><code>borderless</code></td><td>bool</td><td>true</td><td>无边框窗口</td></tr>
        <tr><td><code>always_on_top</code></td><td>bool</td><td>false</td><td>始终置顶</td></tr>
        <tr><td><code>opacity</code></td><td>float</td><td>1.0</td><td>不透明度 (0.0~1.0)</td></tr>
        <tr><td><code>window_position</code></td><td>string</td><td>"center"</td><td>位置：""/"center"/"x,y"</td></tr>
        <tr><td><code>webview_bg_transparent</code></td><td>bool</td><td>false</td><td>WebView 背景透明（跨平台）</td></tr>
        <tr><td><code>input_passthrough</code></td><td>bool</td><td>false</td><td>透明区域点击穿透（Linux）</td></tr>
        <tr><td><code>acrylic</code></td><td>bool</td><td>false</td><td>毛玻璃背景 (Win)</td></tr>
        <tr><td><code>round_corners</code></td><td>bool</td><td>true</td><td>圆角窗口 (Win11)</td></tr>
        <tr><td><code>dark_title_bar</code></td><td>bool</td><td>false</td><td>暗色标题栏 (Win10+)</td></tr>
      </tbody>
    </table>

    <h3>前置依赖</h3>
    <CodeBlock lang="bash"># Linux 构建依赖
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev

# Windows 交叉编译依赖（在 Linux 上构建 Windows EXE）
sudo apt install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64</CodeBlock>
  </div>
</template>

<script setup lang="ts">
import { Box } from '@element-plus/icons-vue'
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; }

.api-table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 12px; }
.api-table th, .api-table td { padding: 8px 10px; text-align: left; border-bottom: 1px solid var(--border); }
.api-table th { color: var(--text-secondary); font-weight: 500; font-size: 12px; }
.api-table code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }
</style>
