<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh } from '@element-plus/icons-vue'
import StatusTag from '../components/StatusTag.vue'
import IconButton from '../components/IconButton.vue'
import { deleteTaskGroup, listTaskGroups } from '../api/taskGroups'
import { listTasks } from '../api/tasks'
import type { SortOrder, TaskGroup } from '../types'

const router = useRouter()
const groups = ref<TaskGroup[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const filters = reactive<{ name: string; status: string; sort_field: 'name' | 'created_at' | 'status' | ''; sort_order: SortOrder }>({
  name: '',
  status: '',
  sort_field: '',
  sort_order: 'asc',
})

async function load() {
  loading.value = true
  try {
    const result = await listTaskGroups({
      name: filters.name || undefined,
      status: filters.status || undefined,
      sort_field: filters.sort_field || undefined,
      sort_order: filters.sort_order,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    groups.value = result.groups
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function reloadFromFirstPage() {
  page.value = 1
  load()
}

onMounted(load)

async function handleDelete(group: TaskGroup) {
  const { tasks: activeTasks } = await listTasks({ group_id: group.id, status: 'active', limit: 100 })

  const warning =
    activeTasks.length > 0
      ? `群組「${group.name}」底下有 ${activeTasks.length} 個執行中的任務（${activeTasks.map((t) => t.name).join('、')}），刪除群組會一併停用這些任務。確定刪除？`
      : `確定刪除群組「${group.name}」？`
  await ElMessageBox.confirm(warning, '刪除群組', { type: 'warning' })

  const result = await deleteTaskGroup(group.id)
  const deactivated = result.deactivated_tasks ?? []
  ElMessage.success(deactivated.length > 0 ? `已刪除，並停用了 ${deactivated.length} 個任務` : '已刪除')
  load()
}
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="filters.name" placeholder="名稱篩選" clearable style="width: 160px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
      <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
        <el-option label="啟用" value="active" />
        <el-option label="停用" value="inactive" />
      </el-select>
      <el-select v-model="filters.sort_field" placeholder="排序欄位" clearable style="width: 140px" @change="reloadFromFirstPage">
        <el-option label="名稱" value="name" />
        <el-option label="建立時間" value="created_at" />
        <el-option label="狀態" value="status" />
      </el-select>
      <el-select v-model="filters.sort_order" style="width: 100px" @change="reloadFromFirstPage">
        <el-option label="正序" value="asc" />
        <el-option label="倒序" value="desc" />
      </el-select>
      <IconButton :icon="Plus" label="新增群組" type="primary" @click="router.push('/groups/new')" />
      <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="load" />
    </div>

    <el-table :data="groups" style="width: 100%">
      <el-table-column prop="name" label="名稱" />
      <el-table-column prop="description" label="描述" />
      <el-table-column label="狀態" width="100">
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button-group>
            <IconButton :icon="Edit" label="編輯" @click="router.push(`/groups/${row.id}/edit`)" />
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
      @current-change="load"
    />
  </el-card>
</template>

<style scoped>
.toolbar {
  margin-bottom: 16px;
}
</style>
