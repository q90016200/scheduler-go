<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Document, Edit, Plus, Refresh, View, VideoPlay } from '@element-plus/icons-vue'
import TaskForm from '../components/TaskForm.vue'
import IconButton from '../components/IconButton.vue'
import RunTaskDialog from '../components/RunTaskDialog.vue'
import TaskName from '../components/TaskName.vue'
import TaskKind from '../components/TaskKind.vue'
import StatusSwitch from '../components/StatusSwitch.vue'
import { describeCron } from '../utils/cron'
import { formatTime } from '../utils/time'
import { createTask, deleteTask, listTasks, updateTask, updateTaskStatus } from '../api/tasks'
import { listTaskGroups } from '../api/taskGroups'
import { useAuthStore } from '../stores/auth'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import { usePagedList } from '../composables/usePagedList'
import type { Status, SortField, SortOrder, Task, TaskGroup, TaskWriteDTO } from '../types'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const groups = ref<TaskGroup[]>([])

const dialogVisible = ref(false)
const editingTask = ref<Task>()
const saving = ref(false)
const runDialogVisible = ref(false)
const runningTask = ref<Task>()
const statusUpdatingIds = ref(new Set<string>())
const formRef = ref<InstanceType<typeof TaskForm>>()
const filters = reactive<{ name: string; groupId: string; status: string; sort_field: SortField | ''; sort_order: SortOrder }>({
  name: '',
  groupId: '',
  status: '',
  sort_field: '',
  sort_order: 'asc',
})

const { items: tasks, total, page, pageSize, loading, load, reloadFromFirstPage, handleSortChange: handleSort } = usePagedList<Task, SortField>(
  (p, ps) =>
    listTasks({
      status: filters.status || undefined,
      group_id: filters.groupId || undefined,
      name: filters.name || undefined,
      sort_field: filters.sort_field || undefined,
      sort_order: filters.sort_order,
      limit: ps,
      offset: (p - 1) * ps,
    }).then((r) => ({ items: r.tasks, total: r.total })),
)

function handleSortChange(e: { prop: string; order: 'ascending' | 'descending' | null }) {
  handleSort(filters, e)
}

// 同一頁出現重複名稱時標上「同名」；跨頁的重複靠每列都帶的短 ID 分辨
const duplicateNames = computed(() => {
  const counts = new Map<string, number>()
  for (const t of tasks.value) counts.set(t.name, (counts.get(t.name) ?? 0) + 1)
  return new Set([...counts].filter(([, n]) => n > 1).map(([name]) => name))
})

function rowClassName({ row }: { row: Task }) {
  return row.consecutive_failures > 0 ? 'row-alert' : ''
}

useAutoRefresh(() => load(true), 120000)

onMounted(async () => {
  // ponytail: 下拉選單需要「全部群組」，用大 limit 取代真正的全量 API；群組數量大到 1000 筆時再改成搜尋式下拉
  groups.value = (await listTaskGroups({ limit: 1000 })).groups
  const groupId = route.query.group_id
  if (typeof groupId === 'string' && groupId) filters.groupId = groupId
  await load()
})

async function handleDelete(task: Task) {
  await ElMessageBox.confirm(`確定刪除任務「${task.name}」？`, '刪除任務', { type: 'warning' })
  await deleteTask(task.id)
  ElMessage.success('已刪除')
  load()
}

async function handleStatusChange(task: Task, status: Status) {
  statusUpdatingIds.value.add(task.id)
  try {
    await updateTaskStatus(task.id, status, auth.user?.id ?? '')
    ElMessage.success('狀態已更新')
    await load()
  } catch (err) {
    ElMessage.error((err as Error).message || '狀態更新失敗')
  } finally {
    statusUpdatingIds.value.delete(task.id)
  }
}

async function handleSubmit(payload: TaskWriteDTO) {
  saving.value = true
  try {
    if (editingTask.value) {
      await updateTask(editingTask.value.id, payload)
      ElMessage.success('任務已更新')
    } else {
      await createTask(payload)
      ElMessage.success('任務已建立')
    }
    dialogVisible.value = false
    load()
  } catch (err) {
    ElMessage.error((err as Error).message || '儲存失敗')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div class="title">
        <h1>任務</h1>
        <span class="meta num">共 {{ total }} 筆</span>
      </div>
      <div class="toolbar">
        <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="() => load()" />
        <el-button type="primary" :icon="Plus" @click="() => { editingTask = undefined; dialogVisible = true }">新增任務</el-button>
      </div>
    </div>

    <el-card v-liquid-glass shadow="never">
      <div class="toolbar filters">
        <el-input v-model="filters.name" placeholder="名稱篩選" clearable style="width: 180px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
        <el-select v-model="filters.groupId" placeholder="群組篩選" clearable style="width: 160px" @change="reloadFromFirstPage">
          <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
        </el-select>
        <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
          <el-option label="啟用" value="enabled" />
          <el-option label="停用" value="disabled" />
        </el-select>
      </div>

      <el-table :data="tasks" style="width: 100%" :row-class-name="rowClassName" @sort-change="handleSortChange">
        <el-table-column prop="name" label="名稱" sortable="custom" min-width="200">
          <template #default="{ row }">
            <TaskName :id="row.id" :name="row.name" :group="row.group?.name" :duplicate="duplicateNames.has(row.name)" />
          </template>
        </el-table-column>
        <el-table-column label="類型 / 方法" min-width="170">
          <template #default="{ row }"><TaskKind :type="row.type" :method="row.method" /></template>
        </el-table-column>
        <el-table-column prop="schedule" label="排程" sortable="custom" min-width="170">
          <template #default="{ row }">
            <div class="mono">{{ row.schedule }}</div>
            <div class="muted">{{ describeCron(row.schedule) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="下次執行" width="170">
          <template #default="{ row }"><span class="num">{{ formatTime(row.next_run_at) }}</span></template>
        </el-table-column>
        <el-table-column label="連續失敗" width="90" align="center">
          <template #default="{ row }">
            <span class="streak num" :class="{ bad: row.consecutive_failures > 0 }">{{ row.consecutive_failures }}</span>
          </template>
        </el-table-column>
        <el-table-column label="狀態" prop="status" width="130" sortable="custom">
          <template #default="{ row }">
            <StatusSwitch
              :model-value="row.status"
              :loading="statusUpdatingIds.has(row.id)"
              @change="(val: Status) => handleStatusChange(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button-group>
              <IconButton :icon="View" label="詳情" @click="router.push(`/tasks/${row.id}`)" />
              <IconButton :icon="Edit" label="編輯" @click="() => { editingTask = row; dialogVisible = true }" />
              <IconButton :icon="VideoPlay" label="立即執行" @click="() => { runningTask = row; runDialogVisible = true }" />
              <IconButton :icon="Document" label="執行紀錄" @click="router.push(`/tasks/${row.id}/executions`)" />
              <IconButton :icon="Delete" label="刪除" type="danger" @click="handleDelete(row)" />
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="() => load()"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editingTask ? `編輯任務：${editingTask.name} #${editingTask.id.slice(-6)}` : '新增任務'" width="600px" append-to-body>
      <TaskForm v-if="dialogVisible" ref="formRef" :initial="editingTask" @submit="handleSubmit" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="formRef?.handleSubmit()">儲存</el-button>
      </template>
    </el-dialog>

    <RunTaskDialog v-if="runDialogVisible" v-model:visible="runDialogVisible" :task="runningTask" />
  </div>
</template>

<style scoped>
.streak {
  font: 12px var(--mono);
  color: var(--text-muted);
}
.streak.bad {
  color: var(--danger);
  font-weight: 700;
}
</style>
