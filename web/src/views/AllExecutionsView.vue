<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Back, Refresh } from '@element-plus/icons-vue'
import ExecutionLogTable from '../components/ExecutionLogTable.vue'
import IconButton from '../components/IconButton.vue'
import { listExecutionLogs } from '../api/executionLogs'
import { listAllTaskGroups } from '../api/taskGroups'
import { getTask } from '../api/tasks'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import { usePagedList } from '../composables/usePagedList'
import type { ExecutionLog, ExecutionStatus, SortOrder, TaskGroup } from '../types'

const props = defineProps<{ id?: string }>()
const router = useRouter()
const taskName = ref<string>()
const groups = ref<TaskGroup[]>([])
const filters = reactive<{ name: string; groupId: string; status: ExecutionStatus | ''; sort_field: 'started_at' | ''; sort_order: SortOrder }>({
  name: '',
  groupId: '',
  status: '',
  sort_field: 'started_at',
  sort_order: 'desc',
})

const { items: logs, total, page, pageSize, loading, load, reloadFromFirstPage, handleSortChange: handleSort } = usePagedList<ExecutionLog, 'started_at'>(
  (p, ps) =>
    listExecutionLogs({
      task_id: props.id,
      task_name: !props.id ? filters.name || undefined : undefined,
      group_id: !props.id ? filters.groupId || undefined : undefined,
      status: filters.status || undefined,
      sort_field: filters.sort_field || undefined,
      sort_order: filters.sort_order,
      limit: ps,
      offset: (p - 1) * ps,
    }).then((r) => ({ items: r.logs, total: r.total })),
)

function handleSortChange(e: { prop: string; order: 'ascending' | 'descending' | null }) {
  handleSort(filters, e)
}

useAutoRefresh(() => load(true), 30000)

// 兩條路由（/executions、/tasks/:id/executions）共用這個元件，切換時 id 會變但實例可能被重用，
// 所以用 watch 而非只在 onMounted 抓資料
async function loadForId() {
  taskName.value = undefined
  if (props.id) {
    getTask(props.id).then((t) => (taskName.value = t.name))
  } else if (!groups.value.length) {
    groups.value = await listAllTaskGroups()
  }
  page.value = 1
  await load()
}

onMounted(loadForId)
watch(() => props.id, loadForId)
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div class="title">
        <IconButton v-if="id" :icon="Back" label="返回" @click="router.back()" />
        <h1 v-if="id">{{ taskName ?? '任務' }} 的執行紀錄</h1>
        <h1 v-else>執行紀錄</h1>
        <span v-if="id" class="mono muted" :title="id">#{{ id.slice(-6) }}</span>
        <span v-else class="meta num">共 {{ total }} 筆</span>
      </div>
      <div v-if="id" class="toolbar">
        <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
          <el-option label="成功" value="success" />
          <el-option label="失敗" value="failed" />
        </el-select>
        <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="() => load()" />
      </div>
      <IconButton v-else :icon="Refresh" label="重新整理" :loading="loading" @click="() => load()" />
    </div>
    <el-card v-liquid-glass shadow="never">
      <div v-if="!id" class="toolbar filters">
        <el-input v-model="filters.name" placeholder="任務名稱篩選" clearable style="width: 160px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
        <el-select v-model="filters.groupId" placeholder="群組篩選" clearable style="width: 160px" @change="reloadFromFirstPage">
          <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
        </el-select>
        <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
          <el-option label="成功" value="success" />
          <el-option label="失敗" value="failed" />
        </el-select>
      </div>
      <ExecutionLogTable :logs="logs" :show-task="!id" @sort-change="handleSortChange" />
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="() => load()"
      />
    </el-card>
  </div>
</template>
