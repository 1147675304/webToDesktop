<template>
  <div>
    <h2><el-icon><Bell /></el-icon> Go↔JS 双向流式数据推送</h2>
    <p class="desc">
      基于 <b>Go channel + wv.Eval() + CustomEvent + Bridge RPC</b> 实现的双向数据流。
      Go 端主动推送数据到 JS，JS 端随时通过 Bridge 反向控制流的行为（启动/暂停/恢复/跳转/停止）。
    </p>

    <!-- ========== 架构概览 ========== -->
    <h3>架构概览</h3>
    <CodeBlock lang="text">┌──────────────────────────────────────────────────────────────┐
│                         Go 进程                              │
│                                                              │
│  ┌─────────────┐     chan      ┌───────────────┐              │
│  │  生产者      │─────────────→│   消费者        │              │
│  │ goroutine   │              │  goroutine     │              │
│  │             │              │               │              │
│  │ for {       │              │ for data :=    │              │
│  │   data :=   │              │   range ch {   │              │
│  │   produce() │              │   wv.Eval(     │              │
│  │   ch <- data│              │     CustomEvent│              │
│  │ }           │              │   )            │              │
│  └─────┬───────┘              │ }              │              │
│        │ 控制信号              └───────────────┘              │
│  ┌─────▼───────┐                      │                       │
│  │  ctrl 结构   │ ←──── Bridge RPC ───│─────── window  ──────│
│  │ { paused,   │      window.__lhpanda__     .dispatchEvent  │
│  │   seekLine, │      方向: JS→Go            方向: Go→JS     │
│  │   cancel }  │                                              │
│  └─────────────┘                                              │
└──────────────────────────────────────────────────────────────┘

   JS 侧                                      数据流方向
  ────────                                   ──────────
  // 接收（Go→JS）:监听 CustomEvent            Go→JS: stream-data / stream-end
  window.addEventListener('stream-data', e => {
    const { topic, data } = e.detail
  })

  // 控制（JS→Go）:通过 Bridge RPC              JS→Go: listenStream / stopStream / ...
  bridge.listenStream({ topic: 'xxx' })
  bridge.pauseReadmeStream()
  bridge.seekReadmeStream({ line: 10 })</CodeBlock>

    <!-- ========== Go 端基础设施 ========== -->
    <h3>Go 端基础设施（pkg/bridge/stream.go）</h3>

    <h4>核心类型</h4>
    <CodeBlock lang="go">// NewStream 创建一个具名数据通道，自动启动消费者 goroutine。
// 消费者从 channel 读取数据 → 通过 wv.Eval() 以 CustomEvent 形式推送到 JS。
//
// 参数:
//   b          *Bridge  桥接实例（持有 webview 引用）
//   topic      string   数据流主题（JS 端通过 stream-data 事件中的 topic 字段区分）
//   bufferSize int      通道缓冲区大小
//
// 返回:
//   chan<- interface{}  发送端，业务方向此通道写入数据即可推送到 JS
//
// 生命周期:
//   - close(ch)  → 自动发送 stream-end 事件，清理注册
//   - StopStream(topic) → 立即终止流
func NewStream(b *Bridge, topic string, bufferSize int) chan<- interface{}

// StopStream 停止一个活跃的数据流。
func StopStream(topic string) bool</CodeBlock>

    <h4>内置通用 Bridge 方法</h4>
    <p class="hint">以下方法自动注册，JS 可直接调用：</p>
    <CodeBlock lang="go">// 启动监听（创建空 stream，业务方后续通过 NewStream 获取 channel 写入数据）
func (b *Bridge) HandleListenStream(params)  // JS: bridge.listenStream({topic})
func (b *Bridge) HandleStopStream(params)    // JS: bridge.stopStream({topic})
func (b *Bridge) HandleSendToStream(params)  // JS: bridge.sendToStream({topic, data})
func (b *Bridge) HandleListStreams(params)   // JS: bridge.listStreams()</CodeBlock>

    <h4>推送数据到 JS（内部实现）</h4>
    <CodeBlock lang="go">// pushToJS 在 UI 线程执行 wv.Eval()，以 CustomEvent 形式传递数据。
// 消费者 goroutine 中调用，数据经 JSON 序列化后嵌入 JS 代码字符串。
func (b *Bridge) pushToJS(topic string, data interface{}) {
    payload := map[string]interface{}{"topic": topic, "data": data}
    jsonBytes, _ := json.Marshal(payload)
    js := fmt.Sprintf(
        `window.dispatchEvent(new CustomEvent('stream-data',{detail:%s}))`,
        string(jsonBytes),
    )
    b.wv.Dispatch(func() { b.wv.Eval(js) })  // ★ 必须在 UI 线程
}</CodeBlock>

    <!-- ========== JS 端 composable ========== -->
    <h3>JS 端 Composables</h3>

    <h4>useStream(topic) — 通用流式监听</h4>
    <p class="hint">
      封装了 <code>window.__lhpanda__</code> 调用 + <code>stream-data</code> / <code>stream-end</code> 事件监听，
      自动处理组件卸载时的资源清理。浏览器模式下自动降级为模拟数据。
    </p>
    <CodeBlock lang="javascript">// demo/src/composables/useStream.js
import { useStream } from '../composables/useStream'

const {
  data,       // Ref&lt;Array&gt;    所有已收到的数据（逐条追加）
  latest,     // Ref&lt;any&gt;      最新一条数据
  listening,  // Ref&lt;boolean&gt; 是否正在监听
  error,      // Ref&lt;string&gt;  错误信息
  listen,     // async () =&gt; void     开始监听
  stop,       // async () =&gt; void     停止监听
  send,       // async (payload) =&gt; void  从 JS 向流写入数据
} = useStream('my-topic')

// 用法
await listen()
// data.value = [{...}, {...}, ...]  ← 实时追加
// latest.value = {...}              ← 最新一条
await stop()</CodeBlock>

    <h4>useBridge() — Bridge RPC 调用</h4>
    <p class="hint">
      用于 JS→Go 的控制调用（启动流、暂停、跳转等）。Proxy 模式自动补默认参数。
    </p>
    <CodeBlock lang="javascript">const { bridge } = useBridge()

// 启动流（通用）
await bridge.listenStream({ topic: 'my-topic' })
// 停止流
await bridge.stopStream({ topic: 'my-topic' })
// JS 向流写数据（双向）
await bridge.sendToStream({ topic: 'my-topic', data: { msg: 'hello' } })
// 列出活跃流
const { streams } = await bridge.listStreams()</CodeBlock>

    <!-- ========== 快速开始 ========== -->
    <h3>快速开始：最简单的单向推送</h3>
    <p class="hint">只需 Go 端 5 行代码 + JS 端 3 行代码。</p>

    <CodeBlock lang="go">// pkg/bridge/myfeature.go — Go 端

func (b *Bridge) HandleMyDataStream(params map[string]interface{}) (interface{}, error) {
    ch := NewStream(b, "my-data", 64)       // 1. 创建数据流
    go func() {
        defer close(ch)                      // 2. 结束时关闭通道
        for _, item := range fetchData() {   // 3. 数据源
            ch <- item                       // 4. 写入通道 → 自动推送到 JS
            time.Sleep(500 * time.Millisecond)
        }
    }()
    return map[string]interface{}{"listening": true}, nil
}</CodeBlock>

    <CodeBlock lang="javascript">// my-component.vue — JS 端

import { useStream } from '../composables/useStream'
import { useBridge } from '../composables/useBridge'

const { bridge } = useBridge()
const { data, listen, stop } = useStream('my-data')

// 启动
await bridge.myDataStream()   // 调用 Go handler
await listen()                // 开始接收数据
// data.value 实时更新...

await stop()                  // 停止</CodeBlock>

    <!-- ========== 双向交互式控制 ========== -->
    <h3>进阶：双向交互式控制</h3>
    <p class="desc">
      Go 端通过<b>共享的 ctrl 结构体</b>接收 JS 的运行时控制指令（暂停/恢复/跳转/停止），
      goroutine 在每轮循环检查 ctrl 状态做出响应。
    </p>

    <h4>步骤 1：定义控制结构</h4>
    <CodeBlock lang="go">// 控制 goroutine 运行状态的共享结构
type myCtrl struct {
    mu     sync.Mutex    // 保护并发访问
    paused bool           // 暂停标志
    cancel chan struct{}  // 停止信号（close 即停止）
}

func (c *myCtrl) isPaused() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.paused }
func (c *myCtrl) setPaused(v bool) { c.mu.Lock(); c.paused = v; c.mu.Unlock() }</CodeBlock>

    <h4>步骤 2：写生产 goroutine（带控制检查）</h4>
    <CodeBlock lang="go">func runProducer(ch chan<- interface{}, ctrl *myCtrl, dataSource func() interface{}) {
    defer close(ch)
    for {
        // ── 暂停检查 ──
        for ctrl.isPaused() {
            select {
            case <-ctrl.cancel:
                return    // 暂停期间被取消
            case <-time.After(200 * time.Millisecond):
                // 轮询等待恢复
            }
        }

        // ── 生产数据并推送 ──
        select {
        case ch <- dataSource():
            // 间隔控制（可被取消打断）
            select {
            case <-time.After(time.Second):
            case <-ctrl.cancel:
                return
            }
        case <-ctrl.cancel:
            return
        }
    }
}</CodeBlock>

    <h4>步骤 3：写 Handler 方法（自动注册为 JS 可调用）</h4>
    <CodeBlock lang="go">var currentCtrl *myCtrl
var ctrlMu sync.Mutex

// 启动流 → JS: bridge.startMyStream()
func (b *Bridge) HandleStartMyStream(params map[string]interface{}) (interface{}, error) {
    ctrl := &myCtrl{cancel: make(chan struct{})}
    ctrlMu.Lock(); currentCtrl = ctrl; ctrlMu.Unlock()

    ch := NewStream(b, "my-topi", 64)
    go runProducer(ch, ctrl, myDataSource)
    return map[string]interface{}{"listening": true}, nil
}

// 暂停 → JS: bridge.pauseMyStream()
func (b *Bridge) HandlePauseMyStream(params map[string]interface{}) (interface{}, error) {
    if currentCtrl != nil { currentCtrl.setPaused(true) }
    return map[string]interface{}{"paused": true}, nil
}

// 恢复 → JS: bridge.resumeMyStream()
func (b *Bridge) HandleResumeMyStream(params map[string]interface{}) (interface{}, error) {
    if currentCtrl != nil { currentCtrl.setPaused(false) }
    return map[string]interface{}{"resumed": true}, nil
}

// 停止 → JS: bridge.stopMyStream()
func (b *Bridge) HandleStopMyStream(params map[string]interface{}) (interface{}, error) {
    StopStream("my-topic")
    if currentCtrl != nil { close(currentCtrl.cancel) }
    return map[string]interface{}{"stopped": true}, nil
}</CodeBlock>

    <!-- ========== JS 端监听事件 ========== -->
    <h3>JS 端监听事件</h3>

    <h4>stream-data 事件</h4>
    <CodeBlock lang="javascript">// Go 端每推送一条数据，JS 端触发一次 stream-data 事件
window.addEventListener('stream-data', (e) => {
  const { topic, data } = e.detail
  // topic: 数据流主题（如 "my-data"）
  // data:  业务数据（Go 端写入 channel 的原始值，经 JSON 透传）
})

// 使用 useStream 时自动处理，无需手动监听</CodeBlock>

    <h4>stream-end 事件</h4>
    <CodeBlock lang="javascript">// Go 端 close(ch) 时触发
window.addEventListener('stream-end', (e) => {
  const { topic } = e.detail
  // 流已结束，可做清理或展示"完成"提示
})</CodeBlock>

    <!-- ========== 数据流图 ========== -->
    <h3>完整数据流图</h3>
    <CodeBlock lang="text">时序图：双向交互式数据流

  JS (前端)                         Go (后端)
  ────────                         ────────
  │                                │
  │ bridge.startMyStream()         │
  │ ──── RPC ──────────────────→  │ HandleStartMyStream
  │                                │   NewStream(b, "my-topic")
  │ ← {listening:true} ────────   │   go runProducer(ch, ctrl)
  │                                │
  │ listen()  (useStream)          │    循环 {
  │                                │      检查 ctrl.isPaused()
  │                                │      ch ← data
  │ ←── CustomEvent ────────────  │      pushToJS → wv.Eval()
  │     stream-data                │    }
  │     {topic:"my-topic", data}   │
  │                                │
  │ bridge.pauseMyStream()         │
  │ ──── RPC ──────────────────→  │ ctrl.setPaused(true)
  │                                │    goroutine 暂停
  │                                │
  │ bridge.resumeMyStream()        │
  │ ──── RPC ──────────────────→  │ ctrl.setPaused(false)
  │                                │    goroutine 恢复
  │ ←── CustomEvent ────────────  │     继续推送
  │                                │
  │ bridge.stopMyStream()          │
  │ ──── RPC ──────────────────→  │ close(ctrl.cancel)
  │                                │    defer close(ch)
  │ ←── CustomEvent ────────────  │ ← stream-end 事件
  │     stream-end                 │</CodeBlock>

    <!-- ========== 注意事项 ========== -->
    <h3>注意事项</h3>
    <ul class="note-list">
      <li><b>Eval 必须在 UI 线程</b> — 消费者 goroutine 调用 <code>pushToJS</code> 内部通过 <code>wv.Dispatch()</code> 调度到 UI 线程执行 JS。</li>
      <li><b>JSON 序列化限制</b> — channel 中传递的数据必须可 JSON 序列化（Go 的 <code>map[string]interface{}</code>、<code>string</code>、<code>float64</code>、<code>bool</code> 等）。Channel、函数指针等不可传递。</li>
      <li><b>RPC 调用需两个参数</b> — <code>window.__lhpanda__</code> 底层 Bind 严格校验参数数量。始终传第二个参数（空对象 <code>{}</code> 或带参数对象）。使用 <code>useBridge()</code> 的 Proxy 会自动补 <code>{}</code>。</li>
      <li><b>并发安全</b> — ctrl 结构体的字段需加锁保护（<code>sync.Mutex</code>），因为 handler 方法和 goroutine 在不同 goroutine 中访问。</li>
      <li><b>同一 topic 互斥</b> — 同一 topic 同时只允许一个活跃流。启动新流前应先停止旧流。</li>
      <li><b>浏览器降级</b> — <code>useStream</code> 在浏览器模式下自动使用模拟数据，无需特殊处理。</li>
    </ul>
  </div>
</template>

<script setup>
import { Bell } from '@element-plus/icons-vue'
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.desc code, .desc b { color: var(--accent); }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; line-height: 1.6; }
.hint code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }

.note-list { color: var(--text-secondary); font-size: 13px; line-height: 1.8; padding-left: 20px; }
.note-list li { margin-bottom: 6px; }
.note-list b { color: var(--text); }
</style>
