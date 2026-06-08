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
import { inject, ref, reactive, onMounted } from 'vue'
import { Monitor, CircleCheckFilled, WarningFilled, Lock, Connection, Delete, Key, User, Sort, Cloudy, Box, Tickets, Crop, FullScreen, Close, Pointer, InfoFilled, UploadFilled } from '@element-plus/icons-vue'

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

onMounted(async () => {
  try {
    const cfg = await bridge.getWindowConfig()
    if (cfg) {
      winCfg.webview_bg_transparent = cfg.webview_bg_transparent || cfg.acrylic
      winCfg.input_passthrough = cfg.input_passthrough
    }
  } catch (e) { /* browser mode */ }
})

async function saveCred() { saveCredFn() }
async function loadCreds() { loadCredsFn() }
async function deleteCred(username) { deleteCredFn(username) }
async function clearCreds() { clearCredsFn() }
async function dragWindow() { bridge.dragWindow().catch(() => {}) }
async function toggleMaximize() { bridge.toggleMaximize().catch(() => {}) }
async function toggleFullscreen() { bridge.toggleFullscreen().catch(() => {}) }
async function closeWindow() { bridge.closeWindow().catch(() => {}) }

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
