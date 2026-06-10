import { ref, computed, inject } from 'vue'
import { getStorageMode, flushStorage, isLocalStorageProxied } from './localStorageProxy.js'

/**
 * useStorage — 统一键值对持久化存储 composable
 *
 * 底层已由 localStorageProxy 接管，在桌面模式下透明地将 localStorage 调用
 * 路由到 Go 后端的 AES-256-GCM 加密存储。因此本层只是对标准 localStorage API
 * 的轻量封装，提供响应式绑定等便利功能。
 *
 * 无论桌面端还是浏览器端，都使用完全相同的 API：
 *
 *   import { useStorage } from '../composables/useStorage.js'
 *
 *   const storage = useStorage()
 *   storage.setItem('theme', 'dark')          // 同步 API（底层自动异步持久化）
 *   const theme = storage.getItem('theme')    // 'dark'
 *   storage.removeItem('theme')
 *   storage.clear()
 *
 *   // 响应式绑定（自动同步）
 *   const themeRef = storage.useRef('theme', 'default-dark')
 *   themeRef.value = 'light'  // 自动持久化 + 响应式更新
 */

export function useStorage() {
  const isDesktop = inject('isDesktop', ref(false))

  // ———— 运行时检测 ————
  const backend = computed(() => {
    if (isLocalStorageProxied()) return 'desktop-encrypted'
    return 'browser-localstorage'
  })

  const isEncrypted = computed(() => backend.value === 'desktop-encrypted')

  // ———— 基础 API：直接使用标准的 localStorage（已被 proxy 接管） ————

  function getItem(key) {
    try {
      return localStorage.getItem(key)
    } catch { return null }
  }

  function setItem(key, value) {
    try {
      localStorage.setItem(key, String(value))
    } catch (e) {
      console.warn('[storage] setItem 失败:', e)
    }
  }

  function removeItem(key) {
    try {
      localStorage.removeItem(key)
    } catch { /* ignore */ }
  }

  function clear() {
    try {
      localStorage.clear()
    } catch { /* ignore */ }
  }

  function getAll() {
    try {
      const result = {}
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i)
        if (key !== null) {
          result[key] = localStorage.getItem(key)
        }
      }
      return result
    } catch { return {} }
  }

  function getLength() {
    try {
      return localStorage.length
    } catch { return 0 }
  }

  function key(index) {
    try {
      return localStorage.key(index)
    } catch { return null }
  }

  /**
   * 创建一个与存储自动同步的响应式 ref。
   *
   * 当 ref.value 改变时，自动持久化到存储后端（通过已接管的 localStorage）。
   *
   * @param {string} key           存储键名
   * @param {*}      defaultValue  默认值（当存储中不存在时使用）
   * @returns {Ref<string>} 响应式 ref
   */
  function useRef(key, defaultValue = '') {
    const cache = ref(defaultValue)

    // 从 localStorage 读取初始值
    try {
      const val = localStorage.getItem(key)
      if (val !== null) {
        cache.value = val
      }
    } catch { /* ignore */ }

    // 自动监听变化并持久化
    let syncing = false
    const proxy = new Proxy(cache, {
      set(target, prop, value) {
        const result = Reflect.set(target, prop, value)
        if (prop === 'value' && !syncing) {
          syncing = true
          try {
            localStorage.setItem(key, String(value))
          } catch (e) {
            console.warn('[storage] useRef 持久化失败:', e)
          }
          syncing = false
        }
        return result
      },
      get(target, prop) {
        return Reflect.get(target, prop)
      }
    })

    return proxy
  }

  /**
   * 返回当前存储后端信息。
   */
  function getInfo() {
    return {
      backend: backend.value,
      encrypted: isEncrypted.value,
      proxied: isLocalStorageProxied(),
      length: getLength(),
    }
  }

  return {
    // 标准 API（与 localStorage 一致，同步）
    getItem,
    setItem,
    removeItem,
    clear,
    getAll,
    key,
    get length() { return getLength() },
    // 响应式绑定
    useRef,
    // 信息
    getInfo,
    backend,
    isEncrypted,
  }
}
