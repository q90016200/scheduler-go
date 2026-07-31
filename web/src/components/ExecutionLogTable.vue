<script setup lang="ts">
import { useRouter } from 'vue-router'
import StatusTag from './StatusTag.vue'
import type { ExecutionLog } from '../types'

withDefaults(defineProps<{ logs: ExecutionLog[]; showTask?: boolean }>(), { showTask: true })
const router = useRouter()

function formatTime(unixSeconds: number) {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}
</script>

<template>
  <el-table :data="logs" style="width: 100%">
    <el-table-column v-if="showTask" prop="task.name" label="任務名稱" />
    <el-table-column v-if="showTask" prop="task.group.name" label="群組" />
    <el-table-column label="狀態" width="100">
      <template #default="{ row }"><StatusTag :status="row.status" /></template>
    </el-table-column>
    <el-table-column label="開始時間" width="180">
      <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
    </el-table-column>
    <el-table-column label="結束時間" width="180">
      <template #default="{ row }">{{ formatTime(row.finished_at) }}</template>
    </el-table-column>
    <el-table-column prop="output" label="輸出" show-overflow-tooltip />
    <el-table-column prop="error_message" label="錯誤訊息" show-overflow-tooltip />
    <el-table-column label="操作" width="100">
      <template #default="{ row }">
        <el-button size="small" @click="router.push(`/executions/${row.id}`)">詳情</el-button>
      </template>
    </el-table-column>
  </el-table>
</template>
