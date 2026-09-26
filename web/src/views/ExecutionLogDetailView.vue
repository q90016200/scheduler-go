<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Back, Document } from '@element-plus/icons-vue'
import StatusTag from '../components/StatusTag.vue'
import IconButton from '../components/IconButton.vue'
import TaskName from '../components/TaskName.vue'
import { getExecutionLog } from '../api/executionLogs'
import { formatTime } from '../utils/time'
import type { ExecutionLog } from '../types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const log = ref<ExecutionLog>()

async function load() {
  log.value = await getExecutionLog(props.id)
}

onMounted(load)
</script>

<template>
  <div v-if="log" class="page">
    <div class="page-head">
      <div class="title">
        <IconButton :icon="Back" label="返回" @click="router.back()" />
        <h1>執行紀錄詳情</h1>
        <StatusTag :status="log.status" />
      </div>
      <el-button :icon="Document" @click="router.push(`/tasks/${log.task_id}/executions`)">此任務的其他執行紀錄</el-button>
    </div>
    <el-card v-liquid-glass shadow="never">

    <el-descriptions :column="2" border>
      <el-descriptions-item label="任務名稱"><TaskName :id="log.task_id" :name="log.task?.name" /></el-descriptions-item>
      <el-descriptions-item label="群組">{{ log.task?.group?.name || '-' }}</el-descriptions-item>
      <el-descriptions-item label="任務 ID"><span class="mono">{{ log.task_id }}</span></el-descriptions-item>
      <el-descriptions-item label="開始時間">{{ formatTime(log.started_at) }}</el-descriptions-item>
      <el-descriptions-item label="結束時間">{{ formatTime(log.finished_at) }}</el-descriptions-item>
      <el-descriptions-item label="輸出" :span="2">
        <pre>{{ log.output || '-' }}</pre>
      </el-descriptions-item>
      <el-descriptions-item label="錯誤訊息" :span="2">
        <pre>{{ log.error_message || '-' }}</pre>
      </el-descriptions-item>
    </el-descriptions>
    </el-card>
  </div>
</template>
