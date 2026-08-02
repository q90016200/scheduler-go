<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Back, Document, VideoPlay } from '@element-plus/icons-vue'
import StatusTag from '../components/StatusTag.vue'
import IconButton from '../components/IconButton.vue'
import { getTask, runTask } from '../api/tasks'
import type { Task } from '../types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const task = ref<Task>()
const running = ref(false)

function formatTime(unixSeconds: number) {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}

async function load() {
  task.value = await getTask(props.id)
}

onMounted(load)

async function handleRun() {
  running.value = true
  try {
    await runTask(props.id)
    ElMessage.success('已觸發執行，請至執行紀錄查看結果')
  } catch (err) {
    ElMessage.error((err as Error).message || '觸發執行失敗')
  } finally {
    running.value = false
  }
}
</script>

<template>
  <el-card v-if="task">
    <div class="toolbar">
      <IconButton :icon="Back" label="返回清單" @click="router.push('/tasks')" />
      <IconButton :icon="VideoPlay" label="立即執行" :loading="running" @click="handleRun" />
      <IconButton :icon="Document" label="執行紀錄" @click="router.push(`/tasks/${task.id}/executions`)" />
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="名稱">{{ task.name }}</el-descriptions-item>
      <el-descriptions-item label="狀態"><StatusTag :status="task.status" /></el-descriptions-item>
      <el-descriptions-item label="群組">{{ task.group?.name ?? '-' }}</el-descriptions-item>
      <el-descriptions-item label="類型">{{ task.type }}</el-descriptions-item>
      <el-descriptions-item label="排程 (cron)">{{ task.schedule }}</el-descriptions-item>
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
