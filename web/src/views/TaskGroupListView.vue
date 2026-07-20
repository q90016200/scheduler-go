<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import StatusTag from '../components/StatusTag.vue'
import { deleteTaskGroup, listTaskGroups } from '../api/taskGroups'
import type { TaskGroup } from '../types'

const router = useRouter()
const groups = ref<TaskGroup[]>([])

async function load() {
  groups.value = await listTaskGroups()
}

onMounted(load)

async function handleDelete(group: TaskGroup) {
  await ElMessageBox.confirm(`確定刪除群組「${group.name}」？`, '刪除群組', { type: 'warning' })
  await deleteTaskGroup(group.id)
  ElMessage.success('已刪除')
  load()
}
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-button type="primary" @click="router.push('/groups/new')">新增群組</el-button>
    </div>

    <el-table :data="groups" style="width: 100%">
      <el-table-column prop="name" label="名稱" />
      <el-table-column prop="description" label="描述" />
      <el-table-column label="狀態" width="100">
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column label="操作" width="180">
        <template #default="{ row }">
          <el-button size="small" @click="router.push(`/groups/${row.id}/edit`)">編輯</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">刪除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.toolbar {
  margin-bottom: 16px;
}
</style>
