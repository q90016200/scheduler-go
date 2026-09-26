<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh, View } from '@element-plus/icons-vue'
import IconButton from '../components/IconButton.vue'
import StatusSwitch from '../components/StatusSwitch.vue'
import { deleteTaskGroup, listTaskGroups, updateTaskGroup } from '../api/taskGroups'
import { listTasks } from '../api/tasks'
import { useAuthStore } from '../stores/auth'
import { usePagedList } from '../composables/usePagedList'
import type { SortOrder, Status, TaskGroup } from '../types'

const router = useRouter()
const auth = useAuthStore()
const statusUpdatingIds = ref(new Set<string>())
const filters = reactive<{ name: string; status: string; sort_field: 'name' | 'created_at' | 'status' | ''; sort_order: SortOrder }>({
  name: '',
  status: '',
  sort_field: '',
  sort_order: 'asc',
})

const { items: groups, total, page, pageSize, loading, load, reloadFromFirstPage, handleSortChange: handleSort } = usePagedList<TaskGroup, 'name' | 'created_at' | 'status'>(
  (p, ps) =>
    listTaskGroups({
      name: filters.name || undefined,
      status: filters.status || undefined,
      sort_field: filters.sort_field || undefined,
      sort_order: filters.sort_order,
      limit: ps,
      offset: (p - 1) * ps,
    }).then((r) => ({ items: r.groups, total: r.total })),
)

function handleSortChange(e: { prop: string; order: 'ascending' | 'descending' | null }) {
  handleSort(filters, e)
}

onMounted(load)

async function handleStatusChange(group: TaskGroup, status: Status) {
  if (statusUpdatingIds.value.has(group.id)) return
  statusUpdatingIds.value.add(group.id)
  try {
    if (status === 'disabled') {
      const enabled = await listTasks({ group_id: group.id, status: 'enabled', limit: 1 })
      if (enabled.total > 0) {
        await ElMessageBox.confirm(
          `將停用群組「${group.name}」下 ${enabled.total} 個啟用中任務，確定停用？`,
          '停用群組',
          { type: 'warning' },
        )
      }
    }

    await updateTaskGroup(group.id, { name: group.name, description: group.description, status, user_id: auth.user?.id ?? '' })
    ElMessage.success('狀態已更新')
    await load()
  } catch (err) {
    if (err === 'cancel' || err === 'close') return
    ElMessage.error((err as Error).message || '狀態更新失敗')
  } finally {
    statusUpdatingIds.value.delete(group.id)
  }
}

async function handleDelete(group: TaskGroup) {
  try {
    const [all, enabled] = await Promise.all([
      listTasks({ group_id: group.id, limit: 5 }),
      listTasks({ group_id: group.id, status: 'enabled', limit: 1 }),
    ])

    let warning = `確定刪除群組「${group.name}」？`
    if (all.total > 0) {
      const names = all.tasks.map((t) => t.name).join('、') + (all.total > all.tasks.length ? '…' : '')
      warning =
        `群組「${group.name}」底下共 ${all.total} 個任務（其中 ${enabled.total} 個啟用中）：${names}。` +
        `刪除群組不會刪除這些任務，但會變成沒有群組；啟用中的任務會被停用。確定刪除？`
    }
    await ElMessageBox.confirm(warning, '刪除群組', { type: 'warning' })

    const result = await deleteTaskGroup(group.id)
    const deactivated = result.deactivated_tasks ?? []
    ElMessage.success(deactivated.length > 0 ? `已刪除，並停用了 ${deactivated.length} 個任務` : '已刪除')
    load()
  } catch (err) {
    if (err === 'cancel' || err === 'close') return
    ElMessage.error((err as Error).message || '刪除群組失敗')
  }
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div class="title">
        <h1>任務群組</h1>
        <span class="meta num">共 {{ total }} 筆</span>
      </div>
      <div class="toolbar">
        <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="() => load()" />
        <el-button type="primary" :icon="Plus" @click="router.push('/groups/new')">新增群組</el-button>
      </div>
    </div>
    <el-card v-liquid-glass shadow="never">
    <div class="toolbar filters">
      <el-input v-model="filters.name" placeholder="名稱篩選" clearable style="width: 160px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
      <el-select v-model="filters.status" placeholder="狀態篩選" clearable style="width: 140px" @change="reloadFromFirstPage">
        <el-option label="啟用" value="enabled" />
        <el-option label="停用" value="disabled" />
      </el-select>
    </div>

    <el-table :data="groups" style="width: 100%" @sort-change="handleSortChange">
      <el-table-column prop="name" label="名稱" sortable="custom" />
      <el-table-column prop="description" label="描述" />
      <el-table-column label="狀態" prop="status" width="140" sortable="custom">
        <template #default="{ row }">
          <StatusSwitch
            :model-value="row.status"
            :loading="statusUpdatingIds.has(row.id)"
            @change="(val: Status) => handleStatusChange(row, val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button-group>
            <IconButton :icon="View" label="查看任務" @click="router.push(`/tasks?group_id=${row.id}`)" />
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
      @current-change="() => load()"
    />
    </el-card>
  </div>
</template>
