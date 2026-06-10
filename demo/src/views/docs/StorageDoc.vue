<template>
  <div>
    <h2><el-icon><FolderOpened /></el-icon> 键值对持久化存储</h2>
    <p class="desc">
      提供类似浏览器 <code>localStorage</code> 的键值对存储能力，桌面端数据经 AES-256-GCM 加密后持久化到本地磁盘。
      <strong>已透明接管浏览器原生 <code>localStorage</code> API</strong>，前端代码无需任何改造即可使用。
    </p>

    <h3>无缝切换 — 透明接管 localStorage</h3>
    <p class="hint">桌面端启动时自动用 Proxy 替换 <code>window.localStorage</code>，所有读写操作同步到内存缓存，异步批量持久化到 Go 后端加密文件。浏览器模式下保持原生 <code>localStorage</code> 不变。</p>
    <div class="arch-flow">
      <div class="a-step"><strong>前端代码</strong><br><code>localStorage.setItem('theme', 'dark')</code></div>
      <div class="a-arrow">→</div>
      <div class="a-step bright"><strong>localStorage Proxy</strong><br>内存缓存 + 异步批量 flush</div>
      <div class="a-arrow">→</div>
      <div class="a-step"><strong>桥接层</strong><br><code>window.__lhpanda__</code></div>
      <div class="a-arrow">→</div>
      <div class="a-step"><strong>Go 后端</strong><br>AES-256-GCM 加密 → <code>credentials.dat</code></div>
    </div>

    <div class="tip-box">
      <el-icon><InfoFilled /></el-icon>
      <strong>无需学习新 API</strong>：任何使用 <code>localStorage.getItem/setItem</code> 的现有代码在桌面端自动获得加密持久化能力。
    </div>

    <h3>前端使用（标准 localStorage API）</h3>
    <p class="hint">所有代码可直接使用浏览器原生 <code>localStorage</code> API，桌面端自动路由到加密存储：</p>
    <CodeBlock lang="javascript">// ★ 完全兼容标准 localStorage API — 桌面端透明接管，无需任何改动

// 存储
localStorage.setItem('theme', 'dark')
localStorage.setItem('token', 'eyJhbGci...')
localStorage.setItem('preferences', JSON.stringify({ lang: 'zh', fontSize: 14 }))

// 读取
const theme = localStorage.getItem('theme')  // 'dark'

// 删除
localStorage.removeItem('token')

// 清空
localStorage.clear()

// 遍历
for (let i = 0; i &lt; localStorage.length; i++) {
  const key = localStorage.key(i)
  console.log(key, localStorage.getItem(key))
}

// 对象存储
localStorage.setItem('user', JSON.stringify({ id: 1, name: 'admin' }))
const user = JSON.parse(localStorage.getItem('user') || '{}')</CodeBlock>

    <h3>响应式绑定（Vue composable）</h3>
    <p class="hint"><code>useStorage</code> 在 <code>localStorage</code> API 之上提供了 Vue 响应式封装：</p>
    <CodeBlock lang="javascript">import { useStorage } from '../composables/useStorage.js'

const storage = useStorage()

// 标准 API（与 localStorage 一致，同步）
storage.setItem('theme', 'dark')
const theme = storage.getItem('theme')
storage.removeItem('theme')
storage.clear()
const all = storage.getAll()       // 获取所有键值对
const count = storage.length       // 键值对数量
const firstKey = storage.key(0)    // 按索引获取键名

// 响应式绑定（自动持久化）
const themeRef = storage.useRef('theme', 'default-dark')
themeRef.value = 'light'   // 自动写入 localStorage（→ 桌面端加密持久化）

// 运行时信息
const info = storage.getInfo()
// → { backend: 'desktop-encrypted', proxied: true, encrypted: true, length: 3 }</CodeBlock>

    <h3>初始化（在 App.vue 中）</h3>
    <CodeBlock lang="javascript">// App.vue onMounted
import { initLocalStorageProxy } from './composables/localStorageProxy.js'

onMounted(async () => {
  await initAppInfo()

  // 接管 localStorage — 桌面端自动路由到 Go 加密存储
  const result = await initLocalStorageProxy(bridge)
  console.log(`存储模式: ${result.desktop ? '桌面加密' : '浏览器localStorage'}，已加载 ${result.loaded} 项`)
})</CodeBlock>

    <h3>响应式绑定原理</h3>
    <CodeBlock lang="javascript">// useStorage().useRef(key, defaultValue)
// 底层使用 Proxy 拦截 ref.value 的 set 操作，自动调用 localStorage.setItem()

function useRef(key, defaultValue = '') {
  const cache = ref(defaultValue)

  // 初始值从 localStorage 读取
  const val = localStorage.getItem(key)
  if (val !== null) cache.value = val

  // Proxy 拦截 set，自动持久化
  return new Proxy(cache, {
    set(target, prop, value) {
      const result = Reflect.set(target, prop, value)
      if (prop === 'value') {
        localStorage.setItem(key, String(value))
      }
      return result
    },
    get(target, prop) { return Reflect.get(target, prop) }
  })
}</CodeBlock>

    <h3>异步持久化策略</h3>
    <div class="flow">
      <div class="step"><strong>1.</strong> 启动时预加载所有键值对到内存缓存（一次异步调用）</div>
      <div class="step"><strong>2.</strong> 所有读写操作直接操作内存缓存（同步、零延迟）</div>
      <div class="step"><strong>3.</strong> 写入操作标记脏键，300ms 防抖后批量 flush 到 Go 后端</div>
      <div class="step"><strong>4.</strong> 最长等待 2 秒，超时强制 flush</div>
      <div class="step"><strong>5.</strong> 页面关闭前（<code>beforeunload</code>）执行最后一次 flush</div>
      <div class="step"><strong>6.</strong> flush 失败时脏键自动回队，下次重试</div>
    </div>

    <h3>安全说明</h3>
    <table class="api-table">
      <thead><tr><th>项目</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td>加密算法</td><td>AES-256-GCM（与凭证存储共用密钥）</td></tr>
        <tr><td>存储位置</td><td><code>~/.config/&lt;AppName&gt;/credentials.dat</code>（与凭证同文件但不同区域）</td></tr>
        <tr><td>密钥派生</td><td><code>SHA256(config.yaml → security.aes_key)</code></td></tr>
        <tr><td>数据隔离</td><td>键值对与账号密码存储在同一文件的不同 JSON 字段，API 完全隔离</td></tr>
        <tr><td>密码保护</td><td>键值对 API 只能操作 <code>KeyValues</code> 区域，<code>Credentials</code> 区域永不可通过键值对接口访问</td></tr>
        <tr><td>迁移工具</td><td>桌面端启动时自动加载已有数据，首次使用无需手动导入</td></tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { FolderOpened, InfoFilled } from '@element-plus/icons-vue'
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.desc code, .hint code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 12px; }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; }

.tip-box {
  display: flex; align-items: flex-start; gap: 8px;
  padding: 12px 16px; background: #fff3cd; border-radius: 8px;
  font-size: 13px; color: #856404; margin: 16px 0; line-height: 1.6;
}
.tip-box .el-icon { font-size: 18px; flex-shrink: 0; margin-top: 1px; }

.arch-flow {
  display: flex; align-items: center; gap: 8px;
  background: var(--card-bg); padding: 16px; border-radius: 8px;
  margin: 12px 0; flex-wrap: wrap;
}
.a-step {
  text-align: center; font-size: 12px; line-height: 1.6;
  padding: 8px 12px; background: #f0f0f0; border-radius: 6px; flex: 1; min-width: 100px;
}
.a-step.bright { background: #e8f4e8; }
.a-step strong { font-size: 13px; }
.a-step code { font-size: 10px; }
.a-arrow { font-size: 18px; color: var(--text-secondary); }

.flow { display: flex; flex-direction: column; gap: 8px; margin-top: 12px; }
.step { padding: 8px 12px; background: var(--card-bg); border-radius: 6px; font-size: 13px; color: var(--text-secondary); }
.step strong { color: var(--accent); margin-right: 6px; }
.step code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }

.api-table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 12px; }
.api-table th, .api-table td { padding: 8px 10px; text-align: left; border-bottom: 1px solid var(--border); }
.api-table th { color: var(--text-secondary); font-weight: 500; font-size: 12px; }
.api-table code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }
</style>
