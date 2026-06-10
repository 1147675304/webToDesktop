/**
 * localStorageProxy — 接管浏览器原生 localStorage API，无缝切换存储后端
 *
 * 在桌面模式下，透明地将 localStorage 调用路由到 Go 后端的 AES-256-GCM 加密存储。
 * 在浏览器模式下，保持原生 localStorage 行为不变。
 *
 * 原理：
 *   1. 启动时预加载 Go 后端所有键值对到内存缓存
 *   2. 用 Proxy 替换 window.localStorage，读写操作先更新内存缓存（同步）
 *   3. 异步批量 flush 到 Go 后端加密存储
 *   4. 页面关闭前 flush 所有待写入
 *
 * 使用方式：
 *   // 在 App.vue 等最早初始化的地方调用一次即可
 *   import { initLocalStorageProxy } from './composables/localStorageProxy.js'
 *   await initLocalStorageProxy(bridge)
 *
 *   // 之后所有代码都可直接使用标准的 localStorage API
 *   localStorage.setItem('theme', 'dark')
 *   const theme = localStorage.getItem('theme')
 */

// ———— 内部状态 ————
let _initialized = false
let _desktopMode = false
let _cache = null          // Map<string, string> 内存缓存
let _realStorage = null    // 真实的原生 localStorage（浏览器模式用）
let _flushTimer = null
let _dirtyKeys = null      // Set<string> 待 flush 的键
let _bridge = null
let _flushInProgress = false

// ———— 常量 ————
const FLUSH_DELAY_MS = 300       // 批量 flush 防抖延迟
const FLUSH_MAX_WAIT_MS = 2000   // 最大等待时间，超时强制 flush

/**
 * 初始化 localStorage 代理。
 * 在桌面环境下自动接管 localStorage，浏览器环境下保持原生不变。
 *
 * @param {Object} bridge  - 桥接实例（来自 useBridge）
 * @param {Object} options
 * @param {boolean} options.forceDesktop  - 强制使用桌面模式（用于测试）
 * @returns {Promise<{desktop: boolean, loaded: number}>}
 */
export async function initLocalStorageProxy(bridge = {}, options = {}) {
  if (_initialized) {
    return { desktop: _desktopMode, loaded: _cache ? _cache.size : 0 }
  }

  // 保存真实的 localStorage 引用
  try {
    _realStorage = window.localStorage
  } catch {
    _realStorage = null
  }

  // 判断是否桌面模式
  const hasBridge = typeof window.__lhpanda__ !== 'undefined' && !!window.__lhpanda__
  _desktopMode = options.forceDesktop || hasBridge

  if (!_desktopMode) {
    // 浏览器模式：不做任何接管，保持原生 localStorage
    _initialized = true
    return { desktop: false, loaded: 0 }
  }

  // ———— 桌面模式：接管 localStorage ————
  _bridge = bridge
  _cache = new Map()
  _dirtyKeys = new Set()
  _initialized = true

  // 1. 预加载所有键值对
  let loadedCount = 0
  try {
    const result = await bridge.getAllItems()
    if (result && result.items) {
      for (const [key, value] of Object.entries(result.items)) {
        _cache.set(key, value)
        loadedCount++
      }
    }
  } catch (e) {
    console.warn('[localStorageProxy] 预加载失败，使用空缓存:', e)
  }

  // 2. 用 Proxy 替换 window.localStorage
  const proxy = createLocalStorageProxy()
  try {
    Object.defineProperty(window, 'localStorage', {
      value: proxy,
      writable: false,
      configurable: true,
    })
  } catch (e) {
    console.warn('[localStorageProxy] 接管 localStorage 失败:', e)
    // 降级：保持原生 localStorage
    _desktopMode = false
    return { desktop: false, loaded: loadedCount }
  }

  // 3. 注册页面关闭前 flush
  if (typeof window !== 'undefined') {
    window.addEventListener('beforeunload', flushNow)
    // 也监听 pagehide 以兼容移动端/部分浏览器
    window.addEventListener('pagehide', flushNow)
  }

  console.log(`[localStorageProxy] 已接管 localStorage（桌面加密模式），预加载 ${loadedCount} 个键值对`)
  return { desktop: true, loaded: loadedCount }
}

// ———— 创建 localStorage Proxy ————

function createLocalStorageProxy() {
  return new Proxy({}, {
    // ———— 读取属性 ————
    get(target, prop, receiver) {
      switch (prop) {
        // === 标准 API ===
        case 'getItem':
          return getItem
        case 'setItem':
          return setItem
        case 'removeItem':
          return removeItem
        case 'clear':
          return clear
        case 'key':
          return keyMethod

        // === 属性 ===
        case 'length':
          return _cache ? _cache.size : 0

        // === 迭代器 ===
        case Symbol.iterator:
          return entriesIterator
        case 'entries':
          return entriesMethod
        case 'keys':
          return keysMethod
        case 'values':
          return valuesMethod
        case 'forEach':
          return forEachMethod

        // === 工具 ===
        case 'toString':
          return () => '[object Storage]'
        case 'toJSON':
          return () => Object.fromEntries(_cache || [])

        // === 调试 ===
        case '__isDesktopProxy':
          return true

        // === 数字索引（localStorage 支持数字索引访问） ===
        default:
          if (typeof prop === 'string' && !prop.startsWith('__')) {
            // 检查是否为数字索引
            const idx = Number(prop)
            if (!isNaN(idx) && Number.isInteger(idx) && idx >= 0 && _cache) {
              const keys = [..._cache.keys()]
              return keys[idx] || undefined
            }
            // 返回缓存中的值（直接属性访问）
            if (_cache && _cache.has(prop)) {
              return _cache.get(prop)
            }
          }
          return undefined
      }
    },

    // ———— 设置属性（支持直接赋值 localStorage.key = value） ————
    set(target, prop, value) {
      if (typeof prop === 'string' && prop in target) {
        // 不要覆盖内置方法
        return false
      }
      if (typeof prop === 'string' && prop !== 'length') {
        setItem(prop, String(value))
        return true
      }
      return false
    },

    // ———— 删除属性（支持 delete localStorage.key） ————
    deleteProperty(target, prop) {
      if (typeof prop === 'string' && prop !== 'length') {
        removeItem(prop)
        return true
      }
      return false
    },

    // ———— 检查属性是否存在 ————
    has(target, prop) {
      if (typeof prop === 'string') {
        return _cache ? _cache.has(prop) : false
      }
      return prop in target
    },

    // ———— 获取属性描述符 ————
    getOwnPropertyDescriptor(target, prop) {
      if (typeof prop === 'string' && _cache && _cache.has(prop)) {
        return {
          value: _cache.get(prop),
          writable: true,
          enumerable: true,
          configurable: true,
        }
      }
      return undefined
    },

    // ———— 枚举自身属性 ————
    ownKeys() {
      return _cache ? [..._cache.keys()] : []
    },

    // ———— 阻止扩展 ————
    preventExtensions() {
      return false
    },
    isExtensible() {
      return true
    },
  })
}

// ———— 标准 API 实现 ————

function getItem(key) {
  if (!_cache) return null
  const val = _cache.get(key)
  return val !== undefined ? val : null
}

function setItem(key, value) {
  if (!_cache) return
  const strValue = String(value)
  _cache.set(key, strValue)
  markDirty(key)
}

function removeItem(key) {
  if (!_cache) return
  _cache.delete(key)
  markDirty(key)
}

function clear() {
  if (!_cache) return
  // 标记所有现有键为 dirty（需要删除）
  for (const key of _cache.keys()) {
    _dirtyKeys.add(key)
  }
  _cache.clear()
  scheduleFlush()
}

function keyMethod(index) {
  if (!_cache) return null
  const keys = [..._cache.keys()]
  return keys[index] || null
}

// ———— 迭代器 ————

function* entriesIterator() {
  if (!_cache) return
  for (const [key, value] of _cache) {
    yield [key, value]
  }
}

function entriesMethod() {
  return entriesIterator()
}

function* keysMethod() {
  if (!_cache) return
  for (const key of _cache.keys()) {
    yield key
  }
}

function* valuesMethod() {
  if (!_cache) return
  for (const value of _cache.values()) {
    yield value
  }
}

function forEachMethod(callback, thisArg) {
  if (!_cache) return
  for (const [key, value] of _cache) {
    callback.call(thisArg, value, key, this)
  }
}

// ———— 异步持久化 ————

/**
 * 标记一个键为 dirty，安排一次批量 flush。
 */
function markDirty(key) {
  _dirtyKeys.add(key)
  scheduleFlush()
}

/**
 * 防抖调度批量 flush。
 */
function scheduleFlush() {
  if (_flushTimer) return

  _flushTimer = setTimeout(() => {
    _flushTimer = null
    flushDirty()
  }, FLUSH_DELAY_MS)

  // 兜底：超过最大等待时间强制 flush
  setTimeout(() => {
    if (_dirtyKeys.size > 0) {
      clearTimeout(_flushTimer)
      _flushTimer = null
      flushDirty()
    }
  }, FLUSH_MAX_WAIT_MS)
}

/**
 * 批量将脏数据 flush 到 Go 后端。
 * 对于已删除的键（缓存中不存在但在 _dirtyKeys 中），调用 removeItem。
 * 对于存在的键，调用 setItem。
 */
async function flushDirty() {
  if (_flushInProgress || _dirtyKeys.size === 0 || !_bridge) return

  _flushInProgress = true
  const keysToFlush = new Set(_dirtyKeys)
  _dirtyKeys.clear()

  try {
    for (const key of keysToFlush) {
      if (_cache && _cache.has(key)) {
        await _bridge.setItem({ key, value: _cache.get(key) })
      } else {
        // 键已被删除，同步到后端
        await _bridge.removeItem({ key })
      }
    }
  } catch (e) {
    console.warn('[localStorageProxy] flush 失败:', e)
    // 失败时重新加入脏队列，下次重试
    for (const key of keysToFlush) {
      _dirtyKeys.add(key)
    }
  } finally {
    _flushInProgress = false
  }
}

/**
 * 立即 flush 所有待写入数据（页面关闭前调用）。
 */
async function flushNow() {
  if (!_dirtyKeys || _dirtyKeys.size === 0) return

  // 同步模式：在 beforeunload 中使用 navigator.sendBeacon 或同步 XHR
  // 但由于我们的桥接不支持同步调用，这里尝试最后一次异步 flush
  try {
    await flushDirty()
  } catch (e) {
    console.warn('[localStorageProxy] 关闭前 flush 失败:', e)
  }
}

// ———— 工具函数 ————

/**
 * 检查当前 localStorage 是否已被代理接管。
 */
export function isLocalStorageProxied() {
  try {
    return window.localStorage && window.localStorage.__isDesktopProxy === true
  } catch {
    return false
  }
}

/**
 * 获取当前存储模式。
 */
export function getStorageMode() {
  if (_desktopMode) return 'desktop-encrypted'
  return 'browser-localstorage'
}

/**
 * 手动触发一次 flush。
 */
export async function flushStorage() {
  if (_desktopMode) {
    await flushDirty()
  }
}

/**
 * 获取当前内存缓存中所有数据（用于调试）。
 */
export function getCacheSnapshot() {
  if (!_cache) return {}
  return Object.fromEntries(_cache)
}
