<script setup lang="ts">
import { Folder, Histogram, List, Odometer } from '@element-plus/icons-vue'
import { useAuthStore } from './stores/auth'
import { autoRefreshEnabled } from './composables/useAutoRefresh'

const auth = useAuthStore()
</script>

<template>
  <el-container class="app-shell">
    <el-header class="app-header">
      <el-menu mode="horizontal" router :ellipsis="false" class="app-menu">
        <el-menu-item index="/dashboard"><el-icon><Odometer /></el-icon>儀表板</el-menu-item>
        <el-menu-item index="/tasks"><el-icon><List /></el-icon>任務</el-menu-item>
        <el-menu-item index="/groups"><el-icon><Folder /></el-icon>任務群組</el-menu-item>
        <el-menu-item index="/executions"><el-icon><Histogram /></el-icon>執行紀錄</el-menu-item>
      </el-menu>
      <div class="app-user">
        <el-switch v-model="autoRefreshEnabled" active-text="自動刷新" />
        <span>{{ auth.user?.user_name }}</span>
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
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.app-menu {
  flex: 1;
}
.app-user {
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>
