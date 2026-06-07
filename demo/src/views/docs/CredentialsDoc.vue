<template>
  <div>
    <h2>🔐 凭证加密存储</h2>
    <p class="desc">使用 AES-256-GCM 加密保存用户登录凭据，密码永不暴露到前端。登录时勾选"记住密码"，代理层自动将请求体中的 <code>__DESKTOP_PWD__</code> 占位符替换为真实密码。</p>

    <h3>前端示例代码（Plain JS + Bridge Proxy）</h3>
    <p class="hint">使用 <code>useBridge()</code> 获取 bridge 代理对象，直接调用 Go handler。Element Plus 组件构建 UI。</p>

    <CodeBlock>&lt;template&gt;
  &lt;el-form :model="form" label-width="80px"&gt;
    &lt;el-form-item label="用户名"&gt;
      &lt;el-input v-model="form.username" placeholder="请输入用户名" /&gt;
    &lt;/el-form-item&gt;
    &lt;el-form-item label="密码"&gt;
      &lt;el-input v-model="form.password" type="password"
        placeholder="请输入密码" show-password /&gt;
    &lt;/el-form-item&gt;
    &lt;el-form-item&gt;
      &lt;el-checkbox v-model="remember"&gt;记住密码&lt;/el-checkbox&gt;
    &lt;/el-form-item&gt;
    &lt;el-form-item&gt;
      &lt;el-button type="primary" @click="handleLogin"&gt;
        登录
      &lt;/el-button&gt;
      &lt;el-button @click="loadSavedCreds"&gt;
        加载已保存凭据
      &lt;/el-button&gt;
    &lt;/el-form-item&gt;
  &lt;/el-form&gt;

  &lt;el-divider /&gt;

  &lt;h4&gt;已保存的凭据&lt;/h4&gt;
  &lt;el-table :data="credList" v-if="credList.length"&gt;
    &lt;el-table-column prop="username" label="用户名" /&gt;
    &lt;el-table-column label="操作" width="100"&gt;
      &lt;template #default="{ row }"&gt;
        &lt;el-button type="danger" size="small"
          @click="deleteCred(row.username)"&gt;
          删除
        &lt;/el-button&gt;
      &lt;/template&gt;
    &lt;/el-table-column&gt;
  &lt;/el-table&gt;
  &lt;el-empty v-else description="暂无已保存的凭据" /&gt;
&lt;/template&gt;

&lt;script setup&gt;
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useBridge } from './composables/useBridge.js'

// ★ 获取 bridge 代理 — 所有 Go handler 自动可用
const { bridge } = useBridge()

const form = reactive({ username: '', password: '' })
const remember = ref(false)
const credList = ref([])

async function handleLogin() {
  if (remember.value) {
    // ★ 直接调用 bridge.saveCredentials()，AES 加密存入本地
    await bridge.saveCredentials({
      username: form.username,
      password: form.password,
    })
    ElMessage.success('凭据已加密保存')
  }
}

async function loadSavedCreds() {
  const result = await bridge.getCredentials()
  credList.value = result.credentials || []
}

async function deleteCred(username) {
  await bridge.deleteCredentials({ username })
  await loadSavedCreds()
}
&lt;/script&gt;</CodeBlock>

    <h3>安全架构</h3>
    <div class="flow">
      <div class="step"><strong>1.</strong> 用户登录 → 勾选"记住密码"</div>
      <div class="step"><strong>2.</strong> Bridge <code>saveCredentials</code> → AES-256 加密 → 写入 <code>credentials.dat</code></div>
      <div class="step"><strong>3.</strong> 后续请求自动携带 <code>X-Credential-Username</code> 请求头</div>
      <div class="step"><strong>4.</strong> 代理层将请求体 <code>__DESKTOP_PWD__</code> 替换为真实密码</div>
      <div class="step"><strong>5.</strong> 转发到远程 API → 密码永不出现在前端</div>
    </div>
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
