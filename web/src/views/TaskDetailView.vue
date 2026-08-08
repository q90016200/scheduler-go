<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Back, Document, VideoPlay } from '@element-plus/icons-vue'
import StatusTag from '../components/StatusTag.vue'
import IconButton from '../components/IconButton.vue'
import RunTaskDialog from '../components/RunTaskDialog.vue'
import { getTask, getTaskStats } from '../api/tasks'
import type { Task, TaskStats } from '../types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const task = ref<Task>()
const stats = ref<TaskStats>()
const runDialogVisible = ref(false)

function formatTime(unixSeconds?: number | null) {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}

async function load() {
  task.value = await getTask(props.id)
  stats.value = await getTaskStats(props.id)
}

onMounted(load)
</script>

<template>
  <el-card v-if="task">
    <div class="toolbar">
      <IconButton :icon="Back" label="返回清單" @click="router.push('/tasks')" />
      <IconButton :icon="VideoPlay" label="立即執行" @click="runDialogVisible = true" />
      <IconButton :icon="Document" label="執行紀錄" @click="router.push(`/tasks/${task.id}/executions`)" />
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="名稱">{{ task.name }}</el-descriptions-item>
      <el-descriptions-item label="狀態"><StatusTag :status="task.status" /></el-descriptions-item>
      <el-descriptions-item label="群組">{{ task.group?.name ?? '-' }}</el-descriptions-item>
      <el-descriptions-item label="類型">{{ task.type }}</el-descriptions-item>
      <el-descriptions-item label="排程 (cron)">{{ task.schedule }}</el-descriptions-item>
      <el-descriptions-item label="下次執行時間">{{ formatTime(task.next_run_at) }}</el-descriptions-item>
      <el-descriptions-item label="Method">{{ task.method }}</el-descriptions-item>
      <el-descriptions-item label="Endpoint" :span="2">{{ task.endpoint }}</el-descriptions-item>
      <el-descriptions-item label="描述" :span="2">{{ task.description || '-' }}</el-descriptions-item>
      <el-descriptions-item label="Payload" :span="2">
        <pre class="payload">{{ task.payload || '-' }}</pre>
      </el-descriptions-item>
      <el-descriptions-item label="建立者">{{ task.created_by || '-' }}</el-descriptions-item>
      <el-descriptions-item label="更新者">{{ task.updated_by || '-' }}</el-descriptions-item>
      <el-descriptions-item label="建立時間">{{ formatTime(task.created_at) }}</el-descriptions-item>
      <el-descriptions-item label="更新時間">{{ formatTime(task.updated_at) }}</el-descriptions-item>
    </el-descriptions>

    <el-descriptions v-if="stats" title="執行儀表板（近 30 天）" :column="3" border style="margin-top: 16px">
      <el-descriptions-item label="成功率">{{ stats.success_rate_30d.toFixed(1) }}%</el-descriptions-item>
      <el-descriptions-item label="平均耗時">{{ Math.round(stats.avg_duration_ms_30d) }} ms</el-descriptions-item>
      <el-descriptions-item label="近期失敗次數">{{ stats.recent_fail_count }} / 10</el-descriptions-item>
    </el-descriptions>

    <RunTaskDialog v-if="runDialogVisible" v-model:visible="runDialogVisible" :task="task" />
  </el-card>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.payload {
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
