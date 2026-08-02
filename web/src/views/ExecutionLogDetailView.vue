<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Back, Document } from '@element-plus/icons-vue'
import StatusTag from '../components/StatusTag.vue'
import IconButton from '../components/IconButton.vue'
import { getExecutionLog } from '../api/executionLogs'
import type { ExecutionLog } from '../types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const log = ref<ExecutionLog>()

function formatTime(unixSeconds: number) {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}

async function load() {
  log.value = await getExecutionLog(props.id)
}

onMounted(load)
</script>

<template>
  <el-card v-if="log">
    <div class="toolbar">
      <IconButton :icon="Back" label="返回" @click="router.back()" />
      <IconButton :icon="Document" label="此任務的其他執行紀錄" @click="router.push(`/tasks/${log.task_id}/executions`)" />
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="狀態"><StatusTag :status="log.status" /></el-descriptions-item>
      <el-descriptions-item label="任務名稱">{{ log.task?.name || '-' }}</el-descriptions-item>
      <el-descriptions-item label="群組">{{ log.task?.group?.name || '-' }}</el-descriptions-item>
      <el-descriptions-item label="任務 ID">{{ log.task_id }}</el-descriptions-item>
      <el-descriptions-item label="開始時間">{{ formatTime(log.started_at) }}</el-descriptions-item>
      <el-descriptions-item label="結束時間">{{ formatTime(log.finished_at) }}</el-descriptions-item>
      <el-descriptions-item label="輸出" :span="2">
        <pre class="output">{{ log.output || '-' }}</pre>
      </el-descriptions-item>
      <el-descriptions-item label="錯誤訊息" :span="2">
        <pre class="output">{{ log.error_message || '-' }}</pre>
      </el-descriptions-item>
    </el-descriptions>
  </el-card>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.output {
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
