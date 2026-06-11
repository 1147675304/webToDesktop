// demo/src/composables/useStream.js
// Go → JS 流式数据监听 composable
//
// 用法:
//   import { useStream } from '../composables/useStream'
//   const { data, error, listening, listen, stop } = useStream('my-topic')
//   listen()  // 开始监听
//
// Go 侧数据通过 CustomEvent('stream-data') 推送，
// 本 composable 负责监听、解析并暴露为 Vue 响应式数据。

import { ref, onUnmounted } from 'vue'

// ———— 全局事件总线 ————
// 所有 useStream 实例共享同一个事件监听器，
// 按 topic 路由到对应的回调。

const listeners = new Map() // topic → Set<callback>

function ensureGlobalListener() {
  if (window.__streamDataListener) return
  window.__streamDataListener = true

  window.addEventListener('stream-data', (e) => {
    const { topic, data } = e.detail || {}
    if (!topic) return
    const cbs = listeners.get(topic)
    if (cbs) {
      cbs.forEach(cb => {
        try { cb(data) } catch {}
      })
    }
  })

  window.addEventListener('stream-end', (e) => {
    const { topic } = e.detail || {}
    if (!topic) return
    const cbs = listeners.get('__end__' + topic)
    if (cbs) {
      cbs.forEach(cb => {
        try { cb() } catch {}
      })
    }
  })
}

// ———— 模拟数据（浏览器模式） ————
const mockStreams = {}

function mockListen(topic) {
  if (mockStreams[topic]) {
    clearInterval(mockStreams[topic].timer)
  }
  let count = 0
  const timer = setInterval(() => {
    count++
    const cbs = listeners.get(topic)
    if (cbs) {
      cbs.forEach(cb => cb({ topic, index: count, time: Date.now(), text: `模拟消息 #${count}（浏览器模式）` }))
    }
    if (count >= 10) {
      mockStop(topic)
    }
  }, 800)
  mockStreams[topic] = { timer, count }
}

function mockStop(topic) {
  if (mockStreams[topic]) {
    clearInterval(mockStreams[topic].timer)
    delete mockStreams[topic]
  }
  const cbs = listeners.get('__end__' + topic)
  if (cbs) {
    cbs.forEach(cb => cb())
  }
}

// ———— Composable ————

/**
 * 创建一个流式数据监听器。
 *
 * @param {string} topic 数据流主题，需与 Go 端一致
 * @returns {{ data: Ref<Array>, latest: Ref<any>, listening: Ref<boolean>, error: Ref<string|null>, listen: Function, stop: Function }}
 */
export function useStream(topic) {
  ensureGlobalListener()

  const data = ref([])           // 所有收到的数据
  const latest = ref(null)       // 最新一条数据
  const listening = ref(false)
  const error = ref(null)

  // 是否在桌面环境
  const isDesktop = !!window.__lhpanda__

  // 注册数据回调
  if (!listeners.has(topic)) {
    listeners.set(topic, new Set())
  }
  const dataCallback = (payload) => {
    data.value.push(payload)
    latest.value = payload
  }
  listeners.get(topic).add(dataCallback)

  // 注册结束回调
  const endKey = '__end__' + topic
  if (!listeners.has(endKey)) {
    listeners.set(endKey, new Set())
  }
  const endCallback = () => {
    listening.value = false
  }
  listeners.get(endKey).add(endCallback)

  // 开始监听
  async function listen() {
    error.value = null
    if (isDesktop) {
      try {
        const result = await window.__lhpanda__('listenStream', { topic })
        if (result.success) {
          listening.value = true
          data.value = []  // 重置数据
          latest.value = null
        }
      } catch (err) {
        error.value = err?.message || String(err)
      }
    } else {
      // 浏览器模式：模拟数据
      listening.value = true
      data.value = []
      latest.value = null
      mockListen(topic)
    }
  }

  // 停止监听
  async function stop() {
    if (isDesktop) {
      try {
        await window.__lhpanda__('stopStream', { topic })
      } catch {}
    } else {
      mockStop(topic)
    }
    listening.value = false
  }

  // 向流发送数据（从 JS 侧推送）
  async function send(dataPayload) {
    if (isDesktop) {
      try {
        await window.__lhpanda__('sendToStream', { topic, data: dataPayload })
      } catch (err) {
        error.value = err?.message || String(err)
      }
    } else {
      // 浏览器模拟：直接触发
      const cbs = listeners.get(topic)
      if (cbs) {
        cbs.forEach(cb => cb(dataPayload))
      }
    }
  }

  // 组件卸载时自动清理
  onUnmounted(() => {
    stop()
    const cbs = listeners.get(topic)
    if (cbs) cbs.delete(dataCallback)
    const endCbs = listeners.get(endKey)
    if (endCbs) endCbs.delete(endCallback)
  })

  return { data, latest, listening, error, listen, stop, send }
}
