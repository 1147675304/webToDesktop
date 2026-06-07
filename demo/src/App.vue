<template>
  <div class="app">
    <!-- ====== 拖拽条 ====== -->
    <div class="drag-bar" :class="{ 'drag-native': isWindows }" @mousedown="onDragBarMousedown">
      <!-- 菜单按钮（no-drag 区域，不触发窗口拖拽） -->
      <div class="menubar">
        <div
          class="menu-item no-drag"
          v-for="m in menus"
          :key="m.label"
          @click.stop="m.action?.()"
        >
          <span class="menu-icon" v-if="m.icon">{{ m.icon }}</span>
          <span class="menu-label">{{ m.label }}</span>
        </div>
      </div>
      <span class="drag-title">{{ appInfo.platform === 'browser' ? 'WebToDesktop 演示' : 'WebToDesktop' }}</span>
      <!-- 窗口控制按钮（拖拽条内部 no-drag 区域） -->
      <div class="win-controls no-drag">
        <button title="最小化" @click="doMinimize">─</button>
        <button title="最大化" @click="doMaximize">🗖</button>
        <button title="全屏" @click="doFullscreen">⛶</button>
        <button title="重启" @click="doRestart">↻</button>
        <button title="关闭" @click="doClose" class="btn-close">✕</button>
      </div>
    </div>

    <!-- ====== Resize 手柄 ====== -->
    <div class="resize-top"    @mousedown="doResize(12)"></div>
    <div class="resize-right"  @mousedown="doResize(11)"></div>
    <div class="resize-bottom" @mousedown="doResize(15)"></div>
    <div class="resize-left"   @mousedown="doResize(10)"></div>
    <div class="resize-tl" @mousedown="doResize(13)"></div>
    <div class="resize-tr" @mousedown="doResize(14)"></div>
    <div class="resize-bl" @mousedown="doResize(16)"></div>
    <div class="resize-br" @mousedown="doResize(17)"></div>

    <!-- ====== 主内容区 ====== -->
    <div class="content">
      <router-view />
    </div>
  </div>
</template>

<script setup>
import { provide, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useBridge } from './composables/useBridge.js'
import { useCredentials } from './composables/useCredentials.js'

const router = useRouter()

// ———— 使用 composables（共享状态） ————
const { bridge, appInfo, bridgeReady, isDesktop, isWindows, initAppInfo, onDragBarMouseDown } = useBridge()
const { credList, credForm, loadCreds, saveCred, deleteCred, clearCreds } = useCredentials()

// ———— 菜单系统 ————
const menus = [
  { label: '偏好设置', icon: '⚙️', action: () => router.push('/preferences') },
  { label: '文档', icon: '📖', action: () => router.push('/docs') },
]

// ———— 初始化 ————
onMounted(async () => {
  await initAppInfo()
  loadCreds()
})

// ———— 拖拽条事件 ————
function onDragBarMousedown(e) {
  onDragBarMouseDown(e, isWindows.value)
}

// ———— 窗口控制（包装 bridge 调用，加错误处理） ————
function doDragWindow()    { bridge.dragWindow().catch(() => {}) }
function doMinimize()      { bridge.toggleMinimize().catch(() => {}) }
function doMaximize()      { bridge.toggleMaximize().catch(() => {}) }
function doFullscreen()    { bridge.toggleFullscreen().catch(() => {}) }
function doClose()         { bridge.closeWindow().catch(() => {}) }
function doRestart()       { bridge.restartApp().catch(() => {}) }
function doResize(edge)    { bridge.resizeWindow({ edge }).catch(() => {}) }

// ———— 向子页面提供共享状态 ————
provide('bridge', bridge)
provide('appInfo', appInfo)
provide('bridgeReady', bridgeReady)
provide('isDesktop', isDesktop)
provide('credList', credList)
provide('credForm', credForm)
provide('saveCred', saveCred)
provide('loadCreds', loadCreds)
provide('deleteCred', deleteCred)
provide('clearCreds', clearCreds)
</script>

<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

html { background: #ffffff; }

:root {
  --bg: #ffffff; --card-bg: #f5f5f5; --border: #e0e0e0;
  --text: #1a1a1a; --text-secondary: #666666;
  --accent: #333333; --accent-dim: #00000015;
  --green: #555555; --orange: #777777; --red: #444444;
  --radius: 12px;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans SC', sans-serif;
  background: var(--bg); color: var(--text); line-height: 1.6;
  min-height: 100vh; overflow: hidden;
}

.drag-bar {
  height: 36px; display: flex; align-items: center;
  width: 100%;
  padding: 0 8px; background: #f0f0f0; border-bottom: 1px solid var(--border);
  cursor: grab; user-select: none;
  position: sticky; top: 0; z-index: 100;
  gap: 12px;
}
.drag-bar.drag-native {
  -webkit-app-region: drag;
}
/* 拖拽条内的交互元素不参与拖拽，恢复可点击 */
.drag-bar .no-drag,
.drag-bar button,
.drag-bar a,
.drag-bar input,
.drag-bar .menubar {
  -webkit-app-region: no-drag;
}
.drag-bar:active { cursor: grabbing; }
.drag-title { font-size: 13px; color: var(--text-secondary); font-weight: 500; flex-shrink: 0; }

/* ———— 菜单栏（拖拽条内 no-drag 区域） ———— */
.menubar {
  display: flex; gap: 2px; flex-shrink: 0; height: 100%; align-items: center;
}
.menu-item {
  position: relative;
  padding: 0 10px; height: 28px; display: flex; align-items: center;
  border-radius: 6px; cursor: pointer; font-size: 13px;
  color: var(--text-secondary);
  transition: background 0.12s, color 0.12s;
  white-space: nowrap;
}
.menu-item:hover { background: #e8e8e8; color: var(--text); }
.menu-icon { font-size: 14px; flex-shrink: 0; }
.menu-label { user-select: none; }

/* ———— 拖拽条标题 ———— */
.win-controls {
  display: flex; gap: 4px; margin-left: auto; flex-shrink: 0;
}
.win-controls button {
  width: 32px; height: 28px; border: none; border-radius: 6px;
  background: transparent; color: var(--text-secondary);
  font-size: 14px; cursor: pointer; display: flex; align-items: center; justify-content: center;
  transition: background 0.15s;
}
.win-controls button:hover { background: #e0e0e0; color: var(--text); }
.win-controls .btn-close:hover { background: var(--red); color: #fff; }

.resize-top, .resize-bottom { position: fixed; left: 0; right: 0; height: 6px; z-index: 200; }
.resize-left, .resize-right { position: fixed; top: 0; bottom: 0; width: 6px; z-index: 200; }
.resize-top { top: 0; cursor: n-resize; }
.resize-bottom { bottom: 0; cursor: s-resize; }
.resize-left { left: 0; cursor: w-resize; }
.resize-right { right: 0; cursor: e-resize; }
.resize-tl, .resize-tr, .resize-bl, .resize-br {
  position: fixed; width: 12px; height: 12px; z-index: 201;
}
.resize-tl { top: 0; left: 0; cursor: nw-resize; }
.resize-tr { top: 0; right: 0; cursor: ne-resize; }
.resize-bl { bottom: 0; left: 0; cursor: sw-resize; }
.resize-br { bottom: 0; right: 0; cursor: se-resize; }

.content { height: calc(100vh - 36px); overflow-y: auto; padding: 24px; }
.content::-webkit-scrollbar { width: 0; }

.hero {
  text-align: center; padding: 32px 24px 24px;
  border-bottom: 1px solid var(--border); margin-bottom: 20px;
}
.logo { font-size: 56px; margin-bottom: 12px; }
.hero h1 {
  font-size: 32px; font-weight: 700;
  background: linear-gradient(135deg, var(--accent), #a371f7);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  background-clip: text; margin-bottom: 6px;
}
.subtitle { font-size: 16px; color: var(--text-secondary); margin-bottom: 16px; }
.badges { display: flex; gap: 8px; justify-content: center; flex-wrap: wrap; }
.badge { padding: 4px 12px; border-radius: 20px; background: var(--accent-dim); color: var(--accent); font-size: 12px; font-weight: 500; }

.status-bar { display: flex; gap: 12px; margin-bottom: 20px; flex-wrap: wrap; }
.status-item {
  flex: 1; min-width: 200px; padding: 12px 16px;
  background: var(--card-bg); border: 1px solid var(--border); border-radius: 8px;
  display: flex; flex-direction: column; gap: 4px;
}
.status-label { font-size: 12px; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; }
.status-value { font-size: 14px; font-weight: 500; }
.status-value.ok { color: var(--green); }
.status-value.warn { color: var(--orange); }

.demo-grid { max-width: 960px; margin: 0 auto; display: flex; flex-direction: column; gap: 20px; }

.card {
  background: var(--card-bg); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 24px;
}
.card h2 { font-size: 18px; margin-bottom: 6px; }
.card-desc { font-size: 13px; color: var(--text-secondary); margin-bottom: 16px; }

.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 4px; }
.form-group input {
  width: 100%; padding: 8px 12px; border-radius: 6px;
  border: 1px solid var(--border); background: #fff; color: var(--text);
  font-size: 14px; outline: none; transition: border-color 0.2s;
}
.form-group input:focus { border-color: var(--accent); }
.input-row { display: flex; gap: 8px; }

.btn-row { display: flex; gap: 8px; flex-wrap: wrap; margin: 12px 0; }
.btn {
  padding: 8px 16px; border-radius: 6px; border: 1px solid var(--border);
  background: var(--card-bg); color: var(--text); font-size: 13px;
  cursor: pointer; transition: all 0.15s; display: inline-flex; align-items: center; gap: 4px;
}
.btn:hover { background: #e8e8e8; border-color: #bbb; }
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn.primary { background: var(--accent-dim); border-color: var(--accent); color: var(--accent); }
.btn.primary:hover { background: #1f6feb55; }
.btn.danger { color: var(--red); }
.btn.danger:hover { background: #f8514922; border-color: var(--red); }
.btn-sm { padding: 4px 10px; font-size: 12px; border-radius: 4px; }

.cred-list { margin-top: 12px; display: flex; flex-direction: column; gap: 6px; }
.cred-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 12px; background: #f5f5f5; border-radius: 6px; font-size: 14px;
}
.empty-hint { font-size: 13px; color: var(--text-secondary); text-align: center; padding: 20px 0; }

.tip-box {
  margin-top: 12px; padding: 12px 16px;
  background: #f9f9f9; border-left: 3px solid var(--accent); border-radius: 0 8px 8px 0;
  font-size: 13px; color: var(--text-secondary); line-height: 1.7;
}
.tip-box strong { color: var(--text); }
.tip-box code { background: #eee; color: var(--green); padding: 1px 6px; border-radius: 4px; font-family: monospace; font-size: 12px; }

.proxy-flow { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 16px 0; flex-wrap: wrap; }
.flow-step { text-align: center; padding: 12px; background: #f5f5f5; border-radius: 8px; min-width: 110px; }
.flow-icon { font-size: 24px; margin-bottom: 4px; }
.flow-label { font-size: 13px; font-weight: 600; margin-bottom: 2px; }
.flow-detail { font-size: 11px; color: var(--text-secondary); font-family: monospace; }
.flow-arrow { font-size: 20px; color: var(--text-secondary); }

.proxy-rules { margin-top: 16px; }
.proxy-rules h3 { font-size: 14px; margin-bottom: 10px; }
.rule { display: flex; align-items: center; gap: 8px; padding: 6px 0; font-size: 13px; color: var(--text-secondary); }
.rule-num {
  width: 22px; height: 22px; border-radius: 50%;
  background: var(--accent-dim); color: var(--accent);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 700; flex-shrink: 0;
}

.code-block {
  margin-top: 12px; padding: 12px 16px;
  background: #f5f5f5; border: 1px solid var(--border); border-radius: 6px;
  font-family: monospace; font-size: 12px;
  color: var(--text); white-space: pre-wrap; word-break: break-all;
  max-height: 200px; overflow-y: auto;
}

.commands { display: flex; flex-direction: column; gap: 8px; }
.cmd { display: flex; align-items: center; gap: 12px; padding: 6px 0; }
.cmd-code {
  background: #f5f5f5; border: 1px solid var(--border); border-radius: 6px;
  padding: 5px 12px; font-family: monospace; font-size: 12px;
  color: var(--green); white-space: nowrap; flex-shrink: 0; min-width: 280px;
}
.cmd-desc { font-size: 13px; color: var(--text-secondary); }

@media (max-width: 700px) {
  .content { padding: 16px; }
  .hero { padding: 24px 12px; }
  .hero h1 { font-size: 24px; }
  .proxy-flow { gap: 4px; }
  .flow-step { min-width: 80px; padding: 8px; }
  .flow-arrow { font-size: 14px; }
  .cmd-code { min-width: auto; }
}
</style>
