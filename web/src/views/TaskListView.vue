<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import StatusTag from '../components/StatusTag.vue'
import { deleteTask, listTasks } from '../api/tasks'
import type { SortField, SortOrder, Task } from '../types'

const router = useRouter()
const tasks = ref<Task[]>([])
const filters = reactive<{ status: string; sort_field: SortField | ''; sort_order: SortOrder }>({
  status: '',
  sort_field: '',
  sort_order: 'asc',
})

async function load() {
  tasks.value = await listTasks({
    status: filters.status || undefined,
    sort_field: filters.sort_field || undefined,
    sort_order: filters.sort_order,
  })
}

onMounted(load)

async function handleDelete(task: Task) {
  await ElMessageBox.confirm(`確定刪除任務「${task.name}」？`, '刪除任務', { type: 'warning' })
  await deleteTask(task.id)
  ElMessage.success('已刪除')
  load()
}
</script>

<template>
  <el-card>
    <div class="toolbar">
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
      <el-button type="primary" @click="router.push('/tasks/new')">新增任務</el-button>
    </div>

    <el-table :data="tasks" style="width: 100%">
      <el-table-column prop="name" label="名稱" />
      <el-table-column prop="group.name" label="群組" />
      <el-table-column prop="type" label="類型" width="80" />
      <el-table-column prop="schedule" label="排程" />
      <el-table-column label="狀態" width="100">
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column label="操作" width="240">
        <template #default="{ row }">
          <el-button size="small" @click="router.push(`/tasks/${row.id}/edit`)">編輯</el-button>
          <el-button size="small" @click="router.push(`/tasks/${row.id}/executions`)">執行紀錄</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">刪除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
</style>
