<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import StatusTag from '../components/StatusTag.vue'
import TaskForm from '../components/TaskForm.vue'
import { createTask, deleteTask, listTasks, updateTask } from '../api/tasks'
import { listTaskGroups } from '../api/taskGroups'
import type { SortField, SortOrder, Task, TaskGroup, TaskWriteDTO } from '../types'

const router = useRouter()
const tasks = ref<Task[]>([])
const groups = ref<TaskGroup[]>([])

const dialogVisible = ref(false)
const editingTask = ref<Task>()
const saving = ref(false)
const formRef = ref<InstanceType<typeof TaskForm>>()
const filters = reactive<{ name: string; groupId: string; status: string; sort_field: SortField | ''; sort_order: SortOrder }>({
  name: '',
  groupId: '',
  status: '',
  sort_field: '',
  sort_order: 'asc',
})

const filteredTasks = computed(() =>
  tasks.value.filter((task) => {
    if (filters.name && !task.name.toLowerCase().includes(filters.name.toLowerCase())) return false
    if (filters.groupId && task.group_id !== filters.groupId) return false
    return true
  }),
)

async function load() {
  tasks.value = await listTasks({
    status: filters.status || undefined,
    sort_field: filters.sort_field || undefined,
    sort_order: filters.sort_order,
  })
}

onMounted(async () => {
  groups.value = await listTaskGroups()
  await load()
})

async function handleDelete(task: Task) {
  await ElMessageBox.confirm(`確定刪除任務「${task.name}」？`, '刪除任務', { type: 'warning' })
  await deleteTask(task.id)
  ElMessage.success('已刪除')
  load()
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
      <el-input v-model="filters.name" placeholder="名稱篩選" clearable style="width: 160px" />
      <el-select v-model="filters.groupId" placeholder="群組篩選" clearable style="width: 160px">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
      <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="load">
        <el-option label="啟用" value="active" />
        <el-option label="停用" value="inactive" />
      </el-select>
      <el-select v-model="filters.sort_field" placeholder="排序欄位" clearable style="width: 140px" @change="load">
        <el-option label="名稱" value="name" />
        <el-option label="建立時間" value="created_at" />
        <el-option label="狀態" value="status" />
        <el-option label="排程" value="schedule" />
      </el-select>
      <el-select v-model="filters.sort_order" style="width: 100px" @change="load">
        <el-option label="正序" value="asc" />
        <el-option label="倒序" value="desc" />
      </el-select>
      <el-button type="primary" @click="openCreate">新增任務</el-button>
    </div>

    <el-table :data="filteredTasks" style="width: 100%">
      <el-table-column prop="name" label="名稱" />
      <el-table-column prop="group.name" label="群組" />
      <el-table-column prop="type" label="類型" width="80" />
      <el-table-column prop="schedule" label="排程" />
      <el-table-column label="狀態" width="100">
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column label="操作" width="240">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">編輯</el-button>
          <el-button size="small" @click="router.push(`/tasks/${row.id}/executions`)">執行紀錄</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">刪除</el-button>
        </template>
      </el-table-column>
    </el-table>

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
