<template>
  <div class="code-block" ref="root">
    <button class="copy-btn" @click="doCopy">{{ copied ? '✓ 已复制' : '📋 复制' }}</button>
    <pre><code><slot /></code></pre>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.min.css'

const props = defineProps<{ lang?: string }>()
const root = ref<HTMLElement | null>(null)
const copied = ref(false)

onMounted(async () => {
  await nextTick()
  const code = root.value?.querySelector('code')
  if (code) {
    if (props.lang) code.classList.add(`language-${props.lang}`)
    hljs.highlightElement(code)
  }
})

async function doCopy() {
  const code = root.value?.querySelector('code')
  if (!code) return
  try {
    await navigator.clipboard.writeText(code.innerText)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = code.innerText
    ta.style.cssText = 'position:fixed;opacity:0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}
</script>

<style scoped>
.code-block {
  position: relative;
  background: #f6f6f6; border: 1px solid var(--border, #e0e0e0);
  border-radius: 8px; margin-bottom: 24px;
  max-height: 460px; overflow-y: auto;
  /* 覆盖 WebView 注入的 user-select: none，允许选择文本 */
  -webkit-user-select: text !important;
  user-select: text !important;
}
.code-block pre {
  margin: 0; padding: 16px 20px;
  font-family: 'Fira Code', 'Cascadia Code', monospace; font-size: 12px;
  line-height: 1.7; color: var(--text, #1a1a1a);
  white-space: pre; overflow-x: auto;
  -webkit-user-select: text !important;
  user-select: text !important;
}
.code-block code {
  font-family: inherit; font-size: inherit;
  white-space: pre !important;
  display: block;
}
.copy-btn {
  position: absolute; top: 6px; right: 8px; z-index: 2;
  padding: 3px 10px; border: 1px solid var(--border, #e0e0e0);
  border-radius: 5px; background: #eee; color: var(--text-secondary, #666);
  font-size: 11px; cursor: pointer; opacity: 0;
  transition: opacity 0.15s, background 0.15s;
}
.code-block:hover .copy-btn { opacity: 1; }
.copy-btn:hover { background: #ddd; color: var(--text, #1a1a1a); }
</style>

<!-- 非 scoped：覆盖 WebView 注入的隐藏滚动条 -->
<style>
.code-block::-webkit-scrollbar {
  width: 6px; height: 6px;
}
.code-block::-webkit-scrollbar-thumb {
  background: rgba(0,0,0,0.15);
  border-radius: 4px;
}
.code-block::-webkit-scrollbar-thumb:hover {
  background: rgba(255,255,255,0.3);
}
.code-block::-webkit-scrollbar-track {
  background: transparent;
}
</style>
