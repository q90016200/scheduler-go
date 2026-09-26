<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import TaskName from '../components/TaskName.vue'
import { getDashboardStats } from '../api/dashboard'
import { autoRefreshEnabled, useAutoRefresh } from '../composables/useAutoRefresh'
import type { DashboardStats } from '../types'

const stats = ref<DashboardStats>()
const updatedAt = ref<Date>()

async function load() {
  stats.value = await getDashboardStats()
  updatedAt.value = new Date()
}

onMounted(load)
useAutoRefresh(load, 30000)

const maxFail = computed(() => Math.max(1, ...(stats.value?.top_failing_tasks_24h ?? []).map((t) => t.fail_count)))
</script>

<template>
  <div class="page">
    <div class="page-head">
      <h1>儀表板</h1>
      <span v-if="updatedAt" class="meta">
        更新於 {{ updatedAt.toLocaleTimeString() }}{{ autoRefreshEnabled ? ' · 每 30 秒自動刷新' : '' }}
      </span>
    </div>

    <template v-if="stats">
      <div class="kpis">
        <div v-liquid-glass class="kpi">
          <span class="label">啟用任務</span>
          <span class="value">{{ stats.total_tasks_enabled }}</span>
          <span class="muted">另有 {{ stats.total_tasks_disabled }} 個停用</span>
        </div>
        <div v-liquid-glass class="kpi">
          <span class="label">近 24 小時執行</span>
          <span class="value">{{ stats.executions_24h.toLocaleString() }}</span>
          <span class="muted">只計排程觸發的最終結果</span>
        </div>
        <div v-liquid-glass class="kpi">
          <span class="label">近 24 小時成功率</span>
          <span class="value">{{ stats.success_rate_24h.toFixed(1) }}%</span>
          <span class="muted">&nbsp;</span>
        </div>
        <div v-liquid-glass class="kpi" :class="{ alert: stats.currently_failing.length }">
          <span class="label">連續失敗中</span>
          <span class="value">{{ stats.currently_failing.length }}</span>
          <span class="muted">{{ stats.currently_failing.length ? '見下方清單' : '全部正常' }}</span>
        </div>
      </div>

      <div class="panels">
        <el-card v-liquid-glass shadow="never">
          <h2>近 24 小時失敗最多</h2>
          <ul v-if="stats.top_failing_tasks_24h.length" class="rank">
            <li v-for="t in stats.top_failing_tasks_24h" :key="t.task_id">
              <div>
                <TaskName :id="t.task_id" :name="t.task_name" />
                <div class="bar"><b :style="{ width: (t.fail_count / maxFail) * 100 + '%' }"></b></div>
              </div>
              <span class="fail num">{{ t.fail_count }} 次</span>
            </li>
          </ul>
          <el-empty v-else :image-size="64" description="近 24 小時沒有失敗紀錄" />
        </el-card>

        <el-card v-liquid-glass shadow="never">
          <h2>目前連續失敗中的任務</h2>
          <ul v-if="stats.currently_failing.length" class="rank">
            <li v-for="t in stats.currently_failing" :key="t.task_id">
              <TaskName :id="t.task_id" :name="t.task_name" />
              <router-link :to="`/tasks/${t.task_id}/executions`" class="link">執行紀錄</router-link>
            </li>
          </ul>
          <el-empty v-else :image-size="64" description="目前沒有連續失敗中的任務" />
        </el-card>
      </div>
    </template>
  </div>
</template>

<style scoped>
.panels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 12px;
}
.rank {
  list-style: none;
  margin: 0;
  padding: 0;
}
.rank li {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px solid var(--line);
}
.rank li:first-child {
  border-top: 0;
}
.bar {
  height: 4px;
  margin-top: 6px;
  border-radius: 2px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.bar b {
  display: block;
  height: 100%;
  background: var(--danger);
}
.fail {
  font: 500 12px var(--mono);
  color: var(--danger);
}
.link {
  font-size: 12px;
  color: var(--accent);
  text-decoration: none;
}
.link:hover {
  text-decoration: underline;
}
</style>
