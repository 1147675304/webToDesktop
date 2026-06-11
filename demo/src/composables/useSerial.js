// demo/src/composables/useSerial.js
// 串口双向数据流 composable
//
// 用法:
//   import { useSerial } from '../composables/useSerial'
//   const { ports, opened, listening, data, listPorts, open, close, send } = useSerial()
//
// 架构:
//   JS → bridge.openSerialPort({port, baudRate, ...})  → Go 打开串口
//   Go serial.Read() → stream channel → pushToJS        → JS stream-data 事件
//   JS → bridge.sendSerialData({data})                  → Go serial.Write()

import { ref, onUnmounted } from 'vue'

// ———— 全局 stream-data 监听（共享） ————
const serialListeners = new Set()

function ensureSerialListener() {
  if (window.__serialListenerSetup) return
  window.__serialListenerSetup = true

  window.addEventListener('stream-data', (e) => {
    const { topic, data } = e.detail || {}
    if (topic !== 'serial-data') return
    serialListeners.forEach(cb => {
      try { cb(data) } catch {}
    })
  })

  window.addEventListener('stream-end', (e) => {
    const { topic } = e.detail || {}
    if (topic !== 'serial-data') return
    serialListeners.forEach(cb => {
      try { cb({ type: 'end' }) } catch {}
    })
  })
}

// ———— Composable ————

export function useSerial() {
  ensureSerialListener()

  const ports = ref([])          // [{name: "/dev/ttyUSB0"}, ...]
  const opened = ref(false)      // 串口是否已打开
  const portInfo = ref(null)     // 当前打开的串口配置
  const data = ref([])           // 接收到的数据 [{type, text, hex, bytes, timestamp}, ...]
  const error = ref('')

  const isDesktop = !!window.__lhpanda__

  // 注册串口数据回调
  const onData = (payload) => {
    if (payload.type === 'end') {
      opened.value = false
      return
    }
    if (payload.type === 'error') {
      error.value = payload.msg
      return
    }
    data.value.push(payload)
    // 限制缓存条数
    if (data.value.length > 500) {
      data.value = data.value.slice(-300)
    }
  }
  serialListeners.add(onData)

  onUnmounted(() => {
    serialListeners.delete(onData)
    if (isDesktop) {
      window.__lhpanda__('closeSerialPort', {}).catch(() => {})
    }
  })

  // 扫描串口列表
  async function listPorts() {
    error.value = ''
    if (isDesktop) {
      try {
        const result = await window.__lhpanda__('listSerialPorts', {})
        if (result.success) {
          ports.value = result.data.ports || []
        }
      } catch (err) {
        error.value = err?.message || String(err)
      }
    } else {
      // 浏览器模拟
      ports.value = [
        { name: '/dev/ttyUSB0' },
        { name: '/dev/ttyACM0' },
        { name: 'COM1' },
        { name: 'COM3' },
      ]
    }
  }

  // 打开串口
  async function open(portName, config = {}) {
    error.value = ''
    data.value = []

    const params = {
      port: portName,
      baudRate: config.baudRate || 115200,
      dataBits: config.dataBits || 8,
      parity: config.parity || 'none',
      stopBits: config.stopBits || 1,
    }

    if (isDesktop) {
      try {
        const result = await window.__lhpanda__('openSerialPort', params)
        if (result.success) {
          opened.value = true
          portInfo.value = result.data
        }
      } catch (err) {
        error.value = err?.message || String(err)
      }
    } else {
      // 浏览器模拟
      opened.value = true
      portInfo.value = { port: portName, ...params }
      mockSerialData()
    }
  }

  // 关闭串口
  async function close() {
    if (isDesktop) {
      try { await window.__lhpanda__('closeSerialPort', {}) } catch {}
    }
    opened.value = false
    portInfo.value = null
  }

  // 发送数据到串口
  async function send(text) {
    error.value = ''
    // 始终先在本地显示发送数据（前端保障）
    const hex = Array.from(new TextEncoder().encode(text)).map(b => b.toString(16).padStart(2, '0').toUpperCase()).join(' ')
    data.value.push({ type: 'data', text, hex, bytes: text.length, timestamp: Date.now(), sent: true })
    if (isDesktop) {
      window.__lhpanda__('sendSerialData', { data: text }).catch(err => { error.value = err?.message || String(err) })
    }
  }

  return { ports, opened, portInfo, data, error, listPorts, open, close, send }
}

// ———— 浏览器模拟 ————
function mockSerialData() {
  const messages = [
    'AT\r\n', 'OK\r\n',
    'AT+GMR\r\n', 'SIM800 R14.18\r\n', 'OK\r\n',
    '\r\nREADY\r\n',
  ]
  let i = 0
  const timer = setInterval(() => {
    if (i >= messages.length) {
      clearInterval(timer)
      return
    }
    const text = messages[i]
    serialListeners.forEach(cb => cb({
      type: 'data',
      text,
      hex: Array.from(new TextEncoder().encode(text)).map(b => b.toString(16).padStart(2, '0').toUpperCase()).join(' '),
      bytes: text.length,
      timestamp: Date.now(),
    }))
    i++
  }, 600 + Math.random() * 400)
}
