<script setup lang="ts">
import { Folder, Histogram, List, Odometer } from '@element-plus/icons-vue'
import { useAuthStore } from './stores/auth'
import { autoRefreshEnabled } from './composables/useAutoRefresh'

const auth = useAuthStore()
</script>

<template>
  <el-container class="app-shell">
    <el-header class="app-header">
      <div class="glass-nav">
        <el-menu mode="horizontal" router :ellipsis="false" class="app-menu">
          <el-menu-item index="/dashboard"><el-icon><Odometer /></el-icon>儀表板</el-menu-item>
          <el-menu-item index="/tasks"><el-icon><List /></el-icon>任務</el-menu-item>
          <el-menu-item index="/groups"><el-icon><Folder /></el-icon>任務群組</el-menu-item>
          <el-menu-item index="/executions"><el-icon><Histogram /></el-icon>執行紀錄</el-menu-item>
        </el-menu>
        <div class="app-user">
          <el-switch v-model="autoRefreshEnabled" active-text="自動刷新" />
          <span class="app-avatar">{{ auth.user?.user_name?.slice(0, 1)?.toUpperCase() }}</span>
        </div>
      </div>
    </el-header>
    <el-main>
      <router-view />
    </el-main>
  </el-container>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
}
.app-header {
  height: auto;
  padding: 16px 20px 0;
  background: transparent;
}
.glass-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 18px;
  border-radius: 999px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  box-shadow: 0 8px 24px var(--glass-shadow), inset 0 1px 0 oklch(1 0 0 / 0.6);
}
.app-menu {
  --el-menu-horizontal-height: 44px;
  flex: 1;
  background: transparent;
  border: none;
}
.app-menu :deep(.el-menu-item) {
  border-radius: 999px;
  border-bottom: none !important;
  color: var(--text-muted);
  margin: 0 2px;
}
.app-menu :deep(.el-menu-item.is-active) {
  color: var(--accent) !important;
  background: var(--glass-bg-strong);
  box-shadow: inset 0 0 0 1px var(--glass-border);
}
.app-menu :deep(.el-menu-item:hover) {
  background: oklch(1 0 0 / 0.3);
}
.app-user {
  display: flex;
  align-items: center;
  gap: 14px;
}
.app-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: var(--accent);
  background: var(--glass-bg-strong);
  box-shadow: inset 0 0 0 1px var(--glass-border);
}
</style>
