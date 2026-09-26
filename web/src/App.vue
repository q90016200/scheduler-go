<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Folder, Histogram, List, Odometer } from '@element-plus/icons-vue'
import { useAuthStore } from './stores/auth'
import { autoRefreshEnabled } from './composables/useAutoRefresh'

const route = useRoute()
const auth = useAuthStore()

// 使用者管理（/users）先不放進導覽，路由保留
const nav = [
  { to: '/dashboard', label: '儀表板', icon: Odometer },
  { to: '/tasks', label: '任務', icon: List },
  { to: '/groups', label: '任務群組', icon: Folder },
  { to: '/executions', label: '執行紀錄', icon: Histogram },
]
// 詳情頁（/tasks/:id、/executions/:id）也要讓所屬選單亮起，所以比對第一段路徑
const section = computed(() => '/' + route.path.split('/')[1])
</script>

<template>
  <router-view v-if="route.name === 'login'" />
  <div v-else class="shell">
    <aside v-liquid-glass class="sidebar">
      <div class="brand"><span class="brand-mark" aria-hidden="true"></span>schedule-go</div>
      <nav class="nav" aria-label="主選單">
        <router-link v-for="item in nav" :key="item.to" :to="item.to" class="nav-item" :class="{ active: section === item.to }">
          <el-icon><component :is="item.icon" /></el-icon>{{ item.label }}
        </router-link>
      </nav>
      <div class="side-foot">
        <el-switch v-model="autoRefreshEnabled" size="small" active-text="自動刷新" />
        <span v-if="auth.user" class="avatar" :title="auth.user.user_name">{{ auth.user.user_name.slice(0, 1).toUpperCase() }}</span>
      </div>
    </aside>
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: 208px minmax(0, 1fr);
  gap: 16px;
  padding: 12px 16px 12px 12px;
  min-height: 100svh;
  box-sizing: border-box;
}
.sidebar {
  position: sticky;
  top: 12px;
  height: calc(100svh - 24px);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 18px 10px 14px;
  border-radius: 22px;
  box-shadow: 0 10px 30px var(--glass-shadow), inset 0 1px 0 oklch(1 0 0 / 0.5);
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  font: 600 14px var(--mono);
}
.brand-mark {
  width: 14px;
  height: 14px;
  border-radius: 4px;
  background: var(--accent);
}
.nav {
  display: grid;
  gap: 2px;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 10px;
  color: var(--text-muted);
  text-decoration: none;
  white-space: nowrap;
}
.nav-item:hover {
  background: var(--el-fill-color-light);
  color: var(--text);
}
.nav-item:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}
.nav-item.active {
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 500;
}
.side-foot {
  margin-top: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 10px;
}
.avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 12px;
  font-weight: 700;
  color: var(--accent);
  background: var(--glass-bg-strong);
  box-shadow: inset 0 0 0 1px var(--glass-border);
}
.content {
  min-width: 0;
  padding-block: 6px 24px;
}

/* 窄螢幕：側邊欄變成上方橫向導覽 */
@media (max-width: 860px) {
  .shell {
    grid-template-columns: minmax(0, 1fr);
    padding: 10px 16px;
  }
  .sidebar {
    position: static;
    height: auto;
    flex-direction: row;
    align-items: center;
    gap: 12px;
    padding: 8px 10px;
    border-radius: 16px;
    overflow-x: auto;
  }
  .brand {
    display: none;
  }
  .nav {
    display: flex;
  }
  .side-foot {
    margin-top: 0;
    margin-left: auto;
  }
}
</style>
