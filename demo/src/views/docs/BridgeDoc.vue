<template>
  <div>
    <h2><el-icon><Switch /></el-icon> Go↔JS 桥接 API</h2>
    <p class="desc">前端通过 Bridge Proxy 直接调用 Go 方法 — <code>bridge.methodName(params)</code>，底层自动转为 <code>window.__lhpanda__('methodName', params)</code>。Go 端通过反射自动注册所有 <code>handle*</code> 方法，新增方法零配置。</p>

    <h3>前端调用（Bridge Proxy）</h3>
    <p class="hint">使用 <code>useBridge()</code> 获取 <code>bridge</code> 代理对象，所有 Go handler 自动可用：</p>
    <CodeBlock lang="javascript">import { useBridge } from './composables/useBridge.js'

const { bridge } = useBridge()

// 应用信息
const info = await bridge.getAppInfo()
// → { platform: 'linux', arch: 'amd64', version: '1.0.0' }

// 凭证管理
await bridge.saveCredentials({ username: 'admin', password: '123456' })
const creds = await bridge.getCredentials()
// → { found: true, credentials: [{ username: 'admin' }] }
await bridge.deleteCredentials({ username: 'admin' })
await bridge.clearCredentials()

// 窗口控制
await bridge.toggleMaximize()
await bridge.toggleMinimize()
await bridge.closeWindow()
await bridge.resizeWindow({ edge: 11 })   // 11 = 右边缘

// 窗口配置
const cfg = await bridge.getWindowConfig()
await bridge.saveWindowConfig({ title: '我的应用', width: 1280, ... })

// 方法发现
const { methods } = await bridge.listMethods()
// → ['getAppInfo', 'saveCredentials', 'dragWindow', ...]</CodeBlock>

    <h3>初始化流程</h3>
    <CodeBlock lang="javascript">import { useBridge } from './composables/useBridge.js'

const { bridge, appInfo, isDesktop, initAppInfo } = useBridge()

// App.vue onMounted 中:
await initAppInfo()  // 调用 bridge.getAppInfo() 获取平台信息
// appInfo.platform = 'linux' | 'windows' | 'browser'
// isDesktop = true 当非浏览器时
// bridge 代理立即可用，所有方法自动路由</CodeBlock>

    <h3>Go 端：新增方法（零配置）</h3>
    <p class="hint">只需在任意 <code>pkg/bridge/*.go</code> 文件中添加 <code>handleXxx</code> 方法，反射自动发现并注册。命名规则：<code>handleGetAppInfo</code> → JS 方法 <code>"getAppInfo"</code>。</p>
    <CodeBlock lang="go">// pkg/bridge/mycustom.go — 新建文件或添加到现有文件

// ★ 只需写这个函数，无需 init()，无需修改任何注册代码
func (b *Bridge) handleMyCustomMethod(params map[string]interface{}) (interface{}, error) {
    name := getString(params, "name")
    return map[string]interface{}{
        "greeting": "Hello " + name,
        "ok":       true,
    }, nil
}

// JS 端自动可用: bridge.myCustomMethod({ name: 'World' })
// 启动日志: [bridge] auto-registered: myCustomMethod → handleMyCustomMethod</CodeBlock>

    <h3>反射自动注册原理</h3>
    <CodeBlock lang="go">// pkg/bridge/bridge.go — registerBuiltins()
func (b *Bridge) registerBuiltins() {
    t := reflect.TypeOf(b)        // *Bridge 的类型信息
    for i := 0; i &lt; t.NumMethod(); i++ {
        name := t.Method(i).Name
        if !strings.HasPrefix(name, "handle") { continue }  // 过滤非 handler
        // 验证签名: func(map[string]interface{}) (interface{}, error)
        // handleGetAppInfo → "getAppInfo"
        jsName := strings.ToLower(name[6:7]) + name[7:]
        b.methods[jsName] = /* 闭包绑定方法索引 */
    }
}</CodeBlock>

    <h3>全部可用方法</h3>
    <table class="api-table">
      <thead><tr><th>JS 方法</th><th>参数</th><th>返回值</th><th>Go 方法</th></tr></thead>
      <tbody>
        <tr><td colspan="4" class="section"><el-icon><Tickets /></el-icon> 应用信息</td></tr>
        <tr><td><code>getAppInfo</code></td><td>—</td><td><code>{ platform, arch, version }</code></td><td><code>handleGetAppInfo</code></td></tr>
        <tr><td><code>listMethods</code></td><td>—</td><td><code>{ methods: [...] }</code></td><td><code>handleListMethods</code></td></tr>
        <tr><td colspan="4" class="section"><el-icon><Lock /></el-icon> 凭证管理</td></tr>
        <tr><td><code>saveCredentials</code></td><td><code>{ username, password }</code></td><td><code>{ saved: true }</code></td><td><code>handleSaveCredentials</code></td></tr>
        <tr><td><code>getCredentials</code></td><td><code>{ username? }</code></td><td><code>{ found, credentials[] }</code></td><td><code>handleGetCredentials</code></td></tr>
        <tr><td><code>deleteCredentials</code></td><td><code>{ username }</code></td><td><code>{ deleted: true }</code></td><td><code>handleDeleteCredentials</code></td></tr>
        <tr><td><code>clearCredentials</code></td><td>—</td><td><code>{ cleared: true }</code></td><td><code>handleClearCredentials</code></td></tr>
        <tr><td colspan="4" class="section">🪟 窗口控制</td></tr>
        <tr><td><code>dragWindow</code></td><td>—</td><td><code>{ ok: true }</code></td><td><code>handleDragWindow</code></td></tr>
        <tr><td><code>resizeWindow</code></td><td><code>{ edge: 10~17 }</code></td><td><code>{ ok: true }</code></td><td><code>handleResizeWindow</code></td></tr>
        <tr><td><code>toggleMinimize</code></td><td>—</td><td><code>{ ok: true }</code></td><td><code>handleToggleMinimize</code></td></tr>
        <tr><td><code>toggleMaximize</code></td><td>—</td><td><code>{ ok: true }</code></td><td><code>handleToggleMaximize</code></td></tr>
        <tr><td><code>toggleFullscreen</code></td><td>—</td><td><code>{ ok: true }</code></td><td><code>handleToggleFullscreen</code></td></tr>
        <tr><td><code>closeWindow</code></td><td>—</td><td><code>{ ok: true }</code></td><td><code>handleCloseWindow</code></td></tr>
        <tr><td colspan="4" class="section"><el-icon><FolderOpened /></el-icon> 键值对存储</td></tr>
        <tr><td><code>setItem</code></td><td><code>{ key, value }</code></td><td><code>{ saved: true }</code></td><td><code>handleSetItem</code></td></tr>
        <tr><td><code>getItem</code></td><td><code>{ key }</code></td><td><code>{ found, value }</code></td><td><code>handleGetItem</code></td></tr>
        <tr><td><code>removeItem</code></td><td><code>{ key }</code></td><td><code>{ removed: true }</code></td><td><code>handleRemoveItem</code></td></tr>
        <tr><td><code>clearItems</code></td><td>—</td><td><code>{ cleared: true }</code></td><td><code>handleClearItems</code></td></tr>
        <tr><td><code>getAllItems</code></td><td>—</td><td><code>{ items: {...} }</code></td><td><code>handleGetAllItems</code></td></tr>
        <tr><td colspan="4" class="section"><el-icon><Setting /></el-icon> 窗口配置</td></tr>
        <tr><td><code>getWindowConfig</code></td><td>—</td><td><code>{ title, width, ... }</code></td><td><code>handleGetWindowConfig</code></td></tr>
        <tr><td><code>saveWindowConfig</code></td><td><code>{ title, ... }</code></td><td><code>{ saved, needRestart }</code></td><td><code>handleSaveWindowConfig</code></td></tr>
      </tbody>
    </table>

    <h3>edge 参数（resizeWindow）</h3>
    <table class="api-table">
      <thead><tr><th>值</th><th>位置</th><th>光标</th></tr></thead>
      <tbody>
        <tr><td>10</td><td>左边框</td><td>←→</td></tr>
        <tr><td>11</td><td>右边框</td><td>←→</td></tr>
        <tr><td>12</td><td>上边框</td><td>↕</td></tr>
        <tr><td>13</td><td>左上角</td><td>↖</td></tr>
        <tr><td>14</td><td>右上角</td><td>↗</td></tr>
        <tr><td>15</td><td>下边框</td><td>↕</td></tr>
        <tr><td>16</td><td>左下角</td><td>↙</td></tr>
        <tr><td>17</td><td>右下角</td><td>↘</td></tr>
      </tbody>
    </table>

    <h3>JS↔Go 类型映射</h3>
    <table class="api-table">
      <thead><tr><th>JavaScript</th><th>JSON</th><th>Go (interface{})</th></tr></thead>
      <tbody>
        <tr><td><code>123</code></td><td>number</td><td><code>float64</code></td></tr>
        <tr><td><code>true / false</code></td><td>boolean</td><td><code>bool</code></td></tr>
        <tr><td><code>"hello"</code></td><td>string</td><td><code>string</code></td></tr>
        <tr><td><code>null</code></td><td>null</td><td><code>nil</code></td></tr>
        <tr><td><code>{ a: 1 }</code></td><td>object</td><td><code>map[string]interface{}</code></td></tr>
        <tr><td><code>[1, 2]</code></td><td>array</td><td><code>[]interface{}</code></td></tr>
      </tbody>
    </table>

    <h3>架构总览</h3>
    <CodeBlock lang="text">JS 侧                                    Go 侧
───────                                  ──────
bridge.saveCredentials({u,p})            Bridge.Call("saveCredentials", {u,p})
  │                                        │
  │ Proxy 拦截 .saveCredentials            │ 查表 methods["saveCredentials"]
  │ → callBridge('saveCredentials', p)     │ → handleSaveCredentials(params)
  │   │                                    │   │
  │   │ window.__lhpanda__(method, p)      │   │ Store.SaveCredentials()
  │   │ ──── webview JSON ────→            │   │
  │   │ ←─── JSON response ────           │   │
  │   └─ resolve/reject                   │   └─ return {saved: true}
  └─ await 拿到结果                         │
                                            │ ★ 注册: 反射扫描 handle* 方法
                                            │   无需 init(), 无需手动注册</CodeBlock>
  </div>
</template>

<script setup>
import { Switch, Tickets, Lock, Setting } from '@element-plus/icons-vue'
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.desc code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 12px; }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; }
.hint code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }

.api-table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 12px; }
.api-table th, .api-table td { padding: 8px 10px; text-align: left; border-bottom: 1px solid var(--border); }
.api-table th { color: var(--text-secondary); font-weight: 500; font-size: 12px; }
.api-table code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }
.api-table .section {
  background: var(--card-bg); color: var(--accent); font-weight: 600;
  font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px;
}
</style>
