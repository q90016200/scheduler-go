<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import TaskForm from '../components/TaskForm.vue'
import { createTask, deleteTask, listTasks, runTask, updateTask } from '../api/tasks'
import { listTaskGroups } from '../api/taskGroups'
import { useAuthStore } from '../stores/auth'
import type { ActiveStatus, SortField, SortOrder, Task, TaskGroup, TaskWriteDTO } from '../types'

const router = useRouter()
const auth = useAuthStore()
const tasks = ref<Task[]>([])
const groups = ref<TaskGroup[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20

const dialogVisible = ref(false)
const editingTask = ref<Task>()
const saving = ref(false)
const runningIds = ref(new Set<string>())
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

async function load() {
  loading.value = true
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

async function handleDelete(task: Task) {
  await ElMessageBox.confirm(`確定刪除任務「${task.name}」？`, '刪除任務', { type: 'warning' })
  await deleteTask(task.id)
  ElMessage.success('已刪除')
  load()
}

async function handleRun(task: Task) {
  runningIds.value.add(task.id)
  try {
    await runTask(task.id)
    ElMessage.success('已觸發執行，請至執行紀錄查看結果')
  } catch (err) {
    ElMessage.error((err as Error).message || '觸發執行失敗')
  } finally {
    runningIds.value.delete(task.id)
  }
}

async function handleStatusChange(task: Task, status: ActiveStatus) {
  statusUpdatingIds.value.add(task.id)
  try {
    await updateTask(task.id, {
      group_id: task.group_id,
      name: task.name,
      description: task.description,
      schedule: task.schedule,
      type: task.type,
      endpoint: task.endpoint,
      method: task.method,
      payload: task.payload,
      status,
      user_id: auth.user?.id ?? '',
    })
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
        <el-option label="啟用" value="active" />
        <el-option label="停用" value="inactive" />
      </el-select>
      <el-select v-model="filters.sort_field" placeholder="排序欄位" clearable style="width: 140px" @change="reloadFromFirstPage">
        <el-option label="名稱" value="name" />
        <el-option label="建立時間" value="created_at" />
        <el-option label="狀態" value="status" />
        <el-option label="排程" value="schedule" />
      </el-select>
      <el-select v-model="filters.sort_order" style="width: 100px" @change="reloadFromFirstPage">
        <el-option label="正序" value="asc" />
        <el-option label="倒序" value="desc" />
      </el-select>
      <el-button type="primary" @click="openCreate">新增任務</el-button>
      <el-button :loading="loading" @click="load">重新整理</el-button>
    </div>

    <el-table :data="tasks" style="width: 100%">
      <el-table-column prop="name" label="名稱" />
      <el-table-column prop="group.name" label="群組" />
      <el-table-column prop="type" label="類型" width="80" />
      <el-table-column prop="schedule" label="排程" />
      <el-table-column label="狀態" width="140">
        <template #default="{ row }">
          <el-switch
            :model-value="row.status"
            active-value="active"
            inactive-value="inactive"
            active-text="啟用"
            inactive-text="停用"
            :loading="statusUpdatingIds.has(row.id)"
            @change="(val: ActiveStatus) => handleStatusChange(row, val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="320">
        <template #default="{ row }">
          <el-button size="small" @click="router.push(`/tasks/${row.id}`)">詳情</el-button>
          <el-button size="small" @click="openEdit(row)">編輯</el-button>
          <el-button size="small" :loading="runningIds.has(row.id)" @click="handleRun(row)">立即執行</el-button>
          <el-button size="small" @click="router.push(`/tasks/${row.id}/executions`)">執行紀錄</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">刪除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      :page-size="pageSize"
      :total="total"
      layout="prev, pager, next"
      style="margin-top: 16px"
      @current-change="load"
    />

    <el-dialog v-model="dialogVisible" :title="editingTask ? '編輯任務' : '新增任務'" width="600px">
      <TaskForm v-if="dialogVisible" ref="formRef" :initial="editingTask" @submit="handleSubmit" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="formRef?.handleSubmit()">儲存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
</style>
