<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import ExecutionLogTable from '../components/ExecutionLogTable.vue'
import IconButton from '../components/IconButton.vue'
import { listExecutionLogs } from '../api/executionLogs'
import { listTaskGroups } from '../api/taskGroups'
import type { ExecutionLog, TaskGroup } from '../types'

const logs = ref<ExecutionLog[]>([])
const groups = ref<TaskGroup[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const filters = reactive({ name: '', groupId: '' })
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const result = await listExecutionLogs({
      task_name: filters.name || undefined,
      group_id: filters.groupId || undefined,
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

onMounted(async () => {
  // ponytail: 下拉選單需要「全部群組」，用大 limit 取代真正的全量 API；群組數量大到 1000 筆時再改成搜尋式下拉
  groups.value = (await listTaskGroups({ limit: 1000 })).groups
  await load()
})
</script>

<template>
  <el-card>
    <h2>所有任務最近執行紀錄</h2>
    <div class="toolbar">
      <el-input v-model="filters.name" placeholder="任務名稱篩選" clearable style="width: 160px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
      <el-select v-model="filters.groupId" placeholder="群組篩選" clearable style="width: 160px" @change="reloadFromFirstPage">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
      <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="load" />
    </div>
    <ExecutionLogTable :logs="logs" />
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
  gap: 12px;
  margin-bottom: 16px;
}
</style>
