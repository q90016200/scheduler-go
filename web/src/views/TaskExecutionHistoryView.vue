<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import ExecutionLogTable from '../components/ExecutionLogTable.vue'
import IconButton from '../components/IconButton.vue'
import { listExecutionLogs } from '../api/executionLogs'
import type { ExecutionLog, ExecutionStatus, SortOrder } from '../types'

const props = defineProps<{ id: string }>()
const logs = ref<ExecutionLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const filters = reactive<{ status: ExecutionStatus | ''; sort_field: 'started_at' | ''; sort_order: SortOrder }>({
  status: '',
  sort_field: 'started_at',
  sort_order: 'desc',
})

async function load() {
  loading.value = true
  try {
    const result = await listExecutionLogs({
      task_id: props.id,
      status: filters.status || undefined,
      sort_field: filters.sort_field || undefined,
      sort_order: filters.sort_order,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    logs.value = result.logs
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function reloadFromFirstPage() {
  page.value = 1
  load()
}

function handleSortChange({ prop, order }: { prop: string; order: 'ascending' | 'descending' | null }) {
  filters.sort_field = order ? (prop as 'started_at') : ''
  filters.sort_order = order === 'ascending' ? 'asc' : 'desc'
  reloadFromFirstPage()
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <h2>任務執行紀錄</h2>
      <div class="toolbar-actions">
        <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
          <el-option label="成功" value="success" />
          <el-option label="失敗" value="failed" />
        </el-select>
        <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="load" />
      </div>
    </div>
    <ExecutionLogTable :logs="logs" :show-task="false" @sort-change="handleSortChange" />
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
.toolbar-actions {
  display: flex;
  gap: 12px;
}
</style>
