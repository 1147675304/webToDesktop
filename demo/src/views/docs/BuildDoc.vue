<template>
  <div>
    <h2>📦 构建与部署</h2>
    <p class="desc">使用 Makefile 管理构建流程，支持交叉编译 Windows EXE（含图标）和 Linux 桌面版。</p>

    <h3>构建命令</h3>
    <table class="api-table">
      <thead><tr><th>命令</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td><code>make</code></td><td>交互式选择项目 + 平台</td></tr>
        <tr><td><code>make run PROJECT=demo</code></td><td>构建并运行（当前平台调试）</td></tr>
        <tr><td><code>make build-windows PROJECT=demo</code></td><td>交叉编译 Windows EXE（含图标）</td></tr>
        <tr><td><code>make build-linux PROJECT=demo</code></td><td>编译 Linux 桌面版</td></tr>
        <tr><td><code>make dev PROJECT=demo</code></td><td>仅启动 Vue 开发服务器</td></tr>
        <tr><td><code>make list</code></td><td>列出所有可构建项目</td></tr>
        <tr><td><code>make clean</code></td><td>清理构建产物</td></tr>
      </tbody>
    </table>

    <h3>Go 项目结构</h3>
    <CodeBlock lang="text">desktop/
├── main.go                    ← 唯一入口，嵌入资源 + 编排启动
├── config.yaml                ← 构建配置（嵌入二进制）
├── Makefile                   ← 构建脚本入口
├── scripts/build.sh           ← 构建逻辑（交叉编译/gzip/图标）
├── pkg/                       ← Go 业务逻辑包
│   ├── config.go              ← Config 类型 + InitConfig
│   ├── store.go               ← AES 加密存储 + 凭证/配置持久化
│   ├── server.go              ← HTTP 服务器 + 智能代理 + gzip 处理
│   ├── webview.go             ← 原生 WebView 窗口创建
│   └── bridge/                ← Go↔JS 桥接子包
│       ├── bridge.go          ← Bridge 调度器
│       ├── credentials.go     ← 凭证管理
│       ├── window.go          ← 窗口控制
│       └── config.go          ← 窗口配置
├── native/                    ← 平台原生 API
│   ├── config.go              ← WindowConfig + Apply/PreShow 接口
│   ├── windows.go             ← Win32/DWM API
│   ├── linux.go               ← GTK3 API
│   └── other.go               ← 其他平台桩
├── demo/                      ← Vue 3 演示项目
│   └── src/
│       ├── App.vue            ← 主布局（拖拽条+菜单+路由）
│       ├── composables/       ← 可复用逻辑（useBridge/useCredentials）
│       ├── components/        ← 通用组件（CodeBlock 等）
│       └── views/             ← 页面组件（Home/Preferences/Docs）
└── vue/                       ← 构建时的前端产出目录（go:embed）</CodeBlock>

    <h3>添加新项目</h3>
    <p class="hint">在 <code>config.yaml</code> 中添加：</p>
    <CodeBlock lang="yaml">projects:
  - name: "my-app"
    vue_dir: "../my-app/vue"
    description: "我的应用"</CodeBlock>

    <p class="hint">然后在 Vue 项目的 <code>.env.production</code> 中配置远程 API：</p>
    <CodeBlock lang="bash">VITE_REMOTE_API_URL=https://api.example.com
VITE_PROXY_PREFIXES=/api/,/storage/</CodeBlock>

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
