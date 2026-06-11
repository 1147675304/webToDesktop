// debugtool/src/composables/useSerial.js
// 串口双向数据流 composable — 独立版本，无外部依赖
//
// 用法:
//   import { useSerial } from './composables/useSerial'
//   const { ports, opened, data, listPorts, open, close, send } = useSerial()
//
// 架构:
//   JS → bridge.listSerialPorts / openSerialPort / sendSerialData / closeSerialPort → Go
//   Go serial.Read() → stream channel → pushToJS → CustomEvent('stream-data', {topic:"serial-data"})
//   JS window.addEventListener('stream-data', ...) → data.value 实时更新
//
// 浏览器模式下自动模拟串口数据，无需硬件即可调试 UI。

import { ref, onUnmounted } from 'vue'

// ——— 全局 stream-data 事件监听（多个 useSerial 实例共享）———

/** 所有 useSerial 实例的数据回调集合 */
const listeners = new Set()

/** 全局监听器是否已注册（只注册一次，避免重复绑定） */
let globalSetup = false

function setupGlobalListener() {
  if (globalSetup) return
  globalSetup = true

  // Go 端推送串口数据 → CustomEvent('stream-data')
  window.addEventListener('stream-data', e => {
    const { topic, data } = e.detail || {}
    if (topic !== 'serial-data') return
    listeners.forEach(cb => { try { cb(data) } catch {} })
  })

  // Go 端关闭串口 / 流结束 → CustomEvent('stream-end')
  window.addEventListener('stream-end', e => {
    const { topic } = e.detail || {}
    if (topic !== 'serial-data') return
    listeners.forEach(cb => { try { cb({ type: 'end' }) } catch {} })
  })
}

// ——— useSerial composable ———

export function useSerial() {
  setupGlobalListener()

  const ports = ref([])        // 可用串口列表 [{name}, ...]
  const opened = ref(false)    // 串口是否已打开
  const portInfo = ref(null)   // 当前串口参数
  const data = ref([])         // 接收/发送的数据 [{type, text, hex, bytes, timestamp, sent?}, ...]
  const error = ref('')        // 错误信息
  const isDesktop = !!window.__lhpanda__

  // 收到 Go 推送的数据: type='data' 追加, 'end' 标记关闭, 'error' 记录错误
  const onData = (payload) => {
    if (payload.type === 'end') { opened.value = false; return }
    if (payload.type === 'error') { error.value = payload.msg; return }
    data.value.push(payload)
    if (data.value.length > 500) data.value = data.value.slice(-300) // 限制缓存
  }
  listeners.add(onData)

  // 组件卸载: 移除回调 + 关闭串口
  onUnmounted(() => {
    listeners.delete(onData)
    if (isDesktop) window.__lhpanda__('closeSerialPort', {}).catch(() => {})
  })

  // —— 公开方法 ——

  /** 扫描可用串口。桌面调 Go bridge，浏览器返回模拟端口。 */
  async function listPorts() {
    error.value = ''
    if (isDesktop) {
      const r = await window.__lhpanda__('listSerialPorts', {})
      if (r.success) ports.value = r.data.ports || []
    } else {
      ports.value = [
        { name: '/dev/ttyUSB0' }, { name: '/dev/ttyACM0' },
        { name: 'COM1' }, { name: 'COM3' },
      ]
    }
  }

  /** 打开串口并启动数据流 @param {string} portName @param {object} cfg {baudRate, dataBits, parity, stopBits} */
  async function open(portName, cfg = {}) {
    error.value = ''; data.value = []
    const p = { port: portName, baudRate: cfg.baudRate || 115200, dataBits: cfg.dataBits || 8, parity: cfg.parity || 'none', stopBits: cfg.stopBits || 1 }
    if (isDesktop) {
      const r = await window.__lhpanda__('openSerialPort', p)
      if (r.success) { opened.value = true; portInfo.value = r.data }
    } else {
      opened.value = true; portInfo.value = { port: portName, ...p }
      mockSerialData() // 浏览器模拟
    }
  }

  /** 关闭串口 */
  async function close() {
    if (isDesktop) await window.__lhpanda__('closeSerialPort', {}).catch(() => {})
    opened.value = false; portInfo.value = null
  }

  /**
   * 向串口发送数据。先在本地 push（前端保障即时显示），再异步调 Go 写入硬件。
   * Go 端也会通过 stream 回显，但本地 push 确保不依赖时序。
   */
  async function send(text) {
    error.value = ''
    data.value.push({ type: 'data', text, hex: toHex(text), bytes: text.length, timestamp: Date.now(), sent: true })
    if (isDesktop) window.__lhpanda__('sendSerialData', { data: text }).catch(() => {})
  }

  /** 查询串口状态 */
  async function getState() {
    if (isDesktop) {
      const r = await window.__lhpanda__('getSerialState', {})
      if (r.success) opened.value = r.data.opened || false
    }
  }

  return { ports, opened, portInfo, data, error, listPorts, open, close, send, getState, isDesktop }
}

// ——— 工具函数 ———

/** 字符串 → 空格分隔的十六进制，如 "AT\r\n" → "41 54 0D 0A" */
function toHex(s) {
  return Array.from(new TextEncoder().encode(s))
    .map(b => b.toString(16).padStart(2, '0').toUpperCase())
    .join(' ')
}

/** 浏览器模式：模拟 AT 指令串口数据（500~900ms 随机间隔） */
function mockSerialData() {
  const msgs = ['AT\r\n', 'OK\r\n', 'AT+GMR\r\n', 'SIM800 R14.18\r\n', 'OK\r\n', '\r\nREADY\r\n']
  let i = 0
  const t = setInterval(() => {
    if (i >= msgs.length) { clearInterval(t); return }
    const text = msgs[i]
    listeners.forEach(cb => cb({ type: 'data', text, hex: toHex(text), bytes: text.length, timestamp: Date.now() }))
    i++
  }, 500 + Math.random() * 400)
}
