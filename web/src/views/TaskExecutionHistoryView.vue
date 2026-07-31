<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ExecutionLogTable from '../components/ExecutionLogTable.vue'
import { listExecutionLogs } from '../api/executionLogs'
import type { ExecutionLog } from '../types'

const props = defineProps<{ id: string }>()
const logs = ref<ExecutionLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const result = await listExecutionLogs({ task_id: props.id, limit: pageSize, offset: (page.value - 1) * pageSize })
    logs.value = result.logs
    total.value = result.total
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <h2>任務執行紀錄</h2>
      <el-button :loading="loading" @click="load">重新整理</el-button>
    </div>
    <ExecutionLogTable :logs="logs" :show-task="false" />
    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="prev, pager, next"
      style="margin-top: 16px"
      @current-change="load"
    />
  </el-card>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
</style>
