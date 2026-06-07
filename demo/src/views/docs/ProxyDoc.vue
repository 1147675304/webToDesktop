<template>
  <div>
    <h2>🌐 智能网络代理</h2>
    <p class="desc">本地 HTTP 服务拦截匹配前缀的请求，自动注入凭证后转发到远程 API。前端无需关心跨域和鉴权细节，使用标准 <code>fetch</code> 即可。</p>

    <h3>前端示例代码（Plain JS + Bridge Proxy）</h3>
    <p class="hint">使用 <code>useBridge()</code> 获取 bridge 代理，所有 API 调用由代理层自动转发到远程服务器。</p>

    <CodeBlock>&lt;template&gt;
  &lt;div&gt;
    &lt;el-button type="primary" @click="fetchUsers"&gt;
      获取用户列表
    &lt;/el-button&gt;
    &lt;el-button @click="createUser"&gt;
      新建用户
    &lt;/el-button&gt;

    &lt;el-table :data="users" v-loading="loading"&gt;
      &lt;el-table-column prop="id" label="ID" width="80" /&gt;
      &lt;el-table-column prop="name" label="姓名" /&gt;
      &lt;el-table-column prop="email" label="邮箱" /&gt;
    &lt;/el-table&gt;
  &lt;/div&gt;
&lt;/template&gt;

&lt;script setup&gt;
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

const users = ref([])
const loading = ref(false)

// ★ 无需指定完整 URL，使用相对路径即可
// 代理层自动匹配 /api/ 前缀 → 转发到远程 API
async function fetchUsers() {
  loading.value = true
  try {
    const resp = await fetch('/api/users')
    users.value = await resp.json()
  } finally {
    loading.value = false
  }
}

// ★ 创建用户：请求体中用 __DESKTOP_PWD__ 占位密码
// 代理层会自动替换为真实密码（前提：已保存凭据）
async function createUser() {
  await fetch('/api/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: '新用户',
      email: 'user@example.com',
      password: '__DESKTOP_PWD__',  // ← 占位符，代理层自动替换
    }),
  })
  ElMessage.success('创建成功')
  fetchUsers()
}
&lt;/script&gt;</CodeBlock>

    <h3>代理流程</h3>
    <div class="flow">
      <div class="step"><strong>1.</strong> 前端 <code>fetch('/api/users')</code></div>
      <div class="step"><strong>2.</strong> 本地 HTTP 服务接收请求，匹配代理前缀 <code>/api/</code></div>
      <div class="step"><strong>3.</strong> 注入 <code>X-Credential-Username</code> 请求头</div>
      <div class="step"><strong>4.</strong> 替换请求体中的 <code>__DESKTOP_PWD__</code> 为真实密码</div>
      <div class="step"><strong>5.</strong> 转发到远程 API 服务器</div>
      <div class="step"><strong>6.</strong> 返回响应给前端</div>
    </div>

    <h3>配置方式</h3>
    <p class="hint">在 Vue 项目的 <code>.env.production</code> 中配置：</p>
    <CodeBlock>VITE_REMOTE_API_URL=https://api.example.com
VITE_PROXY_PREFIXES=/api/,/storage/</CodeBlock>
  </div>
</template>

<script setup lang="ts">
import CodeBlock from '../../components/CodeBlock.vue'
</script>

<style scoped>
.desc { color: var(--text-secondary); font-size: 14px; margin-bottom: 20px; line-height: 1.7; }
.desc code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 12px; }
.hint { color: var(--text-secondary); font-size: 13px; margin-bottom: 12px; }

.flow { display: flex; flex-direction: column; gap: 8px; margin-top: 12px; }
.step { padding: 8px 12px; background: var(--card-bg); border-radius: 6px; font-size: 13px; color: var(--text-secondary); }
.step strong { color: var(--accent); margin-right: 6px; }
.step code { background: #eee; color: var(--text); padding: 1px 6px; border-radius: 4px; font-size: 11px; }
</style>
