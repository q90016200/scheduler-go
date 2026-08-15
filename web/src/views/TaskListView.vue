<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Document, Edit, Plus, Refresh, View, VideoPlay } from '@element-plus/icons-vue'
import TaskForm from '../components/TaskForm.vue'
import IconButton from '../components/IconButton.vue'
import RunTaskDialog from '../components/RunTaskDialog.vue'
import { createTask, deleteTask, listTasks, updateTask, updateTaskStatus } from '../api/tasks'
import { listTaskGroups } from '../api/taskGroups'
import { useAuthStore } from '../stores/auth'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import type { Status, SortField, SortOrder, Task, TaskGroup, TaskWriteDTO } from '../types'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const tasks = ref<Task[]>([])
const groups = ref<TaskGroup[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20

const dialogVisible = ref(false)
const editingTask = ref<Task>()
const saving = ref(false)
const runDialogVisible = ref(false)
const runningTask = ref<Task>()
const statusUpdatingIds = ref(new Set<string>())
const loading = ref(false)
const formRef = ref<InstanceType<typeof TaskForm>>()
const filters = reactive<{ name: string; groupId: string; status: string; sort_field: SortField | ''; sort_order: SortOrder }>({
  name: '',
  groupId: '',
  status: '',
  sort_field: '',
  sort_order: 'asc',
})

function formatTime(unixSeconds?: number | null) {
  if (!unixSeconds) return '-'
  return new Date(unixSeconds * 1000).toLocaleString()
}

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    const result = await listTasks({
      status: filters.status || undefined,
      group_id: filters.groupId || undefined,
      name: filters.name || undefined,
      sort_field: filters.sort_field || undefined,
      sort_order: filters.sort_order,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    tasks.value = result.tasks
    total.value = result.total
  } finally {
    if (!silent) loading.value = false
  }
}

useAutoRefresh(() => load(true), 120000)

function reloadFromFirstPage() {
  page.value = 1
  load()
}

function handleSortChange({ prop, order }: { prop: string; order: 'ascending' | 'descending' | null }) {
  filters.sort_field = order ? (prop as SortField) : ''
  filters.sort_order = order === 'ascending' ? 'asc' : 'desc'
  reloadFromFirstPage()
}

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

function openRunDialog(task: Task) {
  runningTask.value = task
  runDialogVisible.value = true
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

function openCreate() {
  editingTask.value = undefined
  dialogVisible.value = true
}

function openEdit(task: Task) {
  editingTask.value = task
  dialogVisible.value = true
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
  <el-card>
    <div class="toolbar">
      <el-input v-model="filters.name" placeholder="名稱篩選" clearable style="width: 160px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
      <el-select v-model="filters.groupId" placeholder="群組篩選" clearable style="width: 160px" @change="reloadFromFirstPage">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
      <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
        <el-option label="啟用" value="enabled" />
        <el-option label="停用" value="disabled" />
      </el-select>
      <IconButton :icon="Plus" label="新增任務" type="primary" @click="openCreate" />
      <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="() => load()" />
    </div>

    <el-table :data="tasks" style="width: 100%" @sort-change="handleSortChange">
      <el-table-column prop="name" label="名稱" sortable="custom" />
      <el-table-column prop="group.name" label="群組" />
      <el-table-column prop="type" label="類型" width="80" />
      <el-table-column prop="schedule" label="排程" sortable="custom" />
      <el-table-column label="下次執行" width="180">
        <template #default="{ row }">{{ formatTime(row.next_run_at) }}</template>
      </el-table-column>
      <el-table-column label="狀態" prop="status" width="140" sortable="custom">
        <template #default="{ row }">
          <el-switch
            :model-value="row.status"
            active-value="enabled"
            inactive-value="disabled"
            active-text="啟用"
            inactive-text="停用"
            :loading="statusUpdatingIds.has(row.id)"
            @change="(val: Status) => handleStatusChange(row, val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180">
        <template #default="{ row }">
          <el-button-group>
            <IconButton :icon="View" label="詳情" @click="router.push(`/tasks/${row.id}`)" />
            <IconButton :icon="Edit" label="編輯" @click="openEdit(row)" />
            <IconButton :icon="VideoPlay" label="立即執行" @click="openRunDialog(row)" />
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
      style="margin-top: 16px"
      @current-change="() => load()"
    />

    <el-dialog v-model="dialogVisible" :title="editingTask ? '編輯任務' : '新增任務'" width="600px">
      <TaskForm v-if="dialogVisible" ref="formRef" :initial="editingTask" @submit="handleSubmit" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="formRef?.handleSubmit()">儲存</el-button>
      </template>
    </el-dialog>

    <RunTaskDialog v-if="runDialogVisible" v-model:visible="runDialogVisible" :task="runningTask" />
  </el-card>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
</style>
