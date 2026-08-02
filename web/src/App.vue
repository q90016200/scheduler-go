<script setup lang="ts">
import { Folder, Histogram, List, SwitchButton } from '@element-plus/icons-vue'
import IconButton from './components/IconButton.vue'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()

async function handleLogout() {
  await auth.logout()
  location.href = '/login'
}
</script>

<template>
  <el-container class="app-shell">
    <el-header class="app-header">
      <el-menu mode="horizontal" router :ellipsis="false" class="app-menu">
        <el-menu-item index="/tasks"><el-icon><List /></el-icon>任務</el-menu-item>
        <el-menu-item index="/groups"><el-icon><Folder /></el-icon>任務群組</el-menu-item>
        <el-menu-item index="/executions"><el-icon><Histogram /></el-icon>執行紀錄</el-menu-item>
      </el-menu>
      <div class="app-user">
        <span>{{ auth.user?.user_name }}</span>
        <IconButton :icon="SwitchButton" label="登出" @click="handleLogout" />
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
