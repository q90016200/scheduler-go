<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Back, Document, VideoPlay } from '@element-plus/icons-vue'
import StatusTag from '../components/StatusTag.vue'
import IconButton from '../components/IconButton.vue'
import RunTaskDialog from '../components/RunTaskDialog.vue'
import TaskKind from '../components/TaskKind.vue'
import { describeCron } from '../utils/cron'
import { formatTime } from '../utils/time'
import { getTask, getTaskStats } from '../api/tasks'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import type { Task, TaskStats } from '../types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const task = ref<Task>()
const stats = ref<TaskStats>()
const runDialogVisible = ref(false)

async function load() {
  const [t, s] = await Promise.all([getTask(props.id), getTaskStats(props.id)])
  task.value = t
  stats.value = s
}

onMounted(load)
useAutoRefresh(load, 30000)
</script>

<template>
  <div v-if="task" class="page">
    <div class="page-head">
      <div class="title">
        <IconButton :icon="Back" label="返回" @click="router.back()" />
        <h1>{{ task.name }}</h1>
        <span class="mono muted" :title="task.id">#{{ task.id.slice(-6) }}</span>
        <StatusTag :status="task.status" />
      </div>
      <div class="toolbar">
        <el-button :icon="Document" @click="router.push(`/tasks/${task.id}/executions`)">執行紀錄</el-button>
        <el-button type="primary" :icon="VideoPlay" @click="runDialogVisible = true">立即執行</el-button>
      </div>
    </div>

    <div v-if="stats" class="kpis">
      <div v-liquid-glass class="kpi">
        <span class="label">近 30 天成功率</span>
        <span class="value">{{ stats.success_rate_30d.toFixed(1) }}%</span>
      </div>
      <div v-liquid-glass class="kpi">
        <span class="label">近 30 天平均耗時</span>
        <span class="value">{{ Math.round(stats.avg_duration_ms_30d) }} ms</span>
      </div>
      <div v-liquid-glass class="kpi" :class="{ alert: stats.recent_fail_count > 0 }">
        <span class="label">最近 10 次排程失敗</span>
        <span class="value">{{ stats.recent_fail_count }} / 10</span>
      </div>
      <div v-liquid-glass class="kpi" :class="{ alert: task.consecutive_failures > 0 }">
        <span class="label">目前連續失敗</span>
        <span class="value">{{ task.consecutive_failures }}</span>
      </div>
    </div>

    <el-card v-liquid-glass shadow="never">
    <el-descriptions :column="2" border>
      <el-descriptions-item label="任務 ID"><span class="mono">{{ task.id }}</span></el-descriptions-item>
      <el-descriptions-item label="群組">{{ task.group?.name ?? '-' }}</el-descriptions-item>
      <el-descriptions-item label="類型 / 方法"><TaskKind :type="task.type" :method="task.method" /></el-descriptions-item>
      <el-descriptions-item label="下次執行時間">{{ formatTime(task.next_run_at) }}</el-descriptions-item>
      <el-descriptions-item label="排程 (cron)" :span="2">
        <span class="mono">{{ task.schedule }}</span>
        <span class="muted">　{{ describeCron(task.schedule) }}</span>
      </el-descriptions-item>
      <el-descriptions-item label="Endpoint" :span="2"><span class="mono">{{ task.endpoint }}</span></el-descriptions-item>
      <el-descriptions-item label="描述" :span="2">{{ task.description || '-' }}</el-descriptions-item>
      <el-descriptions-item label="Payload" :span="2">
        <pre>{{ task.payload || '-' }}</pre>
      </el-descriptions-item>
      <el-descriptions-item label="建立者">{{ task.created_by || '-' }}</el-descriptions-item>
      <el-descriptions-item label="更新者">{{ task.updated_by || '-' }}</el-descriptions-item>
      <el-descriptions-item label="建立時間">{{ formatTime(task.created_at) }}</el-descriptions-item>
      <el-descriptions-item label="更新時間">{{ formatTime(task.updated_at) }}</el-descriptions-item>
    </el-descriptions>
    </el-card>

    <RunTaskDialog v-if="runDialogVisible" v-model:visible="runDialogVisible" :task="task" />
  </div>
</template>
