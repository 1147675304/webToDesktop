<template>
  <div class="app">
    <!-- 顶部标题栏 -->
    <header class="titlebar" @mousedown="onDragStart">
      <span class="titlebar-icon">⚡</span>
      <span class="titlebar-text">串口调试工具</span>
      <span class="titlebar-badge" v-if="isDesktop">桌面模式</span>
      <span class="titlebar-badge warn" v-else>浏览器模拟</span>
      <div class="titlebar-spacer"></div>
      <button class="titlebar-btn no-drag" @click.stop="toggleMax" title="最大化">🗖</button>
      <button class="titlebar-btn close no-drag" @click.stop="closeWindow" title="关闭">✕</button>
    </header>

    <div class="main">
      <!-- 左侧面板 -->
      <aside class="sidebar">
        <!-- 串口列表 -->
        <div class="panel">
          <h3>串口列表</h3>
          <button class="btn scan" @click="doScan" :disabled="scanning">
            {{ scanning ? '扫描中...' : '🔍 扫描串口' }}
          </button>
          <div v-if="ports.length" class="port-list">
            <div v-for="p in ports" :key="p.name" class="port-item"
              :class="{ active: selectedPort === p.name }"
              @click="selectedPort = p.name">
              {{ p.name }}
              <span class="port-status" v-if="opened && portInfo?.port === p.name">●</span>
            </div>
          </div>
          <p v-else class="dim">点击扫描发现串口设备</p>
        </div>

        <!-- 串口参数 -->
        <div class="panel" v-if="ports.length">
          <h3>串口参数</h3>
          <div class="form-row"><label>波特率</label><select v-model="cfg.baudRate">
            <option :value="9600">9600</option><option :value="19200">19200</option>
            <option :value="38400">38400</option><option :value="57600">57600</option>
            <option :value="115200">115200</option><option :value="230400">230400</option>
            <option :value="460800">460800</option><option :value="921600">921600</option>
          </select></div>
          <div class="form-row"><label>数据位</label><select v-model="cfg.dataBits">
            <option :value="5">5</option><option :value="6">6</option>
            <option :value="7">7</option><option :value="8">8</option>
          </select></div>
          <div class="form-row"><label>校验位</label><select v-model="cfg.parity">
            <option value="none">None</option><option value="odd">Odd</option>
            <option value="even">Even</option><option value="mark">Mark</option><option value="space">Space</option>
          </select></div>
          <div class="form-row"><label>停止位</label><select v-model="cfg.stopBits">
            <option :value="1">1</option><option :value="2">2</option>
          </select></div>

          <button class="btn open" :class="{ danger: opened }"
            @click="opened ? doClose() : doOpen()" :disabled="!selectedPort || scanning">
            {{ opened ? '⏹ 关闭串口' : '▶ 打开串口' }}
          </button>

          <div class="conn-info" v-if="opened && portInfo">
            <span class="conn-dot"></span>
            {{ portInfo.port }} @ {{ portInfo.baudRate }}-{{ portInfo.dataBits }}{{ parityShort }}{{ portInfo.stopBits }}
          </div>
        </div>

        <!-- 硬件控制 -->
        <div class="panel" v-if="opened">
          <h3>硬件控制</h3>
          <div class="hw-controls">
            <button class="hw-btn" :class="{ on: dtrOn }" @click="toggleDtr">DTR</button>
            <button class="hw-btn" :class="{ on: rtsOn }" @click="toggleRts">RTS</button>
            <span class="modem-bits">
              <span class="modem-bit" :class="{ on: modemStatus.cts }" title="CTS">CTS</span>
              <span class="modem-bit" :class="{ on: modemStatus.dsr }" title="DSR">DSR</span>
              <span class="modem-bit" :class="{ on: modemStatus.dcd }" title="DCD">DCD</span>
              <span class="modem-bit" :class="{ on: modemStatus.ri }" title="RI">RI</span>
            </span>
          </div>
        </div>

        <!-- 网络桥接 -->
        <div class="panel" v-if="opened">
          <h3>网络桥接</h3>
          <div class="form-row"><label>模式</label><select v-model="bridgeMode">
            <option value="">— 关闭 —</option>
            <option value="tcp-server">TCP Server</option>
            <option value="tcp-client">TCP Client</option>
            <option value="udp">UDP</option>
          </select></div>
          <div class="form-row" v-if="bridgeMode === 'tcp-client'"><label>主机</label><input v-model="bridgeHost" class="finput" placeholder="192.168.1.1" /></div>
          <div class="form-row" v-if="bridgeMode"><label>端口</label><input v-model.number="bridgePort" class="finput" type="number" min="1" max="65535" /></div>
          <div class="form-row" v-if="bridgeMode === 'udp'"><label>远端</label><input v-model="bridgeRemote" class="finput" placeholder="host:port（可选）" /></div>
          <button class="btn scan" @click="bridgeActive ? doStopBridge() : doStartBridge()"
            :class="{ danger: bridgeActive }">
            {{ bridgeActive ? '⏹ 停止桥接' : '▶ 启动桥接' }}
          </button>
          <div class="conn-info" v-if="bridgeStatus" style="background:var(--bg2);color:var(--text)">{{ bridgeStatus }}</div>
        </div>

        <!-- 预设指令库 -->
        <div class="panel" v-if="opened">
          <h3>预设指令 <button class="btn-icon" @click="addPreset" title="添加当前输入为预设">＋</button></h3>
          <div class="preset-list" v-if="presets.length">
            <div v-for="(p, i) in presets" :key="i" class="preset-item">
              <span class="preset-name" :title="p.text">{{ p.label || p.text.slice(0,20) }}</span>
              <span class="preset-actions">
                <button class="btn-icon" @click="sendPreset(p)" title="发送">▶</button>
                <button class="btn-icon" @click="editPreset(i)" title="编辑">✎</button>
                <button class="btn-icon danger" @click="presets.splice(i,1);savePresets()" title="删除">✕</button>
              </span>
            </div>
          </div>
          <p v-else class="dim">点击 ＋ 保存当前输入为预设指令</p>
        </div>
      </aside>

      <!-- 右侧终端 -->
      <section class="terminal-section">
        <div class="terminal-header">
          <span>终端输出</span>
          <div class="terminal-actions">
            <label class="toggle-label"><input type="checkbox" v-model="showHex" /> HEX</label>
            <label class="toggle-label"><input type="checkbox" v-model="showTimestamp" /> 时间戳</label>
            <label class="toggle-label"><input type="checkbox" v-model="autoScroll" /> 自动滚动</label>
            <label class="toggle-label"><input type="checkbox" v-model="showAsciiOnly" /> 仅ASCII</label>
            <button class="btn-sm" @click="saveLog">💾 保存日志</button>
            <button class="btn-sm" @click="clearTerminal">清空</button>
          </div>
        </div>

        <div class="terminal-body" ref="termRef">
          <div v-if="!terminalData.length" class="dim center">等待接收数据...</div>
          <div v-for="(item, i) in terminalData" :key="i" class="term-line" :class="{ 'term-sent': item.sent }">
            <span v-if="showTimestamp" class="term-time">{{ fmtTime(item.timestamp) }}</span>
            <span v-if="showHex && item.hex" class="term-hex">{{ item.hex }}</span>
            <span v-if="showChecksum && item.text" class="term-crc">{{ calcChecksumStr(item.text) }}</span>
            <span class="term-text" :class="{ 'term-tx': item.sent }">{{ showAsciiOnly ? asciiOnly(item.text) : item.text }}</span>
          </div>
        </div>

        <!-- 发送区域 -->
        <div class="send-bar-wrap">
          <div class="send-options">
            <label class="toggle-label"><input type="checkbox" v-model="hexInput" /> HEX输入</label>
            <select v-model="lineEnding" class="le-select">
              <option value="crlf">⏎ CR+LF</option>
              <option value="cr">← CR</option>
              <option value="lf">↓ LF</option>
              <option value="none">— 无</option>
            </select>
            <label class="toggle-label"><input type="checkbox" v-model="showChecksum" /> 校验</label>
            <select v-model="checksumAlgo" class="le-select" v-if="showChecksum">
              <option value="crc16">CRC16</option>
              <option value="crc32">CRC32</option>
              <option value="xor">XOR</option>
              <option value="sum">累加和</option>
            </select>
            <span v-if="showChecksum && sendText" class="checksum-preview">{{ calcChecksumDisplay() }}</span>
            <label class="toggle-label"><input type="checkbox" v-model="repeatSend" /> 定时发送</label>
            <input v-if="repeatSend" v-model.number="repeatInterval" type="number" min="100" step="100" class="repeat-input" /> ms
            <span v-if="repeatSend && repeatActive" class="repeat-indicator">● 发送中</span>
            <label class="toggle-label" v-if="presets.length"><input type="checkbox" v-model="loopSend" @change="onLoopSendToggle" /> 循环发送</label>
            <input v-if="loopSend" v-model.number="loopInterval" type="number" min="100" step="100" class="repeat-input" /> ms
            <span v-if="loopSend && loopActive" class="repeat-indicator">● {{ loopIndex + 1 }}/{{ presets.length }}</span>
          </div>
          <div class="send-bar">
            <input ref="sendRef" v-model="sendText" class="send-input"
              :placeholder="hexInput ? '输入 HEX 字节，如 41 54 0D 0A' : '输入要发送的数据，按 Enter 发送...'"
              @keyup.enter="hexInput ? sendHex() : doSend(sendText)" :disabled="!opened" />
            <button class="btn send" @click="hexInput ? sendHex() : doSend(sendText)" :disabled="!opened">发送</button>
          </div>
        </div>
      </section>
    </div>

    <!-- 状态栏 -->
    <footer class="statusbar">
      <span>{{ isDesktop ? '🟢 Bridge' : '🟡 模拟' }}</span>
      <span v-if="opened">| {{ portInfo?.port }} @ {{ portInfo?.baudRate }}</span>
      <span v-else>| 未打开</span>
      <span class="statusbar-spacer"></span>
      <span v-if="terminalData.length">RX: {{ rxCount }} | TX: {{ txCount }}</span>
      <span v-if="error" class="status-error">{{ error }}</span>
    </footer>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useSerial } from './composables/useSerial'

const { ports, opened, portInfo, data, error, listPorts, open, close, send, getState, isDesktop } = useSerial()

// ———— 状态 ————
const selectedPort = ref('')
const scanning = ref(false)
const sendText = ref('')
const showHex = ref(true)
const showTimestamp = ref(true)
const autoScroll = ref(true)
const showAsciiOnly = ref(false)
const terminalData = ref([])
const termRef = ref(null)
const sendRef = ref(null)
const hexInput = ref(false)
const showChecksum = ref(false)
const checksumAlgo = ref('crc16')

// 网络桥接
const bridgeMode = ref('')
const bridgeHost = ref('')
const bridgePort = ref(8888)
const bridgeRemote = ref('')
const bridgeActive = ref(false)
const bridgeStatus = ref('')

// 预设指令
const presets = ref(loadPresets())

// 发送选项
const lineEnding = ref('crlf')
const repeatSend = ref(false)
const repeatInterval = ref(1000)
const repeatActive = ref(false)
let repeatTimer = null
let loopTimer = null
const loopSend = ref(false)
const loopInterval = ref(2000)
const loopActive = ref(false)
const loopIndex = ref(0)

// DTR/RTS
const dtrOn = ref(false)
const rtsOn = ref(false)
const modemStatus = reactive({ cts: false, dsr: false, dcd: false, ri: false })

const cfg = reactive({ baudRate: 115200, dataBits: 8, parity: 'none', stopBits: 1 })
const quickCmds = ['AT\r\n', 'AT+GMR\r\n', 'AT+CSQ\r\n', 'AT+CREG?\r\n']

const parityShort = computed(() => ({ none:'N',odd:'O',even:'E',mark:'M',space:'S' })[cfg.parity]||'N')
const rxCount = computed(() => terminalData.value.filter(d => !d.sent).length)
const txCount = computed(() => terminalData.value.filter(d => d.sent).length)

// 行尾
const lineEndingMap = { crlf: '\r\n', cr: '\r', lf: '\n', none: '' }

// 同步数据——computed 快照确保每次变化都触发
const dataSnapshot = computed(() => [...data.value])
watch(dataSnapshot, (nd) => {
  const newItems = nd.slice(terminalData.value.length)
  if (newItems.length) {
    terminalData.value.push(...newItems)
    if (autoScroll.value) nextTick(() => { if (termRef.value) termRef.value.scrollTop = termRef.value.scrollHeight })
  }
})

async function doScan() { scanning.value = true; await listPorts(); if (ports.value.length && !selectedPort.value) selectedPort.value = ports.value[0].name; scanning.value = false }
async function doOpen() { if (!selectedPort.value) return; terminalData.value = []; await open(selectedPort.value, { ...cfg }); nextTick(() => sendRef.value?.focus()); setTimeout(refreshModemStatus, 500) }
async function doClose() { stopRepeat(); stopLoop(); await close(); dtrOn.value = false; rtsOn.value = false }

async function doSend(text) { if (!text || !opened.value) return; await send(text + (lineEndingMap[lineEnding.value] ?? '')); sendText.value = ''; nextTick(() => sendRef.value?.focus()) }

function sendHex() {
  if (!sendText.value || !opened.value) return
  try {
    const hex = sendText.value.replace(/\s+/g, '')
    if (hex.length % 2) throw new Error('HEX 长度必须为偶数')
    const bytes = hex.match(/.{2}/g).map(b => String.fromCharCode(parseInt(b, 16)))
    const text = bytes.join('')
    send(text + (lineEndingMap[lineEnding.value] ?? ''))
    sendText.value = ''
  } catch (e) { error.value = 'HEX 格式错误: ' + e.message }
}

// 定时发送
watch(repeatSend, v => { if (v && opened.value) startRepeat(); else stopRepeat() })
watch(opened, v => { if (!v) stopRepeat() })
function startRepeat() { stopRepeat(); if (!repeatSend.value || !opened.value) return; repeatActive.value = true; repeatTimer = setInterval(() => { if (!opened.value) { stopRepeat(); return }; if (hexInput.value && sendText.value) sendHex(); else if (sendText.value) doSend(sendText.value) }, Math.max(100, repeatInterval.value)) }
function stopRepeat() { if (repeatTimer) { clearInterval(repeatTimer); repeatTimer = null }; repeatActive.value = false }

// ———— 循环发送（遍历预设指令） ————
function onLoopSendToggle() {
  if (loopSend.value) { startLoop() } else { stopLoop() }
}

function stopLoop() {
  if (loopTimer) { clearInterval(loopTimer); loopTimer = null }
  loopActive.value = false; loopIndex.value = 0
}

function startLoop() {
  stopLoop()
  if (!loopSend.value || !presets.value.length || !opened.value) return
  loopActive.value = true
  loopIndex.value = 0
  // 先立即发送第一条
  sendLoopItem()
  loopTimer = setInterval(() => {
    if (!opened.value) { stopLoop(); return }
    loopIndex.value = (loopIndex.value + 1) % presets.value.length
    sendLoopItem()
  }, Math.max(100, loopInterval.value))
}

function sendLoopItem() {
  const p = presets.value[loopIndex.value]
  if (!p) return
  const text = p.text + (lineEndingMap[p.le || lineEnding.value] ?? '')
  send(text)
}

// DTR/RTS
async function toggleDtr() { if (!isDesktop) { dtrOn.value = !dtrOn.value; return }; try { const r = await window.__lhpanda__('setDtr', { on: !dtrOn.value }); if (r.success) dtrOn.value = r.data.dtr } catch {} }
async function toggleRts() { if (!isDesktop) { rtsOn.value = !rtsOn.value; return }; try { const r = await window.__lhpanda__('setRts', { on: !rtsOn.value }); if (r.success) rtsOn.value = r.data.rts } catch {} }
async function refreshModemStatus() { if (!isDesktop || !opened.value) return; try { const r = await window.__lhpanda__('getModemStatus', {}); if (r.success) Object.assign(modemStatus, r.data) } catch {} }

// 日志
function saveLog() {
  const lines = terminalData.value.map(item => {
    let l = ''; if (showTimestamp.value) l += fmtTime(item.timestamp) + ' '; if (showHex.value && item.hex) l += item.hex + ' '; l += (item.sent ? '[TX] ' : '[RX] ') + item.text; return l
  }).join('\n')
  const b = new Blob([lines], { type: 'text/plain' }); const u = URL.createObjectURL(b); const a = document.createElement('a'); a.href = u; a.download = `serial-log-${Date.now()}.txt`; a.click(); URL.revokeObjectURL(u)
}

function clearTerminal() { terminalData.value = [] }
function fmtTime(ts) { if (!ts) return ''; const d = new Date(ts); return d.toLocaleTimeString('zh-CN',{hour12:false}) + '.' + String(d.getMilliseconds()).padStart(3,'0') }
function escapeCrlf(s) { return s.replace(/\r/g,'\\r').replace(/\n/g,'\\n') }
function asciiOnly(s) { return s.replace(/[^\x20-\x7E\r\n\t]/g, '.') }

function onDragStart(e) {
  // 点击 no-drag 元素（按钮等）时不触发拖拽
  if (e.target.closest('.no-drag')) return
  if (isDesktop && window.__lhpanda__) window.__lhpanda__('dragWindow',{}).catch(()=>{})
}
function toggleMax() { if (isDesktop && window.__lhpanda__) window.__lhpanda__('toggleMaximize',{}).catch(()=>{}) }
function closeWindow() { if (isDesktop && window.__lhpanda__) window.__lhpanda__('closeWindow',{}).catch(()=>{}) }

// ———— 预设指令 ————
function loadPresets() { try { return JSON.parse(localStorage.getItem('serial_presets')||'[]') } catch { return [] } }
function savePresets() { localStorage.setItem('serial_presets', JSON.stringify(presets.value)) }

function addPreset() {
  const text = sendText.value
  if (!text) return
  const label = prompt('预设名称（留空使用内容前20字）:', text.slice(0, 20))
  presets.value.push({ label: label || text.slice(0, 20), text, hex: hexInput.value, le: lineEnding.value })
  savePresets()
}

function editPreset(i) {
  const p = presets.value[i]
  sendText.value = p.text
  hexInput.value = p.hex || false
  lineEnding.value = p.le || 'crlf'
  sendRef.value?.focus()
}

function sendPreset(p) {
  const text = p.text + (lineEndingMap[p.le || lineEnding.value] ?? '')
  send(text)
}

// ———— 网络桥接 ————
async function doStartBridge() {
  if (!isDesktop) { bridgeActive.value = true; bridgeStatus.value = '模拟桥接已启动'; return }
  try {
    let r
    if (bridgeMode.value === 'tcp-server') r = await window.__lhpanda__('startSerialTcpServer', { port: bridgePort.value })
    else if (bridgeMode.value === 'tcp-client') r = await window.__lhpanda__('startSerialTcpClient', { host: bridgeHost.value, port: bridgePort.value })
    else if (bridgeMode.value === 'udp') {
      const [rh, rp] = bridgeRemote.value.split(':')
      r = await window.__lhpanda__('startSerialUdp', { bindPort: bridgePort.value, remoteHost: rh || '', remotePort: parseInt(rp) || 0 })
    }
    if (r && r.success) { bridgeActive.value = true; bridgeStatus.value = r.data.mode + ' 已启动' }
  } catch (e) { bridgeStatus.value = '失败: ' + e.message }
}

async function doStopBridge() {
  if (isDesktop) await window.__lhpanda__('stopSerialBridge', {}).catch(() => {})
  bridgeActive.value = false; bridgeStatus.value = ''
}

// ———— 校验算法 ————
function calcCRC16(data) {
  let crc = 0xFFFF
  for (let i = 0; i < data.length; i++) {
    crc ^= data.charCodeAt(i)
    for (let j = 0; j < 8; j++) {
      if (crc & 1) crc = (crc >> 1) ^ 0xA001
      else crc >>= 1
    }
  }
  return crc
}

function calcCRC32(data) {
  let crc = 0xFFFFFFFF
  for (let i = 0; i < data.length; i++) {
    crc ^= data.charCodeAt(i)
    for (let j = 0; j < 8; j++) {
      if (crc & 1) crc = (crc >>> 1) ^ 0xEDB88320
      else crc >>>= 1
    }
  }
  return (crc ^ 0xFFFFFFFF) >>> 0
}

function calcXOR(data) {
  let v = 0
  for (let i = 0; i < data.length; i++) v ^= data.charCodeAt(i)
  return v
}

function calcSum(data) {
  let v = 0
  for (let i = 0; i < data.length; i++) v += data.charCodeAt(i)
  return v & 0xFF
}

function calcChecksumDisplay() {
  const data = sendText.value
  if (!data) return ''
  let val
  switch (checksumAlgo.value) {
    case 'crc16': val = calcCRC16(data); return 'CRC16: 0x' + val.toString(16).toUpperCase().padStart(4, '0')
    case 'crc32': val = calcCRC32(data); return 'CRC32: 0x' + val.toString(16).toUpperCase().padStart(8, '0')
    case 'xor': val = calcXOR(data); return 'XOR: 0x' + val.toString(16).toUpperCase().padStart(2, '0')
    case 'sum': val = calcSum(data); return 'SUM: 0x' + val.toString(16).toUpperCase().padStart(2, '0')
    default: return ''
  }
}

function calcChecksumStr(data) {
  if (!data) return ''
  let val
  switch (checksumAlgo.value) {
    case 'crc16': val = calcCRC16(data); return val.toString(16).toUpperCase().padStart(4, '0')
    case 'crc32': val = calcCRC32(data); return val.toString(16).toUpperCase().padStart(8, '0')
    case 'xor': val = calcXOR(data); return val.toString(16).toUpperCase().padStart(2, '0')
    case 'sum': val = calcSum(data); return val.toString(16).toUpperCase().padStart(2, '0')
    default: return ''
  }
}

// 处理bridge事件——同样使用 computed 快照
const dataSnapshot2 = computed(() => [...data.value])
watch(dataSnapshot2, (nd) => {
  const last = nd[nd.length - 1]
  if (last && last.type === 'bridge') {
    bridgeStatus.value = last.msg
    if (last.event === 'disconnected') bridgeActive.value = false
  }
})

onMounted(async () => {
  await getState(); await doScan(); nextTick(() => sendRef.value?.focus()); setInterval(refreshModemStatus, 2000)
})
onUnmounted(() => { stopRepeat(); stopLoop() })
</script>

<style>
:root { --bg:#f5f5f5; --bg2:#e8e8e8; --bg3:#ddd; --border:#ccc; --text:#1a1a1a; --text2:#666; --accent:#333; --green:#444; --red:#999; --yellow:#666; --orange:#777; }
.app { display:flex;flex-direction:column;height:100vh;background:var(--bg);-webkit-user-select:none;user-select:none; }
input,select,button { -webkit-user-select:text;user-select:text; }

.titlebar { display:flex;align-items:center;gap:8px;height:36px;padding:0 8px;background:var(--bg2);border-bottom:1px solid var(--border);-webkit-app-region:drag; }
.no-drag { -webkit-app-region:no-drag; }
.titlebar-icon { font-size:16px; }
.titlebar-text { font-size:13px;font-weight:600;color:var(--text); }
.titlebar-badge { font-size:10px;padding:1px 8px;border-radius:10px;background:#1a3a1a;color:var(--green); }
.titlebar-badge.warn { background:#3a2a1a;color:var(--yellow); }
.titlebar-spacer { flex:1; }
.titlebar-btn { width:28px;height:28px;border:none;border-radius:6px;background:transparent;color:var(--text2);font-size:14px;cursor:pointer;display:flex;align-items:center;justify-content:center; }
.titlebar-btn:hover { background:var(--bg3);color:var(--text); }
.titlebar-btn.close:hover { background:var(--red);color:#fff; }

.main { display:flex;flex:1;overflow:hidden; }
.sidebar { width:240px;flex-shrink:0;overflow-y:auto;border-right:1px solid var(--border);background:var(--bg2);padding:6px;display:flex;flex-direction:column;gap:6px; }
.panel { background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:10px; }
.panel h3 { font-size:11px;text-transform:uppercase;letter-spacing:0.5px;color:var(--text2);margin-bottom:8px; }
.port-list { display:flex;flex-direction:column;gap:2px;margin-top:6px; }
.port-item { padding:6px 10px;border-radius:4px;font-size:12px;font-family:monospace;color:var(--text2);cursor:pointer;display:flex;align-items:center;justify-content:space-between; }
.port-item:hover { background:var(--bg3);color:var(--text); }
.port-item.active { background:var(--bg3);color:var(--text);font-weight:700; }
.port-status { color:var(--green);font-size:8px; }
.form-row { margin-bottom:6px;display:flex;align-items:center;gap:8px; }
.form-row label { font-size:11px;color:var(--text2);width:50px;flex-shrink:0; }
.form-row select { flex:1;padding:4px 8px;border-radius:4px;border:1px solid var(--border);background:var(--bg2);color:var(--text);font-size:12px;outline:none; }
.form-row select:focus { border-color:var(--accent); }
.conn-info { margin-top:8px;padding:6px 10px;border-radius:4px;background:var(--bg2);font-size:11px;font-family:monospace;color:var(--text);display:flex;align-items:center;gap:6px; }
.conn-dot { width:6px;height:6px;border-radius:50%;background:var(--green);animation:pulse 1.5s infinite; }
.quick-cmds { display:flex;flex-wrap:wrap;gap:4px; }

/* 预设指令 */
.preset-list { display:flex;flex-direction:column;gap:2px;max-height:180px;overflow-y:auto; }
.preset-item { display:flex;align-items:center;justify-content:space-between;padding:4px 8px;border-radius:4px;font-size:11px; }
.preset-item:hover { background:var(--bg3); }
.preset-name { color:var(--text);font-family:monospace;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1; }
.preset-actions { display:flex;gap:2px;flex-shrink:0; }
.btn-icon { width:22px;height:22px;border:none;border-radius:4px;background:transparent;color:var(--text2);font-size:12px;cursor:pointer;display:flex;align-items:center;justify-content:center; }
.btn-icon:hover { background:var(--bg3);color:var(--text); }
.btn-icon.danger:hover { background:var(--red);color:#fff; }

/* 网络桥接输入 */
.finput { flex:1;padding:4px 6px;border-radius:4px;border:1px solid var(--border);background:var(--bg2);color:var(--text);font-size:11px;font-family:monospace;outline:none; }
.finput:focus { border-color:var(--accent); }

/* 校验预览 */
.checksum-preview { font-size:10px;color:var(--yellow);font-family:monospace;white-space:nowrap; }
.term-crc { color:var(--orange);flex-shrink:0;font-size:10px;min-width:40px; }

.hw-controls { display:flex;flex-wrap:wrap;align-items:center;gap:6px; }
.hw-btn { padding:4px 12px;border-radius:4px;border:1px solid var(--border);background:var(--bg3);color:var(--text2);font-size:11px;font-weight:600;cursor:pointer;min-width:44px;text-align:center; }
.hw-btn:hover { background:#30363d; }
.hw-btn.on { background:var(--bg);border-color:var(--text);color:var(--text);font-weight:700; }
.modem-bits { display:flex;gap:4px;margin-left:auto; }
.modem-bit { font-size:10px;padding:1px 5px;border-radius:3px;background:var(--bg3);color:var(--text2); }
.modem-bit.on { background:var(--bg);color:var(--text);font-weight:700; }

.terminal-section { flex:1;display:flex;flex-direction:column;overflow:hidden; }
.terminal-header { display:flex;align-items:center;gap:12px;padding:6px 12px;background:var(--bg2);border-bottom:1px solid var(--border);font-size:12px;color:var(--text2); }
.terminal-actions { display:flex;align-items:center;gap:8px;margin-left:auto;flex-wrap:wrap; }
.toggle-label { font-size:11px;color:var(--text2);display:flex;align-items:center;gap:4px;cursor:pointer;white-space:nowrap; }
.toggle-label input { accent-color:var(--accent); }
.terminal-body { flex:1;overflow-y:auto;padding:8px 0;background:#fff;font-family:'Cascadia Code','Fira Code','Consolas',monospace;font-size:12px;line-height:1.6;border:1px solid var(--border); }
.term-line { display:flex;gap:8px;padding:1px 12px;border-bottom:1px solid #f0f0f0; }
.term-time { color:var(--text2);flex-shrink:0;min-width:80px;font-size:11px; }
.term-hex { color:var(--text2);flex-shrink:0;font-size:11px; }
.term-text { white-space:pre-wrap;word-break:break-all;color:var(--text); }
.term-tx,.term-sent .term-text { color:var(--text2); }
.term-sent { background:#f0f0f0; }

.send-bar-wrap { border-top:1px solid var(--border);background:var(--bg2); }
.send-options { display:flex;align-items:center;gap:8px;padding:4px 12px;flex-wrap:wrap; }
.le-select { padding:2px 6px;border-radius:4px;border:1px solid var(--border);background:var(--bg);color:var(--text2);font-size:11px;outline:none; }
.le-select:focus { border-color:var(--accent); }
.repeat-input { width:55px;padding:2px 4px;border-radius:4px;border:1px solid var(--border);background:var(--bg);color:var(--text);font-size:11px;text-align:center;outline:none; }
.repeat-input:focus { border-color:var(--accent); }
.repeat-indicator { font-size:10px;color:var(--green);animation:pulse 1s infinite; }
.send-bar { display:flex;gap:8px;padding:4px 12px 8px; }
.send-input { flex:1;padding:8px 12px;border-radius:6px;border:1px solid var(--border);background:var(--bg);color:var(--text);font-size:13px;font-family:monospace;outline:none; }
.send-input:focus { border-color:var(--accent); }
.send-input:disabled { opacity:0.4; }

.statusbar { display:flex;align-items:center;gap:8px;height:26px;padding:0 12px;background:var(--bg2);border-top:1px solid var(--border);font-size:11px;color:var(--text2); }
.statusbar-spacer { flex:1; }
.status-error { color:var(--red);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:300px; }

.btn { padding:6px 14px;border-radius:6px;border:1px solid var(--border);background:var(--bg3);color:var(--text);font-size:12px;cursor:pointer;transition:all 0.15s;width:100%; }
.btn:hover { background:#30363d; }
.btn:disabled { opacity:0.4;cursor:not-allowed; }
.btn.scan { border-color:var(--accent);color:var(--accent); }
.btn.scan:hover { background:#1a3a5a; }
.btn.open { border-color:var(--green);color:var(--green); }
.btn.open:hover { background:#1a3a1a; }
.btn.open.danger { border-color:var(--red);color:var(--red); }
.btn.open.danger:hover { background:#3a1a1a; }
.btn.send { padding:8px 20px;border-color:var(--accent);color:var(--accent);width:auto;font-weight:600; }
.btn.send:hover { background:#1a3a5a; }
.btn-sm { padding:3px 10px;border-radius:4px;border:1px solid var(--border);background:var(--bg3);color:var(--text2);font-size:11px;font-family:monospace;cursor:pointer;white-space:nowrap; }
.btn-sm:hover { background:#30363d;color:var(--text); }
.dim { font-size:12px;color:var(--text2);padding:8px 0; }
.center { text-align:center;padding:20px; }

@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.3} }
</style>
