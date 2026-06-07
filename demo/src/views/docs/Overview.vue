<template>
  <div>
    <h2>📖 概述</h2>
    <p class="intro">WebToDesktop 将任意 Vue 前端项目打包为桌面 EXE 程序，提供以下核心能力：</p>

    <div class="feature-cards">
      <router-link to="/docs/credentials" class="f-card">
        <span class="f-icon">🔐</span>
        <span class="f-title">凭证加密存储</span>
        <span class="f-desc">AES-256 加密保存用户密码，密码永不暴露到前端</span>
      </router-link>
      <router-link to="/docs/proxy" class="f-card">
        <span class="f-icon">🌐</span>
        <span class="f-title">智能网络代理</span>
        <span class="f-desc">请求自动匹配前缀 → 凭证注入 → 转发远程 API</span>
      </router-link>
      <router-link to="/docs/window" class="f-card">
        <span class="f-icon">🪟</span>
        <span class="f-title">窗口控制</span>
        <span class="f-desc">无边框窗口拖拽/缩放/最大化/全屏/最小化 API</span>
      </router-link>
      <router-link to="/docs/build" class="f-card">
        <span class="f-icon">📦</span>
        <span class="f-title">构建与部署</span>
        <span class="f-desc">make 命令、交叉编译、配置文件说明</span>
      </router-link>
    </div>
  </div>
</template>

<style scoped>
.intro { color: var(--text-secondary); font-size: 14px; margin-bottom: 24px; }
.feature-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.f-card {
  display: flex; flex-direction: column; gap: 8px;
  padding: 20px; background: var(--card-bg); border: 1px solid var(--border);
  border-radius: var(--radius); text-decoration: none; color: var(--text);
  transition: border-color 0.15s, background 0.15s;
}
.f-card:hover { border-color: var(--accent); background: #e8e8e8; }
.f-icon { font-size: 28px; }
.f-title { font-size: 15px; font-weight: 600; }
.f-desc { font-size: 12px; color: var(--text-secondary); line-height: 1.5; }
</style>
