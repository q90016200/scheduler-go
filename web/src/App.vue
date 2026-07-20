<script setup lang="ts">
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()

async function handleLogout() {
  await auth.logout()
  location.href = '/login'
}
</script>

<template>
  <el-container v-if="auth.isLoggedIn" class="app-shell">
    <el-header class="app-header">
      <el-menu mode="horizontal" router :ellipsis="false" class="app-menu">
        <el-menu-item index="/tasks">任務</el-menu-item>
        <el-menu-item index="/groups">任務群組</el-menu-item>
        <el-menu-item index="/executions">執行紀錄</el-menu-item>
      </el-menu>
      <div class="app-user">
        <span>{{ auth.user?.user_name }}</span>
        <el-button size="small" @click="handleLogout">登出</el-button>
      </div>
    </el-header>
    <el-main>
      <router-view />
    </el-main>
  </el-container>
  <router-view v-else />
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
