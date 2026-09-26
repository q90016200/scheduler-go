<script setup lang="ts">
import { useRouter } from 'vue-router'
import { View } from '@element-plus/icons-vue'
import StatusTag from './StatusTag.vue'
import IconButton from './IconButton.vue'
import TaskName from './TaskName.vue'
import { formatTime } from '../utils/time'
import type { ExecutionLog } from '../types'

withDefaults(defineProps<{ logs: ExecutionLog[]; showTask?: boolean }>(), { showTask: true })
const router = useRouter()
</script>

<template>
  <el-table
    :data="logs"
    style="width: 100%"
    :default-sort="{ prop: 'started_at', order: 'descending' }"
  >
    <el-table-column type="expand">
      <template #default="{ row }">
        <div style="padding: 0 16px 16px 60px">
          <p><strong>輸出</strong></p>
          <pre style="background: var(--el-fill-color-light); padding: 8px; border-radius: 4px">{{ row.output || '(無)' }}</pre>
          <template v-if="row.error_message">
            <p><strong>錯誤訊息</strong></p>
            <pre style="color: var(--el-color-danger)">{{ row.error_message }}</pre>
          </template>
        </div>
      </template>
    </el-table-column>
    <el-table-column v-if="showTask" label="任務名稱" min-width="160">
      <template #default="{ row }"><TaskName :id="row.task_id" :name="row.task?.name" /></template>
    </el-table-column>
    <el-table-column v-if="showTask" prop="task.group.name" label="群組" />
    <el-table-column label="狀態" width="100">
      <template #default="{ row }"><StatusTag :status="row.status" /></template>
    </el-table-column>
    <el-table-column label="來源" width="90">
      <template #default="{ row }">{{ row.source === 'manual' ? '手動' : '排程' }}</template>
    </el-table-column>
    <el-table-column label="嘗試次數" width="90">
      <template #default="{ row }">第 {{ row.attempt_number }} 次{{ row.is_final_attempt ? '' : '（重試中）' }}</template>
    </el-table-column>
    <el-table-column prop="started_at" label="開始時間" width="180" sortable="custom">
      <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
    </el-table-column>
    <el-table-column label="結束時間" width="180">
      <template #default="{ row }">{{ formatTime(row.finished_at) }}</template>
    </el-table-column>
    <el-table-column prop="output" label="輸出" show-overflow-tooltip />
    <el-table-column prop="error_message" label="錯誤訊息" show-overflow-tooltip />
    <el-table-column label="操作" width="60">
      <template #default="{ row }">
        <IconButton :icon="View" label="詳情" @click="router.push(`/executions/${row.id}`)" />
      </template>
    </el-table-column>
  </el-table>
</template>
