<template>
  <div>
    <h2><el-icon><Switch /></el-icon> Go↔JS 桥接 API</h2>
    <p class="desc">前端通过 Bridge Proxy 直接调用 Go 方法 — <code>bridge.methodName(params)</code>，底层自动转为 <code>window.__lhpanda__('methodName', params)</code>。Go 端通过反射自动注册所有 <code>handle*</code> 方法，新增方法零配置。</p>

    <h3>前端调用（Bridge Proxy + useStream）</h3>
    <p class="hint">使用 <code>useBridge()</code> 获取 <code>bridge</code> 代理对象，所有 Go handler 自动可用。流式数据使用 <code>useStream(topic)</code> 监听 CustomEvent。</p>
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

// 流式数据推送 → 详情见「流式数据推送」文档
await bridge.listenStream({ topic: 'my-topic' })
await bridge.sendToStream({ topic: 'my-topic', data: {...} })

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

    <h3>创建自己的 API（零配置）</h3>

    <h4>1. 按功能模块创建文件</h4>
    <p class="desc">
      推荐将相关功能放在同一文件中，类似现有结构。每个文件只包含同类的 <code>handle*</code> 方法和辅助函数。
      文件名体现功能域，例如 <code>chat.go</code>、<code>database.go</code>、<code>notify.go</code>。
    </p>
    <CodeBlock lang="text">pkg/bridge/
├── bridge.go       # 核心调度器、反射自动注册、Call、Bridge 结构体
├── credentials.go  # 凭证管理：save/get/delete/clearCredentials
├── window.go       # 窗口控制：drag/resize/close/toggle*
├── config.go       # 窗口配置：get/saveWindowConfig + getString/getInt 等辅助函数
├── storage.go      # 键值对存储：set/get/remove/clear/getAllItems
├── stream.go       # 流式推送基础设施（NewStream、pushToJS 等）
├── streamdemo.go   # 流式推送示例（仅作为参考，可删除）
└── chat.go         # ★ 你新建的聊天 API（示例见下方）</CodeBlock>

    <h4>2. 写 Handler 方法</h4>
    <p class="hint">
      在新建的 <code>.go</code> 文件中添加 <code>func (b *Bridge) HandleXxx(...)</code> 方法即可。
      必须遵循签名约定，反射自动发现并注册为 JS 方法。
    </p>

    <p class="hint">
      <b>命名规则</b>：Go 方法 <code>HandleSendMessage</code> → JS 方法 <code>"sendMessage"</code>（去掉 Handle，首字母小写）。<br/>
      <b>签名约定</b>：<code>func (b *Bridge) HandleXxx(params map[string]interface{}) (interface{}, error)</code><br/>
      <b>返回值</b>：<code>(data, nil)</code> → JS 收到 <code>{success: true, data}</code>；<code>(nil, err)</code> → JS <code>.catch(err)</code>
    </p>
    <CodeBlock lang="go">// pkg/bridge/chat.go — 聊天 API 完整示例
package bridge

import "fmt"

// ———— 聊天 ————

// HandleSendMessage 发送一条聊天消息。
//
// JS 调用: bridge.sendMessage({ from: "panda", text: "hello" })
// 返回:    { sent: true, echo: "panda: hello" }
func (b *Bridge) HandleSendMessage(params map[string]interface{}) (interface{}, error) {
    from := getString(params, "from")
    text := getString(params, "text")
    if from == "" || text == "" {
        return nil, fmt.Errorf("缺少 from 或 text 参数")
    }
    // 你的业务逻辑...
    return map[string]interface{}{
        "sent": true,
        "echo": from + ": " + text,
    }, nil
}

// HandleGetHistory 获取聊天记录。
//
// JS 调用: bridge.getHistory({ room: "general", limit: 20 })
// 返回:    { messages: [{from:"panda", text:"hello"}, ...] }
func (b *Bridge) HandleGetHistory(params map[string]interface{}) (interface{}, error) {
    room := getString(params, "room")
    limit := getInt(params, "limit")
    if room == "" {
        room = "general"
    }
    if limit <= 0 {
        limit = 50
    }
    // 从数据库/文件读取历史...
    messages := []map[string]interface{}{} // 示例空数组
    return map[string]interface{}{
        "room":     room,
        "messages": messages,
    }, nil
}

// HandleClearHistory 清空聊天记录。
//
// JS 调用: bridge.clearHistory({ room: "general" })
// 返回:    { cleared: true }
func (b *Bridge) HandleClearHistory(params map[string]interface{}) (interface{}, error) {
    room := getString(params, "room")
    if room == "" {
        return nil, fmt.Errorf("缺少 room 参数")
    }
    // 清空逻辑...
    return map[string]interface{}{"cleared": true}, nil
}</CodeBlock>

    <h4>3. 参数提取辅助函数</h4>
    <p class="hint">
      <code>pkg/bridge/config.go</code> 提供了参数类型转换函数。JS 的 number → Go float64，需显式转换。
    </p>
    <CodeBlock lang="go">// 内置辅助函数（pkg/bridge/config.go，所有 handler 可直接使用）
func getString(params map[string]interface{}, key string) string   // 提取 string
func getInt(params map[string]interface{}, key string) int         // 提取 int（从 float64 转换）
func getFloat(params map[string]interface{}, key string) float64   // 提取 float64
func getBool(params map[string]interface{}, key string) bool       // 提取 bool

// 如需更复杂的参数（嵌套对象、数组），直接用类型断言:
nested, ok := params["config"].(map[string]interface{})
items, ok := params["list"].([]interface{})</CodeBlock>

    <h4>4. JS 端立即可用</h4>
    <p class="hint">
      启动应用后，日志会输出 <code>[bridge] auto-registered: sendMessage → HandleSendMessage</code>。
      JS 端通过 Bridge Proxy 直接调用，无需任何额外配置。
    </p>
    <CodeBlock lang="javascript">const { bridge } = useBridge()

// 发送消息
const result = await bridge.sendMessage({ from: 'panda', text: 'hello' })
// result = { success: true, data: { sent: true, echo: 'panda: hello' } }

// 获取历史
const history = await bridge.getHistory({ room: 'general', limit: 20 })
// history = { success: true, data: { room: 'general', messages: [...] } }

// 清空历史
await bridge.clearHistory({ room: 'general' })
// → { success: true, data: { cleared: true } }</CodeBlock>

    <h4>5. 组织建议</h4>
    <table class="api-table">
      <thead><tr><th>原则</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td>一文件一功能域</td><td>每个 <code>.go</code> 文件只包含一类 API，如 <code>chat.go</code>、<code>database.go</code></td></tr>
        <tr><td>方法名语义化</td><td><code>HandleSendMessage</code> 而非 <code>HandleMsg1</code>，JS 方法名会自动去掉 Handle</td></tr>
        <tr><td>错误时返回 error</td><td>返回 <code>(nil, fmt.Errorf("..."))</code>，JS 端 <code>.catch(err)</code> 就能拿到</td></tr>
        <tr><td>零配置</td><td>只需写 <code>handle*</code> 方法，无需 init()、Register()、配置文件</td></tr>
        <tr><td>无循环导入</td><td>handler 文件都在 <code>pkg/bridge/</code> 包内，可互相调用</td></tr>
        <tr><td>可访问 Store</td><td>通过 <code>b.store</code> 访问持久化存储（凭证/配置等）</td></tr>
        <tr><td>可访问 WebView</td><td>通过 <code>b.wv</code> 调用 <code>Eval()</code>/<code>Dispatch()</code>（如流式推送）</td></tr>
      </tbody>
    </table>

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
        <tr><td colspan="4" class="section"><el-icon><Bell /></el-icon> Go→JS 流式数据推送</td></tr>
        <tr><td><code>listenStream</code></td><td><code>{ topic }</code></td><td><code>{ listening, topic }</code></td><td><code>handleListenStream</code></td></tr>
        <tr><td><code>stopStream</code></td><td><code>{ topic }</code></td><td><code>{ stopped, topic }</code></td><td><code>handleStopStream</code></td></tr>
        <tr><td><code>sendToStream</code></td><td><code>{ topic, data }</code></td><td><code>{ sent, topic }</code></td><td><code>handleSendToStream</code></td></tr>
        <tr><td><code>listStreams</code></td><td>—</td><td><code>{ streams: [...] }</code></td><td><code>handleListStreams</code></td></tr>
        <tr><td colspan="4" class="section"><el-icon><Document /></el-icon> README 流式演示</td></tr>
        <tr><td><code>startReadmeStream</code></td><td><code>{ startLine?, speed? }</code></td><td><code>{ listening, totalLines }</code></td><td><code>handleStartReadmeStream</code></td></tr>
        <tr><td><code>pauseReadmeStream</code></td><td>—</td><td><code>{ paused }</code></td><td><code>handlePauseReadmeStream</code></td></tr>
        <tr><td><code>resumeReadmeStream</code></td><td>—</td><td><code>{ resumed }</code></td><td><code>handleResumeReadmeStream</code></td></tr>
        <tr><td><code>seekReadmeStream</code></td><td><code>{ line }</code></td><td><code>{ seek, line }</code></td><td><code>handleSeekReadmeStream</code></td></tr>
        <tr><td><code>getReadmeState</code></td><td>—</td><td><code>{ hasStream, paused }</code></td><td><code>handleGetReadmeState</code></td></tr>
        <tr><td><code>stopReadmeStream</code></td><td>—</td><td><code>{ stopped }</code></td><td><code>handleStopReadmeStream</code></td></tr>
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
import { Switch, Tickets, Lock, Setting, FolderOpened, Bell, Document } from '@element-plus/icons-vue'
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
