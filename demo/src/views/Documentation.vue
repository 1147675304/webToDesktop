<template>
  <div class="docs-layout">
    <!-- 侧边导航 -->
    <nav class="docs-sidebar">
      <h3><el-icon><Reading /></el-icon> 文档</h3>
      <router-link to="/docs" exact-active-class="active" class="nav-item">
        <span>概述</span>
      </router-link>
      <router-link to="/docs/credentials" active-class="active" class="nav-item">
        <span><el-icon><Lock /></el-icon> 凭证加密存储</span>
      </router-link>
      <router-link to="/docs/proxy" active-class="active" class="nav-item">
        <span><el-icon><Connection /></el-icon> 智能网络代理</span>
      </router-link>
      <router-link to="/docs/window" active-class="active" class="nav-item">
        <span><el-icon><Monitor /></el-icon> 窗口控制</span>
      </router-link>
      <router-link to="/docs/storage" active-class="active" class="nav-item">
        <span><el-icon><FolderOpened /></el-icon> 键值对存储</span>
      </router-link>
      <router-link to="/docs/build" active-class="active" class="nav-item">
        <span><el-icon><Box /></el-icon> 构建与部署</span>
      </router-link>
      <router-link to="/docs/bridge" active-class="active" class="nav-item">
        <span><el-icon><Switch /></el-icon> Bridge API</span>
      </router-link>
    </nav>

    <!-- 内容区 -->
    <main class="docs-content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { Reading, Lock, Connection, Monitor, Box, Switch, FolderOpened } from '@element-plus/icons-vue'
</script>

<style scoped>
.docs-layout {
  display: flex; gap: 24px; max-width: 1100px; margin: 0 auto;
}
.docs-sidebar {
  width: 200px; flex-shrink: 0;
  position: sticky; top: 0; align-self: flex-start;
  padding-top: 8px;
}
.docs-sidebar h3 {
  font-size: 15px; margin-bottom: 12px; padding: 0 12px;
}
.nav-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 12px; border-radius: 6px; font-size: 13px;
  color: var(--text-secondary); text-decoration: none;
  transition: background 0.12s, color 0.12s;
  margin-bottom: 2px;
}
.nav-item:hover { background: #e0e0e0; color: var(--text); }
.nav-item.active {
  background: var(--accent-dim); color: var(--accent);
}
.docs-content {
  flex: 1; min-width: 0; padding-top: 8px;
}
.docs-content h2 { font-size: 22px; margin-bottom: 12px; }
.docs-content h3 { font-size: 16px; margin: 20px 0 10px; }
</style>
