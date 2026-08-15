<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getDashboardStats } from '../api/dashboard'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import type { DashboardStats } from '../types'

const router = useRouter()
const stats = ref<DashboardStats>()

async function load() {
  stats.value = await getDashboardStats()
}

onMounted(load)
useAutoRefresh(load, 30000)

const sections = computed(() => [
  { title: '近 24 小時失敗 Top 5', rows: stats.value?.top_failing_tasks_24h ?? [], showFailCount: true, empty: '近 24 小時沒有失敗紀錄' },
  { title: '目前連續失敗中的任務', rows: stats.value?.currently_failing ?? [], showFailCount: false, empty: '目前沒有連續失敗中的任務' },
])
</script>

<template>
  <el-card v-if="stats">
    <h2>全站儀表板</h2>

    <el-descriptions :column="2" border style="margin-top: 16px">
      <el-descriptions-item label="啟用任務數">{{ stats.total_tasks_enabled }}</el-descriptions-item>
      <el-descriptions-item label="停用任務數">{{ stats.total_tasks_disabled }}</el-descriptions-item>
      <el-descriptions-item label="近 24 小時執行次數">{{ stats.executions_24h }}</el-descriptions-item>
      <el-descriptions-item label="近 24 小時成功率">{{ stats.success_rate_24h.toFixed(1) }}%</el-descriptions-item>
    </el-descriptions>

    <template v-for="section in sections" :key="section.title">
      <h3 style="margin-top: 24px">{{ section.title }}</h3>
      <el-table :data="section.rows" style="width: 100%">
        <el-table-column prop="task_name" label="任務名稱" />
        <el-table-column v-if="section.showFailCount" prop="fail_count" label="失敗次數" width="120" />
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button text type="primary" @click="router.push(`/tasks/${row.task_id}`)">詳情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!section.rows.length" :description="section.empty" />
    </template>
  </el-card>
</template>
