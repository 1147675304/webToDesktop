<template>
  <div class="preferences-page">
    <h1>⚙️ 偏好设置</h1>
    <p class="subtitle">配置窗口外观和行为，修改后需<strong>重启应用</strong>生效</p>

    <div class="pref-grid">
      <!-- 基本设置 -->
      <section class="card">
        <h2>📐 基本设置</h2>

        <div class="form-group">
          <label>窗口标题</label>
          <input v-model="form.title" placeholder="窗口标题" />
        </div>

        <div class="form-row">
          <div class="form-group" style="flex:1">
            <label>宽度 (px)</label>
            <input v-model.number="form.width" type="number" min="300" />
          </div>
          <div class="form-group" style="flex:1">
            <label>高度 (px)</label>
            <input v-model.number="form.height" type="number" min="200" />
          </div>
        </div>

        <div class="form-group">
          <label>窗口位置</label>
          <select v-model="form.window_position">
            <option value="">由系统决定</option>
            <option value="center">屏幕居中</option>
            <option value="0,0">左上角 (0,0)</option>
          </select>
        </div>

        <div class="form-group">
          <label>不透明度: {{ (form.opacity * 100).toFixed(0) }}%</label>
          <input v-model.number="form.opacity" type="range" min="0.1" max="1" step="0.01" />
        </div>
      </section>

      <!-- 窗口状态 -->
      <section class="card">
        <h2>🪟 窗口状态</h2>

        <div class="checkbox-group">
          <label class="checkbox-label">
            <input v-model="form.maximized" type="checkbox" :disabled="form.fullscreen" />
            <span>启动时最大化</span>
          </label>
          <label class="checkbox-label">
            <input v-model="form.fullscreen" type="checkbox" :disabled="form.maximized" />
            <span>启动时全屏</span>
          </label>
          <p class="hint">（全屏和最大化互斥，全屏优先级更高）</p>
        </div>

        <div class="checkbox-group">
          <label class="checkbox-label">
            <input v-model="form.borderless" type="checkbox" />
            <span>无边框窗口</span>
          </label>
          <label class="checkbox-label">
            <input v-model="form.always_on_top" type="checkbox" />
            <span>始终置顶</span>
          </label>
        </div>
      </section>

      <!-- Windows 专属 -->
      <section class="card">
        <h2>🪟 Windows 专属效果</h2>

        <div class="checkbox-group">
          <label class="checkbox-label">
            <input v-model="form.acrylic" type="checkbox" />
            <span>Acrylic 毛玻璃背景</span>
          </label>
          <label class="checkbox-label">
            <input v-model="form.webview_bg_transparent" type="checkbox" />
            <span>WebView 背景透明</span>
          </label>
          <p class="hint">（启用毛玻璃时自动开启背景透明）</p>
        </div>

        <div class="checkbox-group">
          <label class="checkbox-label">
            <input v-model="form.dark_title_bar" type="checkbox" />
            <span>暗色标题栏 (Win10 1809+)</span>
          </label>
          <label class="checkbox-label">
            <input v-model="form.round_corners" type="checkbox" />
            <span>圆角窗口 (Win11)</span>
          </label>
        </div>
      </section>

      <!-- 操作按钮 -->
      <div class="actions">
        <button class="btn primary" @click="saveConfig" :disabled="saving">
          {{ saving ? '保存中...' : '💾 保存配置' }}
        </button>
        <button class="btn" @click="resetForm">↩ 重置为当前值</button>
        <button class="btn" @click="loadDefaults">🔄 恢复默认值</button>
      </div>

      <div v-if="saveMsg" class="save-msg" :class="saveMsgType">{{ saveMsg }}</div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, inject } from 'vue'

const bridge = inject('bridge', {})

const form = reactive({
  title: '',
  width: 1024, height: 768,
  fullscreen: false, maximized: false,
  borderless: true, always_on_top: false,
  opacity: 1, webview_bg_transparent: false,
  window_position: 'center',
  dark_title_bar: false, round_corners: true, acrylic: false,
})

const defaults = { ...form }
const saving = ref(false)
const saveMsg = ref('')
const saveMsgType = ref('ok')

onMounted(async () => {
  try {
    const cfg = await bridge.getWindowConfig()
    if (cfg) Object.assign(form, cfg)
    Object.assign(defaults, form)
  } catch (e) {
    console.warn('加载窗口配置失败:', e)
  }
})

async function saveConfig() {
  saving.value = true
  saveMsg.value = ''
  const data = { ...form }
  console.log('[PREF] saving config:', JSON.stringify(data, null, 2))
  try {
    const result = await bridge.saveWindowConfig(data)
    console.log('[PREF] save result:', result)
    saveMsg.value = result.needRestart ? '✅ 配置已保存，重启应用后生效' : '✅ 配置已保存'
    saveMsgType.value = 'ok'
    Object.assign(defaults, form)
  } catch (e) {
    console.error('[PREF] save failed:', e)
    saveMsg.value = '❌ 保存失败: ' + (e?.message || e)
    saveMsgType.value = 'err'
  }
  saving.value = false
}

function resetForm() {
  Object.assign(form, defaults)
}

function loadDefaults() {
  Object.assign(form, {
    title: 'WebToDesktop',
    width: 1024,
    height: 768,
    fullscreen: false,
    maximized: false,
    borderless: true,
    always_on_top: false,
    opacity: 1,
    webview_bg_transparent: false,
    window_position: 'center',
    dark_title_bar: false,
    round_corners: true,
    acrylic: false,
  })
}
</script>

<style scoped>
.preferences-page {
  max-width: 800px; margin: 0 auto;
}
.preferences-page h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { font-size: 13px; color: var(--text-secondary); margin-bottom: 24px; }
.subtitle strong { color: var(--orange); }

.pref-grid { display: flex; flex-direction: column; gap: 16px; }

.card {
  background: var(--card-bg); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 20px;
}
.card h2 { font-size: 16px; margin-bottom: 14px; }

.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 4px; }
.form-group input[type="text"],
.form-group input[type="number"],
.form-group select {
  width: 100%; padding: 8px 12px; border-radius: 6px;
  border: 1px solid var(--border); background: #eee; color: var(--text);
  font-size: 14px; outline: none;
}
.form-group input:focus,
.form-group select:focus { border-color: var(--accent); }
.form-group input[type="range"] {
  width: 100%; accent-color: var(--accent);
}

.form-row { display: flex; gap: 12px; }

.checkbox-group { display: flex; flex-direction: column; gap: 8px; margin-bottom: 8px; }
.checkbox-label {
  display: flex; align-items: center; gap: 8px; font-size: 14px; cursor: pointer;
}
.checkbox-label input[type="checkbox"] {
  width: 16px; height: 16px; accent-color: var(--accent);
}
.hint { font-size: 12px; color: var(--text-secondary); margin-top: 2px; }

.actions { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 8px; }

.btn {
  padding: 8px 16px; border-radius: 6px; border: 1px solid var(--border);
  background: var(--card-bg); color: var(--text); font-size: 13px;
  cursor: pointer; transition: all 0.15s;
}
.btn:hover { background: #e0e0e0; border-color: #555; }
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn.primary { background: var(--accent-dim); border-color: var(--accent); color: var(--accent); }
.btn.primary:hover { background: #00000015; }

.save-msg { margin-top: 12px; padding: 10px 14px; border-radius: 6px; font-size: 13px; }
.save-msg.ok { background: #e8e8e8; color: var(--text); }
.save-msg.err { background: #3a1a1a; color: var(--red); }
</style>
