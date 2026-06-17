<template>
  <div>
    <h2><el-icon><Key /></el-icon> 键盘快捷键与按键映射</h2>
    <p class="desc">系统提供<strong>三层</strong>按键控制机制，职责分离、互不干扰：</p>
    <ul class="note-list">
      <li><strong>浏览器快捷键禁用</strong>（JS 注入）：禁用 WebView 内置浏览器行为（Ctrl+S 保存网页等），仅影响当前应用</li>
      <li><strong>按键映射</strong>（原生钩子，粗粒度）：拦截单个修饰键，一次性禁用该键的所有系统快捷键</li>
      <li><strong>组合快捷键注册</strong>（原生钩子，细粒度）：拦截特定组合键，由应用消费不传递到系统</li>
    </ul>

    <!-- ==================== 配置 ==================== -->
    <h3>配置</h3>
    <p class="hint">在 <code>config.yaml</code> 中配置（构建时嵌入，修改需重新构建）：</p>
    <CodeBlock lang="yaml">window:
  # 键盘钩子总开关（关闭后按键映射和组合快捷键注册均失效）
  keyboard_shortcuts: true

  # 禁用浏览器内置快捷键（JS 注入，与键盘钩子无关）
  # 即使 keyboard_shortcuts: false，此项仍生效
  # 禁用列表：Ctrl+S（保存网页）、Ctrl+P（打印）、Ctrl+U（查看源码）、F12（开发者工具）
  disable_browser_shortcuts: true

  # 默认拦截的快捷键列表（钩子安装时自动注册，仅 keyboard_shortcuts=true 时生效）
  # 设为 [] 则不拦截任何默认快捷键
  default_blocked_shortcuts:
    - "Ctrl+S"

  # 按键映射由前端通过 bridge.setKeyMapping 动态管理
  key_mappings: {}</CodeBlock>

    <!-- ==================== ① 浏览器快捷键禁用 ==================== -->
    <h3>① 浏览器快捷键禁用（JS 注入）</h3>
    <p class="hint">通过 JS 注入禁用 WebView 内置浏览器快捷键，<strong>仅影响当前应用</strong>，不影响系统其他应用。与键盘钩子无关，即使 <code>keyboard_shortcuts: false</code> 仍可生效。</p>
    <table class="api-table">
      <thead><tr><th>快捷键</th><th>浏览器默认行为</th><th>禁用后效果</th></tr></thead>
      <tbody>
        <tr><td><code>Ctrl+S</code></td><td>保存网页为 HTML</td><td>不弹出保存对话框</td></tr>
        <tr><td><code>Ctrl+P</code></td><td>打印网页</td><td>不弹出打印对话框</td></tr>
        <tr><td><code>Ctrl+U</code></td><td>查看网页源码</td><td>不跳转源码页</td></tr>
        <tr><td><code>F12</code></td><td>打开开发者工具</td><td>不打开 DevTools</td></tr>
      </tbody>
    </table>
    <p class="hint">配置方式：<code>config.yaml</code> → <code>disable_browser_shortcuts: true/false</code>（构建时配置）。前端如需自定义保存/打印逻辑，可自行监听 <code>keydown</code> 事件（注入的 <code>preventDefault</code> 仅阻止浏览器默认行为，不阻止前端自定义处理）。</p>

    <!-- ==================== ②③ 原生钩子拦截 ==================== -->
    <h3>②③ 原生钩子拦截（需 keyboard_shortcuts: true）</h3>
    <p class="hint">以下两种机制依赖底层键盘钩子（Windows <code>WH_KEYBOARD_LL</code> / Linux GTK <code>key-press-event</code>），<strong>全局拦截</strong>指定按键使其不传递到系统。</p>

    <h4>① 按键映射（Key Mapping）</h4>
    <p class="hint">将修饰键（<code>Super_L</code>、<code>Alt_L</code>、<code>Control_L</code> 等）映射为自定义名称。映射实质是将该按键注册为快捷键（<code>RegisterShortcut</code>），使其被钩子拦截。<strong>所有基于该修饰键的系统快捷键全部失效</strong>。例如映射 <code>Super_L</code> 后，<code>Win+D</code>、<code>Win+R</code>、<code>Win+E</code> 等全部被禁用，无需逐个注册。</p>
    <p class="hint">被映射的键按下时，以<strong>原始键名</strong>（如 <code>"Super_L"</code>）触发 <code>keyboard-shortcut</code> 事件，前端根据映射名自行转换。</p>

    <h4>② 组合快捷键注册（Shortcut Registration）</h4>
    <p class="hint">注册特定组合键（如 <code>Ctrl+S</code>、<code>Alt+W</code>），当该组合被按下时由应用消费，不传递到系统。</p>
    <p class="hint">即使修饰键已被映射，组合键仍可正常工作：<code>GetAsyncKeyState</code>（Windows）或 <code>event->state</code>（Linux）检测的是<strong>物理按键状态</strong>，不受映射拦截影响。</p>

    <h4>默认拦截</h4>
    <p class="hint">默认拦截列表由 <code>config.yaml</code> 中的 <code>default_blocked_shortcuts</code> 定义，构建时嵌入二进制。钩子安装时自动注册这些快捷键。</p>
    <p class="hint"><strong>注意：</strong>Ctrl+S 的浏览器保存行为由 <code>disable_browser_shortcuts</code>（JS 注入）处理，无需在 <code>default_blocked_shortcuts</code> 中配置。仅在需要<strong>全局拦截</strong> Ctrl+S（阻止所有应用接收）时才加入此列表。</p>
    <table class="api-table">
      <thead><tr><th>场景</th><th>配置方式</th><th>影响范围</th></tr></thead>
      <tbody>
        <tr><td>仅禁用 WebView 保存对话框</td><td><code>disable_browser_shortcuts: true</code></td><td>仅当前应用</td></tr>
        <tr><td>全局拦截 Ctrl+S</td><td><code>default_blocked_shortcuts: ["Ctrl+S"]</code></td><td>整个系统</td></tr>
      </tbody>
    </table>

    <!-- ==================== 组合快捷键 API ==================== -->
    <h3>组合快捷键 API</h3>
    <p class="hint">使用 <code>useBridge()</code> 获取 <code>bridge</code> 代理对象后调用：</p>
    <CodeBlock lang="javascript">import { useBridge } from './composables/useBridge.js'

const { bridge } = useBridge()

// 注册自定义快捷键
await bridge.registerShortcut({ keys: ['Ctrl+P', 'Shift+F1'] })
// → { ok: true, registered: ['Ctrl+P', 'Shift+F1'] }

// 注销快捷键
await bridge.unregisterShortcut({ keys: ['Ctrl+P'] })
// → { ok: true }

// 列出所有已注册的快捷键
const list = await bridge.listShortcuts()
// → { ok: true, shortcuts: ['Ctrl+S', 'Shift+F1', ...] }

// 重置为默认（恢复上面表格中的系统快捷键）
await bridge.resetShortcuts()
// → { ok: true }

// 清空所有（包括默认拦截的快捷键）
await bridge.clearShortcuts()
// → { ok: true }</CodeBlock>

    <!-- ==================== 按键映射 API ==================== -->
    <h3>按键映射 API</h3>
    <p class="hint">动态管理按键映射，映射后的按键被钩子拦截并触发事件：</p>
    <CodeBlock lang="javascript">import { useBridge } from './composables/useBridge.js'

const { bridge } = useBridge()

// 设置单个按键映射（空字符串 = 取消映射）
await bridge.setKeyMapping({ key: 'Super_L', mappedName: 'Win' })
// → { ok: true }

// 批量设置按键映射
await bridge.setKeyMapping({
  mappings: {
    'Super_L': 'Win',
    'Alt_L': 'Alt',
    'Control_L': 'Ctrl'
  }
})
// → { ok: true }

// 列出所有按键映射
const result = await bridge.listKeyMappings()
// → { ok: true, mappings: { 'Super_L': 'Win', 'Alt_L': 'Alt', ... } }

// 清空所有按键映射
await bridge.clearKeyMappings()
// → { ok: true }</CodeBlock>

    <!-- ==================== 前端监听 ==================== -->
    <h3>前端监听快捷键事件</h3>
    <p class="hint">无论是按键映射还是组合快捷键，被拦截时都会通过 <code>CustomEvent('keyboard-shortcut')</code> 通知前端：</p>
    <CodeBlock lang="javascript">import { useBridge } from './composables/useBridge.js'

const { bridge, isDesktop } = useBridge()

// 注册快捷键
if (isDesktop) {
  await bridge.registerShortcut({ keys: ['Ctrl+P', 'Alt+W'] })
}

// 获取按键映射表，用于翻译原始键名
const keyMappings = {
  Super_L: 'Win',
  Super_R: 'Win',
  Alt_L: 'Alt',
  Alt_R: 'Alt',
  Control_L: 'Ctrl',
  Control_R: 'Ctrl',
}

// 监听所有按键事件（包括映射和组合）
window.addEventListener('keyboard-shortcut', (e) => {
  const rawKey = e.detail.key
  // 如果是映射的修饰键，转换为映射名
  const key = keyMappings[rawKey] || rawKey

  switch (key) {
    // 组合快捷键
    case 'Ctrl+P':
      handlePrint()
      break
    case 'Alt+W':
      bridge.closeWindow({})
      break
    // 按键映射（单个修饰键）
    case 'Win':
      console.log('Windows 键被按下（已拦截）')
      break
    case 'Alt':
      console.log('Alt 键被按下（已拦截）')
      break
    case 'Ctrl':
      console.log('Ctrl 键被按下（已拦截）')
      break
  }
})</CodeBlock>

    <p class="hint"><code>keyboard-shortcut</code> 事件在 <code>window</code> 上派发。<code>e.detail.key</code> 为被拦截的按键描述：组合快捷键为 <code>"Alt+W"</code> 格式，按键映射为<strong>原始键名</strong>（如 <code>"Super_L"</code>）。前端应维护映射表自行转换。</p>

    <!-- ==================== 快捷键格式 ==================== -->
    <h3>快捷键格式</h3>
    <p class="hint">修饰符顺序：<code>Super</code> &gt; <code>Ctrl</code> &gt; <code>Alt</code> &gt; <code>Shift</code>，键名首字母大写。</p>
    <table class="api-table">
      <thead><tr><th>示例</th><th>说明</th></tr></thead>
      <tbody>
        <tr><td><code>Ctrl+S</code></td><td>Ctrl + S</td></tr>
        <tr><td><code>Ctrl+Shift+F</code></td><td>Ctrl + Shift + F</td></tr>
        <tr><td><code>Alt+W</code></td><td>Alt + W</td></tr>
        <tr><td><code>Alt+Tab</code></td><td>Alt + Tab</td></tr>
        <tr><td><code>Super_L</code></td><td>左 Windows / Command 键</td></tr>
        <tr><td><code>Super_R</code></td><td>右 Windows / Command 键</td></tr>
        <tr><td><code>F1</code> ~ <code>F12</code></td><td>功能键</td></tr>
        <tr><td><code>Space</code></td><td>空格键</td></tr>
        <tr><td><code>Enter</code></td><td>回车键</td></tr>
        <tr><td><code>Esc</code></td><td>ESC 键</td></tr>
        <tr><td><code>Delete</code></td><td>删除键</td></tr>
        <tr><td><code>Home</code> / <code>End</code></td><td>Home / End 键</td></tr>
        <tr><td><code>Left</code> / <code>Right</code> / <code>Up</code> / <code>Down</code></td><td>方向键</td></tr>
        <tr><td><code>PageUp</code> / <code>PageDown</code></td><td>翻页键</td></tr>
      </tbody>
    </table>

    <!-- ==================== 实现原理 ==================== -->
    <h3>实现原理</h3>
    <p class="hint">Windows 使用 <code>WH_KEYBOARD_LL</code> 低层键盘钩子（<code>SetWindowsHookEx</code>），Linux 使用 GTK <code>key-press-event</code> 信号 + X11 <code>XGrabKey</code> 全局抓取。</p>
    <p class="hint">按键映射与组合快捷键共用同一 C 层拦截列表（<code>g_blockedKeys[128]</code>）。<code>setKeyMapping</code> 内部调用 <code>RegisterShortcut</code> 将按键加入此列表，钩子回调按 keycode → 名称 → 查表的流程判断是否拦截。没有独立的 C 层映射表。</p>
    <p class="hint">被拦截的按键通过 C 层事件缓冲区 → Go 轮询协程（50ms）→ <code>window.dispatchEvent(CustomEvent)</code> 推送到前端。</p>
    <p class="hint">为降低杀毒软件误报，敏感 Windows API（<code>SetWindowsHookExA</code>、<code>CallNextHookEx</code>、<code>UnhookWindowsHookEx</code>、<code>GetAsyncKeyState</code>）在运行时通过 <code>GetProcAddress</code> 动态解析，不出现在 PE 导入表中。钩子安装延迟 3~7 秒，避开启动扫描。</p>

    <!-- ==================== 注意事项 ==================== -->
    <h3>注意事项</h3>
    <ul class="note-list">
      <li><strong>按键映射</strong>基于 <code>RegisterShortcut</code> 实现，映射的按键以<strong>原始键名</strong>（如 <code>"Super_L"</code>）触发事件，前端需自行维护名称转换</li>
      <li><strong>组合快捷键</strong>在按键映射之后检查，即使修饰键已被映射，组合键仍可通过物理按键状态检测正常拦截</li>
      <li>按键映射与组合快捷键可并存：映射用于粗粒度禁用整类系统快捷键，组合用于精细控制特定快捷键</li>
      <li><code>Ctrl+Alt+Del</code> 是 Windows 内核安全序列，任何用户态钩子均无法拦截</li>
      <li>Linux Wayland 下无法全局拦截窗口管理器快捷键，仅窗口聚焦时生效</li>
      <li>调用 <code>bridge</code> 方法时需传 2 个参数：方法名 + 参数对象（无参数时传 <code>{}</code>），否则 webview 绑定会报 "arguments mismatch"</li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { Key } from '@element-plus/icons-vue'
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; }

.api-table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 12px; }
.api-table th, .api-table td { padding: 8px 10px; text-align: left; border-bottom: 1px solid var(--border); }
.api-table th { color: var(--text-secondary); font-weight: 500; font-size: 12px; }
.api-table code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }

.note-list { padding-left: 20px; font-size: 13px; color: var(--text-secondary); line-height: 2; }
</style>
