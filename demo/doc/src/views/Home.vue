<template>
  <div>
    <header class="hero">
      <div class="logo"><el-icon :size="56"><Monitor /></el-icon></div>
      <h1>WebToDesktop</h1>
      <p class="subtitle">将任意 Vue 前端项目打包为桌面 EXE 程序</p>
      <div class="badges">
        <span class="badge">Go</span>
        <span class="badge">Vue 3</span>
        <span class="badge">WebView2</span>
        <span class="badge">WebKitGTK</span>
        <span class="badge">跨平台</span>
      </div>
    </header>

    <section class="status-bar">
      <div class="status-item">
        <span class="status-label">运行平台</span>
        <span class="status-value">{{ appInfo.platform }}/{{ appInfo.arch }}</span>
      </div>
      <div class="status-item">
        <span class="status-label">桥接状态</span>
        <span class="status-value" :class="bridgeReady ? 'ok' : 'warn'">
          <template v-if="bridgeReady"><el-icon><CircleCheckFilled /></el-icon> Go↔JS 已连接</template>
          <template v-else><el-icon><WarningFilled /></el-icon> 浏览器模式（模拟数据）</template>
        </span>
      </div>
      <div class="status-item">
        <span class="status-label">窗口模式</span>
        <span class="status-value">{{ isDesktop ? '桌面原生窗口' : '浏览器标签页' }}</span>
      </div>
      <div class="status-item">
        <span class="status-label">透明背景</span>
        <span class="status-value" :class="winCfg.webview_bg_transparent ? 'ok' : ''">
          <template v-if="winCfg.webview_bg_transparent"><el-icon><CircleCheckFilled /></el-icon> 已启用</template>
          <template v-else>未启用</template>
        </span>
      </div>
      <div v-if="appInfo.platform === 'linux'" class="status-item">
        <span class="status-label">点击穿透</span>
        <span class="status-value" :class="winCfg.input_passthrough ? 'warn' : ''">
          <template v-if="winCfg.input_passthrough"><el-icon><WarningFilled /></el-icon> 已开启</template>
          <template v-else>已禁用</template>
        </span>
      </div>
    </section>

    <div class="demo-grid">
      <section class="card">
        <h2><el-icon><Lock /></el-icon> 凭证加密存储</h2>
        <p class="card-desc">AES-256 加密保存用户密码，密码永不暴露到前端</p>

        <div class="form-group">
          <label>用户名</label>
          <input v-model="credForm.username" placeholder="输入用户名" />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="credForm.password" type="password" placeholder="输入密码" />
        </div>
        <div class="btn-row">
          <button class="btn primary" @click="saveCred" :disabled="!credForm.username||!credForm.password"><el-icon><UploadFilled /></el-icon> 保存凭据</button>
          <button class="btn" @click="loadCreds"><el-icon><Tickets /></el-icon> 刷新列表</button>
          <button class="btn danger" @click="clearCreds" :disabled="credList.length===0"><el-icon><Delete /></el-icon> 清除全部</button>
        </div>

        <div v-if="credList.length" class="cred-list">
          <div class="cred-item" v-for="c in credList" :key="c.username">
            <span><el-icon><User /></el-icon> {{ c.username }}</span>
            <button class="btn-sm danger" @click="deleteCred(c.username)">删除</button>
          </div>
        </div>
        <p v-else class="empty-hint">暂无已保存的凭据</p>

        <div class="tip-box">
          <el-icon><InfoFilled /></el-icon> <strong>工作原理</strong>：登录时勾选"记住密码"→调用 <code>saveCredentials</code>→AES 加密存入本地。
          下次打开页面时，请求头自动携带 <code>X-Credential-Username</code>，代理层将请求体中的
          <code>__DESKTOP_PWD__</code> 替换为真实密码后转发到远程 API。
        </div>
      </section>

      <section class="card">
        <h2><el-icon><Connection /></el-icon> 智能网络代理</h2>
        <div class="proxy-flow">
          <div class="flow-step"><div class="flow-icon"><el-icon><Monitor /></el-icon></div><div class="flow-label">前端请求</div><div class="flow-detail">fetch('/api/users')</div></div>
          <div class="flow-arrow">→</div>
          <div class="flow-step"><div class="flow-icon"><el-icon><Sort /></el-icon></div><div class="flow-label">代理层匹配</div><div class="flow-detail">匹配前缀 /api/</div></div>
          <div class="flow-arrow">→</div>
          <div class="flow-step"><div class="flow-icon"><el-icon><Key /></el-icon></div><div class="flow-label">凭证注入</div><div class="flow-detail">替换 __DESKTOP_PWD__</div></div>
          <div class="flow-arrow">→</div>
          <div class="flow-step"><div class="flow-icon"><el-icon><Cloudy /></el-icon></div><div class="flow-label">远程 API</div><div class="flow-detail">https://api.example.com</div></div>
        </div>

        <div class="proxy-rules">
          <h3>路由规则（4 级优先级）</h3>
          <div class="rule"><span class="rule-num">1</span> 匹配代理前缀 → 转发到远程 API + 凭证注入</div>
          <div class="rule"><span class="rule-num">2</span> 匹配 /storage/ → 代理到远程 + 凭证注入</div>
          <div class="rule"><span class="rule-num">3</span> 本地静态资源 → 直接返回嵌入的 Vue 文件</div>
          <div class="rule"><span class="rule-num">4</span> SPA fallback → 返回 index.html</div>
        </div>

        <div class="form-group" style="margin-top:16px">
          <label>模拟 API 请求测试</label>
          <div class="input-row">
            <input v-model="apiPath" placeholder="/api/test" style="flex:1" />
            <button class="btn primary" @click="testProxy">发送请求</button>
          </div>
        </div>
        <div v-if="proxyResult" class="code-block">{{ proxyResult }}</div>
      </section>

      <section class="card">
        <h2><el-icon><Monitor /></el-icon> 窗口控制</h2>
        <p class="card-desc">无边框窗口下的交互演示（桌面模式生效）</p>
        <div class="btn-row">
          <button class="btn" @click="dragWindow"><el-icon><Pointer /></el-icon> 拖拽窗口</button>
          <button class="btn" @click="toggleMaximize"><el-icon><Crop /></el-icon> 最大化/还原</button>
          <button class="btn" @click="toggleFullscreen"><el-icon><FullScreen /></el-icon> 全屏切换</button>
          <button class="btn danger" @click="closeWindow"><el-icon><Close /></el-icon> 关闭窗口</button>
        </div>
        <div class="tip-box" style="margin-top:12px">
          <el-icon><InfoFilled /></el-icon> 顶部区域可拖拽移动窗口。边缘 6px 范围内可拖拽调整窗口大小。
          <strong>桌面模式下</strong>点击按钮实际控制窗口，<strong>浏览器模式下</strong>仅显示提示。
        </div>
      </section>

      <section class="card">
        <h2><el-icon><Bell /></el-icon> Go→JS 流式数据推送</h2>
        <p class="card-desc">
          Go goroutine 每秒逐行读取 <code>README.md</code>，
          通过 <code>channel → wv.Eval() → CustomEvent</code> 推送到前端实时展示。
        </p>
        <div class="btn-row">
          <button class="btn primary" @click="startReadmeStream" :disabled="readmeListening">
            <el-icon><VideoPlay /></el-icon> 开始推送
          </button>
          <button class="btn" @click="pauseReadmeStream" :disabled="!readmeListening || readmePaused">
            <el-icon><VideoPause /></el-icon> 暂停
          </button>
          <button class="btn" @click="resumeReadmeStream" :disabled="!readmePaused">
            <el-icon><VideoPlay /></el-icon> 恢复
          </button>
          <button class="btn danger" @click="stopReadmeStream" :disabled="!readmeListening">
            <el-icon><Close /></el-icon> 停止
          </button>
        </div>
        <div class="btn-row" style="margin-top:0">
          <label style="font-size:13px;color:var(--text-secondary);display:flex;align-items:center;gap:6px">
            从第 <input v-model="readmeStartLine" type="number" min="1"
              style="width:60px;padding:4px 8px;border-radius:4px;border:1px solid var(--border);text-align:center" />
            行开始
          </label>
          <button class="btn" @click="seekReadmeStream" :disabled="!readmeListening">
            <el-icon><Sort /></el-icon> 跳转
          </button>
          <span v-if="readmeTotal" class="readme-progress">
            进度: {{ readmeLines.length }} / {{ readmeTotal }} 行
          </span>
        </div>
        <div v-if="readmeError" class="error-hint">
          <el-icon><WarningFilled /></el-icon> {{ readmeError }}
        </div>
        <div v-if="readmeDone" class="done-hint">
          <el-icon><CircleCheckFilled /></el-icon> README.md 推送完毕
        </div>
        <div v-if="readmeLines.length" class="readme-viewer">
          <div class="readme-line" v-for="(item, i) in readmeLines" :key="i">
            <span class="line-no">{{ item.line }}</span>
            <span class="line-text">{{ item.text }}</span>
          </div>
        </div>
        <p v-if="!readmeListening && !readmeLines.length" class="empty-hint">
          点击"开始推送"查看 README.md 逐行展示效果
        </p>
        <details class="tip-box" style="margin-top:12px">
          <summary><el-icon><InfoFilled /></el-icon> <strong>架构说明</strong></summary>
          <pre style="white-space:pre-wrap;font-size:13px;margin-top:8px">
Go 侧（streamdemo.go）:
  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
      ch ← {line, text, total}
      time.Sleep(1s)  // 每秒一行
  }

JS 侧:
  window.addEventListener('stream-data', e => {
      // e.detail = {topic, data: {line, text, ...}}
  })
          </pre>
        </details>
      </section>

      <section class="card">
        <h2><el-icon><Connection /></el-icon> 串口双向数据流</h2>
        <p class="card-desc">
          扫描可用串口 → 配置参数打开 → 实时接收数据 → 双向发送。
          Go 端通过 <code>go.bug.st/serial</code> 库读写串口，数据经 stream 通道推送到前端。
        </p>

        <!-- 扫描串口 -->
        <div class="btn-row">
          <button class="btn primary" @click="scanSerialPorts">
            <el-icon><Search /></el-icon> 扫描串口
          </button>
          <span v-if="serialPorts.length" class="readme-progress">
            找到 {{ serialPorts.length }} 个串口
          </span>
        </div>

        <!-- 串口列表 + 参数配置 -->
        <div v-if="serialPorts.length && !serialOpened" class="serial-config">
          <div class="form-group">
            <label>选择串口</label>
            <select v-model="serialSelectedPort" style="width:100%;padding:8px;border-radius:6px;border:1px solid var(--border)">
              <option v-for="p in serialPorts" :key="p.name" :value="p.name">{{ p.name }}</option>
            </select>
          </div>
          <div class="serial-params">
            <div class="form-group" style="flex:1">
              <label>波特率</label>
              <select v-model="serialConfig.baudRate" style="width:100%;padding:8px;border-radius:6px;border:1px solid var(--border)">
                <option :value="9600">9600</option>
                <option :value="19200">19200</option>
                <option :value="38400">38400</option>
                <option :value="57600">57600</option>
                <option :value="115200">115200</option>
                <option :value="230400">230400</option>
                <option :value="460800">460800</option>
                <option :value="921600">921600</option>
              </select>
            </div>
            <div class="form-group" style="flex:1">
              <label>数据位</label>
              <select v-model="serialConfig.dataBits" style="width:100%;padding:8px;border-radius:6px;border:1px solid var(--border)">
                <option :value="5">5</option>
                <option :value="6">6</option>
                <option :value="7">7</option>
                <option :value="8">8</option>
              </select>
            </div>
            <div class="form-group" style="flex:1">
              <label>校验</label>
              <select v-model="serialConfig.parity" style="width:100%;padding:8px;border-radius:6px;border:1px solid var(--border)">
                <option value="none">无</option>
                <option value="odd">奇校验</option>
                <option value="even">偶校验</option>
              </select>
            </div>
            <div class="form-group" style="flex:1">
              <label>停止位</label>
              <select v-model="serialConfig.stopBits" style="width:100%;padding:8px;border-radius:6px;border:1px solid var(--border)">
                <option :value="1">1</option>
                <option :value="2">2</option>
              </select>
            </div>
          </div>
          <button class="btn primary" @click="openSerialPort" style="margin-top:8px">
            <el-icon><VideoPlay /></el-icon> 打开串口
          </button>
        </div>

        <!-- 已打开串口：数据面板 -->
        <div v-if="serialOpened">
          <div class="btn-row">
            <button class="btn danger" @click="closeSerialPort">
              <el-icon><Close /></el-icon> 关闭串口
            </button>
            <span class="readme-progress" v-if="serialPortInfo">
              {{ serialPortInfo.port }} @ {{ serialPortInfo.baudRate }}-{{ serialPortInfo.dataBits }}{{ serialPortInfo.parity === 'none' ? 'N' : serialPortInfo.parity[0].toUpperCase() }}{{ serialPortInfo.stopBits }}
            </span>
          </div>

          <!-- 发送区域 -->
          <div class="input-row" style="margin:8px 0">
            <input v-model="serialSendText" placeholder="输入要发送的数据..." style="flex:1"
              @keyup.enter="sendSerialData" />
            <button class="btn primary" @click="sendSerialData">
              <el-icon><Promotion /></el-icon> 发送
            </button>
          </div>

          <div v-if="serialError" class="error-hint">
            <el-icon><WarningFilled /></el-icon> {{ serialError }}
          </div>

          <!-- 数据终端 -->
          <div class="serial-terminal" v-if="serialData.length">
            <div class="serial-line" v-for="(item, i) in serialData" :key="i"
              :class="{ 'serial-sent': item.sent }">
              <span class="serial-time">{{ new Date(item.timestamp).toLocaleTimeString() }}</span>
              <span class="serial-hex" v-if="item.hex">{{ item.hex }}</span>
              <span class="serial-text" :class="{ 'serial-sent-text': item.sent }">{{ item.text }}</span>
            </div>
          </div>
          <p v-else class="empty-hint">等待接收数据...</p>
        </div>

        <p v-if="!serialOpened && !serialPorts.length" class="empty-hint">
          点击"扫描串口"发现可用设备
        </p>
      </section>

      <section class="card">
        <h2><el-icon><Key /></el-icon> 键盘快捷键测试</h2>
        <p class="card-desc">注册快捷键后按下，看看能否被拦截并收到 <code>keyboard-shortcut</code> 事件。</p>

        <div class="btn-row">
          <button class="btn primary" @click="regShortcut('Ctrl+P')">
            <el-icon><Plus /></el-icon> 注册 Ctrl+P
          </button>
          <button class="btn primary" @click="regShortcut('Shift+F1')">
            <el-icon><Plus /></el-icon> 注册 Shift+F1
          </button>
          <button class="btn primary" @click="regShortcut('Ctrl+Shift+S')">
            <el-icon><Plus /></el-icon> 注册 Ctrl+Shift+S
          </button>
          <button class="btn danger" @click="clearAllShortcuts">
            <el-icon><Delete /></el-icon> 清空所有
          </button>
        </div>

        <div class="btn-row" style="margin-top:4px">
          <input v-model="customKey" placeholder="输入自定义快捷键，如 Alt+F2"
            style="flex:1;padding:8px 12px;border-radius:6px;border:1px solid var(--border)" />
          <button class="btn primary" @click="regShortcut(customKey)" :disabled="!customKey.trim()">
            <el-icon><Plus /></el-icon> 注册
          </button>
        </div>

        <div v-if="registeredKeys.length" class="shortcut-list" style="margin:8px 0">
          <span v-for="k in registeredKeys" :key="k" class="shortcut-tag">{{ k }}</span>
        </div>

        <div class="kb-event-log" ref="kbLogRef">
          <div class="kb-log-header">
            <span>拦截事件日志</span>
            <button class="btn-sm" @click="kbEvents=[]" v-if="kbEvents.length">清空</button>
          </div>
          <div v-if="kbEvents.length" class="kb-log-entries">
            <div v-for="(ev, i) in kbEvents" :key="i" class="kb-log-entry">
              <span class="kb-log-time">{{ ev.time }}</span>
              <span class="kb-log-key">{{ ev.key }}</span>
            </div>
          </div>
          <p v-else class="empty-hint" style="margin:8px 0">尚无拦截事件，注册快捷键后按下测试</p>
        </div>
      </section>

      <section class="card">
        <h2><el-icon><Box /></el-icon> 构建命令</h2>
        <div class="commands">
          <div class="cmd" v-for="c in commands" :key="c.cmd">
            <code class="cmd-code">{{ c.cmd }}</code>
            <span class="cmd-desc">{{ c.desc }}</span>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { inject, ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { useSerial } from '../composables/useSerial'
import { Monitor, CircleCheckFilled, WarningFilled, Lock, Connection, Delete, Key, User, Sort, Cloudy, Box, Tickets, Crop, FullScreen, Close, Pointer, InfoFilled, UploadFilled, Bell, VideoPlay, VideoPause, Search, Promotion, Plus } from '@element-plus/icons-vue'

const bridge = inject('bridge', {})
const appInfo = inject('appInfo', ref({ platform: 'browser', arch: 'x64' }))
const bridgeReady = inject('bridgeReady', ref(false))
const isDesktop = inject('isDesktop', ref(false))
const winCfg = reactive({ webview_bg_transparent: false, input_passthrough: false })
const credList = inject('credList', ref([]))
const credForm = inject('credForm', reactive({ username: '', password: '' }))
const saveCredFn = inject('saveCred', () => {})
const loadCredsFn = inject('loadCreds', () => {})
const deleteCredFn = inject('deleteCred', () => {})
const clearCredsFn = inject('clearCreds', () => {})

const apiPath = ref('/api/test')
const proxyResult = ref('')

const commands = [
  { cmd: 'make', desc: '交互式选择项目 + 平台' },
  { cmd: 'make run PROJECT=xxx', desc: '构建并运行（当前平台调试）' },
  { cmd: 'make build-windows PROJECT=xxx', desc: '交叉编译 Windows EXE（含图标）' },
  { cmd: 'make build-linux PROJECT=xxx', desc: '编译 Linux 桌面版' },
  { cmd: 'make dev PROJECT=xxx', desc: '仅启动 Vue 开发服务器' },
  { cmd: 'make list', desc: '列出所有可构建项目' },
  { cmd: 'make clean', desc: '清理构建产物' }
]

// ———— README 逐行流式推送演示 ————
const readmeLines = ref([])
const readmeListening = ref(false)
const readmePaused = ref(false)
const readmeTotal = ref(0)
const readmeError = ref('')
const readmeDone = ref(false)
const readmeStartLine = ref(1)

function onStreamData(e) {
  const { topic, data: payload } = e.detail || {}
  if (topic !== 'readme-stream') return

  if (payload.type === 'line') {
    readmeLines.value.push(payload)
    readmeTotal.value = payload.total
    readmePaused.value = false
  } else if (payload.type === 'done') {
    readmeDone.value = true
    readmeListening.value = false
    readmePaused.value = false
  } else if (payload.type === 'error') {
    readmeError.value = payload.msg
    readmeListening.value = false
  } else if (payload.type === 'seek') {
    // 跳转成功，清空已有行并从跳转行开始
    readmeLines.value = []
    readmeDone.value = false
    readmePaused.value = false
  }
}

function onStreamEnd(e) {
  const { topic } = e.detail || {}
  if (topic === 'readme-stream') {
    readmeListening.value = false
  }
}

async function startReadmeStream() {
  readmeLines.value = []
  readmeTotal.value = 0
  readmeError.value = ''
  readmeDone.value = false
  readmePaused.value = false

  if (window.__lhpanda__) {
    try {
      const result = await window.__lhpanda__('startReadmeStream', {
        startLine: readmeStartLine.value,
        speed: 800,
      })
      if (result.success) {
        readmeListening.value = true
        readmeTotal.value = result.data.totalLines || 0
      }
    } catch (err) {
      readmeError.value = err?.message || String(err)
    }
  } else {
    readmeListening.value = true
    mockReadmeStream(readmeStartLine.value)
  }
}

async function stopReadmeStream() {
  if (window.__lhpanda__) {
    try { await window.__lhpanda__('stopReadmeStream', {}) } catch {}
  }
  readmeListening.value = false
  readmePaused.value = false
}

async function pauseReadmeStream() {
  if (window.__lhpanda__) {
    try {
      await window.__lhpanda__('pauseReadmeStream', {})
      readmePaused.value = true
    } catch {}
  } else {
    readmePaused.value = true
  }
}

async function resumeReadmeStream() {
  if (window.__lhpanda__) {
    try {
      await window.__lhpanda__('resumeReadmeStream', {})
      readmePaused.value = false
    } catch {}
  } else {
    readmePaused.value = false
    // 浏览器模拟：重新 mock 剩下的行
    mockReadmeStream(readmeLines.value.length + 1)
  }
}

async function seekReadmeStream() {
  if (window.__lhpanda__) {
    try {
      await window.__lhpanda__('seekReadmeStream', { line: readmeStartLine.value })
      readmeLines.value = []
      readmeDone.value = false
      readmePaused.value = false
    } catch (err) {
      readmeError.value = err?.message || String(err)
    }
  } else {
    // 浏览器模拟：清空并重新开始
    readmeLines.value = []
    readmeDone.value = false
    readmeListening.value = true
    mockReadmeStream(readmeStartLine.value)
  }
}

// ———— 键盘快捷键测试 ————
const kbEvents = ref([])
const registeredKeys = ref([])
const customKey = ref('')
const kbLogRef = ref(null)

function handleKeyboardShortcut(e) {
  const { key } = e.detail || {}
  if (!key) return
  const now = new Date()
  const time = now.toLocaleTimeString() + '.' + String(now.getMilliseconds()).padStart(3, '0')
  kbEvents.value.unshift({ time, key })
  if (kbEvents.value.length > 50) kbEvents.value.length = 50
  nextTick(() => {
    if (kbLogRef.value) {
      const el = kbLogRef.value.querySelector('.kb-log-entries')
      if (el) el.scrollTop = 0
    }
  })
}

async function regShortcut(key) {
  if (!key || !key.trim()) return
  key = key.trim()
  if (!window.__lhpanda__) { alert('当前为浏览器模式，快捷键拦截仅在桌面模式生效'); return }
  try {
    await bridge.registerShortcut({ keys: [key] })
    const list = await bridge.listShortcuts()
    registeredKeys.value = list?.shortcuts || []
  } catch (e) {
    console.error('注册失败:', e)
  }
}

async function clearAllShortcuts() {
  if (!window.__lhpanda__) return
  try {
    await bridge.clearShortcuts()
    registeredKeys.value = []
  } catch (e) {
    console.error('清除失败:', e)
  }
}

onMounted(() => {
  window.addEventListener('keyboard-shortcut', handleKeyboardShortcut)
  // 加载已注册的快捷键
  if (window.__lhpanda__) {
    bridge.listShortcuts().then(list => {
      registeredKeys.value = list?.shortcuts || []
    }).catch(() => {})
  }
})

onUnmounted(() => {
  window.removeEventListener('keyboard-shortcut', handleKeyboardShortcut)
})

// 浏览器模拟：用预置文本模拟逐行推送
function mockReadmeStream(startLine) {
  const allLines = [
    '# WebToDesktop',
    '',
    '将任意前端项目打包为桌面应用。',
    'Go + 系统原生 WebView，零运行时依赖。',
    '',
    '## 平台支持',
    '| 平台 | 后端 |',
    '|------|------|',
    '| Linux | GTK + WebKit2GTK |',
    '| Windows | Edge WebView2 |',
    '',
    '> （浏览器模拟模式，共 12 行）',
  ]
  readmeTotal.value = allLines.length
  let i = Math.max(0, (startLine || 1) - 1)
  const timer = setInterval(() => {
    if (!readmePaused.value && i >= allLines.length) {
      clearInterval(timer)
      readmeDone.value = true
      readmeListening.value = false
      return
    }
    if (readmePaused.value) return
    if (i >= allLines.length) { clearInterval(timer); return }
    readmeLines.value.push({ line: i + 1, total: allLines.length, text: allLines[i] })
    i++
  }, 800)
}

// ———— 串口双向数据流 ————
const {
  ports: serialPorts,
  opened: serialOpened,
  portInfo: serialPortInfo,
  data: serialData,
  error: serialError,
  listPorts: scanSerialPorts,
  open: openSerialPortFn,
  close: closeSerialPortFn,
  send: sendSerialDataFn,
} = useSerial()

const serialSelectedPort = ref('')
const serialSendText = ref('')
const serialConfig = reactive({
  baudRate: 115200,
  dataBits: 8,
  parity: 'none',
  stopBits: 1,
})

async function openSerialPort() {
  await openSerialPortFn(serialSelectedPort.value, { ...serialConfig })
}

async function closeSerialPort() {
  await closeSerialPortFn()
}

async function sendSerialData() {
  if (!serialSendText.value) return
  await sendSerialDataFn(serialSendText.value)
  serialSendText.value = ''
}

onMounted(async () => {
  window.addEventListener('stream-data', onStreamData)
  window.addEventListener('stream-end', onStreamEnd)
  try {
    const cfg = await bridge.getWindowConfig()
    if (cfg) {
      winCfg.webview_bg_transparent = cfg.webview_bg_transparent || cfg.acrylic
      winCfg.input_passthrough = cfg.input_passthrough
    }
  } catch (e) { /* browser mode */ }
})

onUnmounted(() => {
  window.removeEventListener('stream-data', onStreamData)
  window.removeEventListener('stream-end', onStreamEnd)
})

async function saveCred() { saveCredFn() }
async function loadCreds() { loadCredsFn() }
async function deleteCred(username) { deleteCredFn(username) }
async function clearCreds() { clearCredsFn() }
async function dragWindow() { bridge.dragWindow().catch(() => {}) }
async function toggleMaximize() { bridge.toggleMaximize().catch(() => {}) }
async function toggleFullscreen() { bridge.toggleFullscreen().catch(() => {}) }
async function closeWindow() { bridge.closeWindow().catch(() => {}) }

// ———— 串口相关 ————
async function testProxy() {
  if (!bridgeReady.value) {
    proxyResult.value = '浏览器模式下代理不可用\n\n请求路径: ' + apiPath.value + '\n代理前缀配置: /api/, /storage/\n\n实际在桌面端运行时会:\n1. 匹配代理前缀 → 转发到远程 API\n2. 注入 X-Credential-Username 请求头\n3. 替换 __DESKTOP_PWD__ 占位符\n4. 返回远程 API 的响应'
    return
  }
  try {
    const resp = await fetch(apiPath.value)
    proxyResult.value = '状态: ' + resp.status + '\n\n' + (await resp.text()).substring(0, 500)
  } catch (err) { proxyResult.value = '请求失败: ' + err.message }
}
</script>

<style scoped>
.shortcut-list { display: flex; flex-wrap: wrap; gap: 6px; }
.shortcut-tag {
  display: inline-block; font-size: 11px; font-family: monospace;
  padding: 2px 8px; border-radius: 4px;
  background: var(--accent-dim); color: var(--accent); border: 1px solid transparent;
}

.kb-event-log {
  border: 1px solid var(--border); border-radius: 6px; overflow: hidden;
  font-size: 13px;
}
.kb-log-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 10px; background: #00000006; border-bottom: 1px solid var(--border);
  font-size: 12px; color: var(--text-secondary);
}
.kb-log-entries {
  max-height: 200px; overflow-y: auto; padding: 4px 0;
}
.kb-log-entry {
  display: flex; gap: 12px; padding: 3px 10px;
  font-family: monospace; font-size: 12px;
}
.kb-log-entry:hover { background: #00000008; }
.kb-log-time { color: var(--text-secondary); flex-shrink: 0; }
.kb-log-key { color: var(--accent); font-weight: 500; }
</style>
