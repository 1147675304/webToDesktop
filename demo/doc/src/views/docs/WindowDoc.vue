<template>
  <div>
    <h2><el-icon><Monitor /></el-icon> 窗口控制</h2>
    <p class="desc">无边框窗口下通过 Go Bridge 调用原生平台 API 控制窗口行为。支持拖拽、缩放、最小化、最大化、全屏、关闭、重启。</p>

    <h3>窗口透明背景（跨平台）</h3>
    <p class="desc">配置 <code>webview_bg_transparent: true</code> 启用 WebView 背景透明：</p>
    <ul class="feature-list">
      <li><strong>Linux</strong>：通过 RGBA visual + WebKit WebView 透明背景实现，需合成器支持</li>
      <li><strong>Windows</strong>：通过 <code>WEBVIEW2_DEFAULT_BACKGROUND_COLOR=00000000</code> 实现，配合 Acrylic 毛玻璃效果</li>
      <li>前端需设置 <code>html { background: transparent; }</code> 配合</li>
    </ul>

    <h3>透明区域点击穿透（跨平台）</h3>
    <p class="desc">配置 <code>input_passthrough: true</code> 允许鼠标穿过透明区域操作下层窗口：</p>

    <h4>Windows（WebView2）</h4>
    <ul class="feature-list">
      <li>通过 <strong>WS_EX_TRANSPARENT</strong> 窗口扩展样式实现</li>
      <li>JS 每帧通过 <code>bridge.getCursorPos</code> 获取鼠标坐标</li>
      <li>逐像素检测透明度（Canvas <code>getImageData</code> + CSS <code>elementFromPoint</code>）</li>
      <li>100ms 防抖后切换窗口穿透状态</li>
      <li>仅 <code>initBridge({inputPassthrough: true})</code> 的页面启用</li>
      <li>启动器页面用 <code>false</code> 禁用，正常交互</li>
    </ul>

    <h4>Linux（GTK/WebKit）</h4>
    <ul class="feature-list">
      <li>通过 <strong>input shape</strong> 机制实现</li>
      <li>默认穿透，调用 <code>setInputShapeFull</code> 时捕获所有点击</li>
    </ul>

    <CodeBlock lang="yaml">window:
  webview_bg_transparent: true
  input_passthrough: true  # 跨平台支持</CodeBlock>

    <h4>前端页面配置</h4>
    <CodeBlock lang="javascript">// 效果页面（纯视觉叠加层）：启用穿透检测
initBridge({ inputPassthrough: true, ready: startAnimation });

// 启动器页面（有交互元素）：禁用穿透
initBridge({ inputPassthrough: false });</CodeBlock>

    <h3>系统托盘（跨平台）</h3>
    <p class="desc">配置 <code>system_tray: true</code> 启用系统托盘模式，关闭窗口时隐藏到托盘图标而非退出程序。</p>

    <h4>Windows</h4>
    <ul class="feature-list">
      <li>通过 <strong>Shell_NotifyIconW</strong> + <strong>WS_EX_TOOLWINDOW</strong> 实现</li>
      <li>托盘图标右键菜单：显示窗口 / 退出</li>
      <li>双击托盘图标恢复窗口</li>
      <li><code>tray_hide_taskbar: true</code> 可隐藏任务栏图标（仅托盘可见）</li>
    </ul>

    <h4>Linux</h4>
    <ul class="feature-list">
      <li>通过 <strong>GtkStatusIcon</strong> 实现（GTK3，X11 桌面）</li>
      <li>右键菜单：显示窗口 / 退出</li>
      <li>拦截 <code>delete-event</code> 实现关闭到托盘</li>
    </ul>

    <CodeBlock lang="yaml">window:
  system_tray: true         # 启用托盘模式
  tray_hide_taskbar: true   # 托盘模式下隐藏任务栏图标</CodeBlock>

    <h4>前端快捷键</h4>
    <CodeBlock lang="javascript">// Alt+W 关闭到托盘，Alt+E 从托盘恢复
window.__lhpanda__('registerShortcut', { keys: ['Alt+W', 'Alt+E'] });

window.addEventListener('keyboard-shortcut', function(e) {
  if (e.detail.key === 'Alt+W') window.__lhpanda__('closeWindow', {});
  if (e.detail.key === 'Alt+E') window.__lhpanda__('showWindow', {});
});</CodeBlock>

    <h3>前端示例代码</h3>
    <p class="hint">拖拽条 + 窗口控制按钮的完整实现，使用 Element Plus 图标。</p>

    <CodeBlock>&lt;template&gt;
  &lt;!-- 拖拽条（Windows 用 -webkit-app-region: drag，Linux 用 mousedown bridge） --&gt;
  &lt;div class="drag-bar"
    :class="{ 'drag-native': isWindows }"
    @mousedown="onDragBarMouseDown"&gt;

    &lt;span class="drag-title"&gt;{{ title }}&lt;/span&gt;

    &lt;!-- 窗口控制按钮 — 必须在 drag-bar 内且添加 no-drag --&gt;
    &lt;div class="win-controls no-drag"&gt;
      &lt;el-button :icon="Minus" circle size="small"
        @click="toggleMinimize" title="最小化" /&gt;
      &lt;el-button :icon="FullScreen" circle size="small"
        @click="toggleMaximize" title="最大化" /&gt;
      &lt;el-button :icon="RefreshRight" circle size="small"
        @click="restartApp" title="重启" /&gt;
      &lt;el-button :icon="Close" circle size="small"
        @click="closeWindow" title="关闭"
        style="--el-button-hover-bg-color: #f56c6c" /&gt;
    &lt;/div&gt;
  &lt;/div&gt;

  &lt;!-- Resize 手柄 — 边缘 6px 区域可拖拽缩放 --&gt;
  &lt;div class="resize-top"    @mousedown="resize(12)"&gt;&lt;/div&gt;
  &lt;div class="resize-right"  @mousedown="resize(11)"&gt;&lt;/div&gt;
  &lt;div class="resize-bottom" @mousedown="resize(15)"&gt;&lt;/div&gt;
  &lt;div class="resize-left"   @mousedown="resize(10)"&gt;&lt;/div&gt;
  &lt;div class="resize-tl" @mousedown="resize(13)"&gt;&lt;/div&gt;
  &lt;div class="resize-tr" @mousedown="resize(14)"&gt;&lt;/div&gt;
  &lt;div class="resize-bl" @mousedown="resize(16)"&gt;&lt;/div&gt;
  &lt;div class="resize-br" @mousedown="resize(17)"&gt;&lt;/div&gt;
&lt;/template&gt;

&lt;script setup&gt;
import { computed } from 'vue'
import { Minus, FullScreen, Close, RefreshRight } from '@element-plus/icons-vue'

// ★ 调用 Go Bridge
const call = (m, p = {}) =&gt; window.__lhpanda__(m, p)

const isWindows = computed(() =&gt; /* from bridge getAppInfo */ false)

function onDragBarMouseDown(e) {
  const t = e.target
  // 不拦截按钮/输入框等交互元素
  if (t.closest('button,a,input,select,textarea,.no-drag')) return
  if (!isWindows.value) call('dragWindow')
}

function toggleMinimize()  { call('toggleMinimize') }
function toggleMaximize()  { call('toggleMaximize') }
function restartApp()      { call('restartApp') }
function closeWindow()     { call('closeWindow') }
function resize(edge)      { call('resizeWindow', { edge }) }
&lt;/script&gt;

&lt;style&gt;
.drag-bar {
  height: 36px; display: flex; align-items: center; width: 100%;
  padding: 0 8px; background: #f0f0f0;
  cursor: grab; user-select: none; gap: 12px;
}
.drag-bar.drag-native { -webkit-app-region: drag; }
.drag-bar .no-drag,
.drag-bar button { -webkit-app-region: no-drag; }
.win-controls { margin-left: auto; display: flex; gap: 4px; flex-shrink: 0; }

.resize-top,.resize-bottom { position:fixed; left:0;right:0; height:6px; z-index:200 }
.resize-left,.resize-right  { position:fixed; top:0;bottom:0; width:6px; z-index:200 }
.resize-top { top:0; cursor:n-resize }
.resize-bottom { bottom:0; cursor:s-resize }
.resize-left { left:0; cursor:w-resize }
.resize-right { right:0; cursor:e-resize }
.resize-tl,.resize-tr,.resize-bl,.resize-br {
  position:fixed; width:12px; height:12px; z-index:201
}
.resize-tl { top:0; left:0; cursor:nw-resize }
.resize-tr { top:0; right:0; cursor:ne-resize }
.resize-bl { bottom:0; left:0; cursor:sw-resize }
.resize-br { bottom:0; right:0; cursor:se-resize }
&lt;/style&gt;</CodeBlock>

    <h3>可用 API</h3>
    <table class="api-table">
      <thead><tr><th>方法</th><th>参数</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td><code>dragWindow</code></td><td>—</td><td>触发窗口拖拽</td></tr>
        <tr><td><code>resizeWindow</code></td><td><code>edge: 10~17</code></td><td>从指定边角缩放<br/>10左 11右 12上 13左上 14右上<br/>15下 16左下 17右下</td></tr>
        <tr><td><code>toggleMinimize</code></td><td>—</td><td>最小化到任务栏</td></tr>
        <tr><td><code>toggleMaximize</code></td><td>—</td><td>最大化 / 还原</td></tr>
        <tr><td><code>toggleFullscreen</code></td><td>—</td><td>全屏 / 退出全屏</td></tr>
        <tr><td><code>showWindow</code></td><td>—</td><td>显示并置前窗口（从托盘恢复）</td></tr>
        <tr><td><code>restartApp</code></td><td>—</td><td>重启应用程序</td></tr>
        <tr><td><code>closeWindow</code></td><td>—</td><td>关闭窗口</td></tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { Monitor } from '@element-plus/icons-vue'
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.desc code { background: #eee; padding: 1px 6px; border-radius: 4px; font-size: 12px; }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; }
.feature-list { color: var(--text-secondary); font-size: 13px; line-height: 1.8; margin: 0 0 20px 18px; }
.feature-list li { margin-bottom: 4px; }
.feature-list strong { color: var(--text); }

.api-table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 12px; }
.api-table th, .api-table td { padding: 8px 10px; text-align: left; border-bottom: 1px solid var(--border); }
.api-table th { color: var(--text-secondary); font-weight: 500; font-size: 12px; }
.api-table code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }
</style>
