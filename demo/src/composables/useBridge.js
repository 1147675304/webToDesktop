import { ref, computed } from 'vue'

// ———— 模拟存储（浏览器模式） ————
const _mockStore = {}
const _mockKeyValues = JSON.parse(localStorage.getItem('wtd_key_values') || '{}')

// ———— 底层调用 ————
function callBridge(method, params = {}) {
  if (window.__lhpanda__) {
    return window.__lhpanda__(method, params).catch(err => {
      console.warn('Bridge ' + method + ':', err)
      throw err
    })
  }
  return Promise.resolve(mockBridge(method, params))
}

function mockBridge(method, params) {
  switch (method) {
    case 'getAppInfo':
      return { platform: 'browser', arch: 'x64', version: '1.0.0' }
    case 'getCredentials':
      if (params.username) return { found: !!_mockStore[params.username], username: params.username }
      return { found: Object.keys(_mockStore).length > 0, credentials: Object.keys(_mockStore).map(u => ({ username: u })) }
    case 'saveCredentials':
      _mockStore[params.username] = params.password; return { saved: true }
    case 'deleteCredentials':
      delete _mockStore[params.username]; return { deleted: true }
    case 'clearCredentials':
      Object.keys(_mockStore).forEach(k => delete _mockStore[k]); return { cleared: true }
    case 'dragWindow': case 'closeWindow': case 'toggleMaximize': case 'toggleFullscreen': case 'toggleMinimize': case 'restartApp':
      alert('桌面模式: ' + method + '（当前为浏览器模拟）'); return { ok: true }
    case 'resizeWindow': return { ok: true }
    case 'getWindowConfig': {
      const saved = localStorage.getItem('wtd_window_config')
      if (saved) { try { return JSON.parse(saved) } catch {} }
      return { title: 'WebToDesktop Demo', width: 1024, height: 768, fullscreen: false, maximized: false, borderless: true, always_on_top: false, opacity: 1, webview_bg_transparent: false, window_position: 'center', dark_title_bar: false, round_corners: true, acrylic: false }
    }
    case 'saveWindowConfig':
      localStorage.setItem('wtd_window_config', JSON.stringify(params)); return { saved: true, needRestart: true }
    case 'setItem':
      _mockKeyValues[params.key] = params.value
      localStorage.setItem('wtd_key_values', JSON.stringify(_mockKeyValues))
      return { saved: true }
    case 'getItem':
      if (params.key in _mockKeyValues) return { found: true, value: _mockKeyValues[params.key] }
      return { found: false }
    case 'removeItem':
      delete _mockKeyValues[params.key]
      localStorage.setItem('wtd_key_values', JSON.stringify(_mockKeyValues))
      return { removed: true }
    case 'clearItems':
      Object.keys(_mockKeyValues).forEach(k => delete _mockKeyValues[k])
      localStorage.setItem('wtd_key_values', JSON.stringify(_mockKeyValues))
      return { cleared: true }
    case 'getAllItems':
      return { items: { ..._mockKeyValues } }
    case 'listMethods':
      return { methods: ['getAppInfo', 'listMethods', 'saveCredentials', 'getCredentials', 'deleteCredentials', 'clearCredentials', 'dragWindow', 'resizeWindow', 'closeWindow', 'toggleMaximize', 'toggleFullscreen', 'toggleMinimize', 'restartApp', 'getWindowConfig', 'saveWindowConfig', 'setItem', 'getItem', 'removeItem', 'clearItems', 'getAllItems'] }
    default: return {}
  }
}

// ———— 应用信息 ————
const appInfo = ref({ platform: 'browser', arch: 'x64', version: '1.0.0' })
const bridgeReady = ref(false)
const isDesktop = computed(() => appInfo.value.platform !== 'browser')
const isWindows = computed(() => appInfo.value.platform === 'windows')

async function initAppInfo() {
  try {
    const info = await bridge.getAppInfo()
    appInfo.value = info
    bridgeReady.value = !!window.__lhpanda__
  } catch {}
}

// ———— 拖拽条 ————
function onDragBarMouseDown(e, win) {
  if (e.target.closest('button, a, input, select, textarea, .no-drag')) return
  if (!win) bridge.dragWindow().catch(() => {})
}

// ———— Bridge 代理：bridge.methodName(params) → callBridge('methodName', params) ————
const bridge = new Proxy({}, {
  get(_, method) {
    return (params) => callBridge(method, params ?? {})
  },
})

export function useBridge() {
  return { bridge, appInfo, bridgeReady, isDesktop, isWindows, initAppInfo, onDragBarMouseDown }
}
