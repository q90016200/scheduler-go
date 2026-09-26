<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh } from '@element-plus/icons-vue'
import UserForm from '../components/UserForm.vue'
import IconButton from '../components/IconButton.vue'
import StatusSwitch from '../components/StatusSwitch.vue'
import { createUser, deleteUser, listUsers, updateUser, updateUserStatus } from '../api/users'
import { usePagedList } from '../composables/usePagedList'
import type { Status, User, UserWriteDTO } from '../types'

const statusUpdatingIds = ref(new Set<string>())
const filters = reactive({ name: '' })

const dialogVisible = ref(false)
const editingUser = ref<User>()
const saving = ref(false)
const formRef = ref<InstanceType<typeof UserForm>>()

const { items: users, total, page, pageSize, loading, load, reloadFromFirstPage } = usePagedList<User>((p, ps) =>
  listUsers({
    name: filters.name || undefined,
    limit: ps,
    offset: (p - 1) * ps,
  }).then((r) => ({ items: r.users, total: r.total })),
)

onMounted(load)

function openCreate() {
  editingUser.value = undefined
  dialogVisible.value = true
}

function openEdit(user: User) {
  editingUser.value = user
  dialogVisible.value = true
}

async function handleSubmit(payload: UserWriteDTO) {
  saving.value = true
  try {
    if (editingUser.value) {
      await updateUser(editingUser.value.id, payload)
      ElMessage.success('使用者已更新')
    } else {
      await createUser(payload)
      ElMessage.success('使用者已建立')
    }
    dialogVisible.value = false
    load()
  } catch (err) {
    ElMessage.error((err as Error).message || '儲存失敗')
  } finally {
    saving.value = false
  }
}

async function handleStatusChange(user: User, status: Status) {
  statusUpdatingIds.value.add(user.id)
  try {
    await updateUserStatus(user.id, status)
    ElMessage.success('狀態已更新')
    await load()
  } catch (err) {
    ElMessage.error((err as Error).message || '狀態更新失敗')
  } finally {
    statusUpdatingIds.value.delete(user.id)
  }
}

async function handleDelete(user: User) {
  await ElMessageBox.confirm(`確定刪除使用者「${user.user_name}」？`, '刪除使用者', { type: 'warning' })
  await deleteUser(user.id)
  ElMessage.success('已刪除')
  load()
}
</script>

<template>
  <el-card>
    <div class="toolbar filters">
      <el-input v-model="filters.name" placeholder="名稱篩選" clearable style="width: 160px" @change="reloadFromFirstPage" @clear="reloadFromFirstPage" />
      <IconButton :icon="Plus" label="新增使用者" type="primary" @click="openCreate" />
      <IconButton :icon="Refresh" label="重新整理" :loading="loading" @click="() => load()" />
    </div>

    <el-table :data="users" style="width: 100%">
      <el-table-column prop="user_name" label="使用者名稱" />
      <el-table-column label="狀態" width="140">
        <template #default="{ row }">
          <StatusSwitch
            :model-value="row.status"
            :loading="statusUpdatingIds.has(row.id)"
            @change="(val: Status) => handleStatusChange(row, val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button-group>
            <IconButton :icon="Edit" label="編輯" @click="openEdit(row)" />
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

    <el-dialog v-model="dialogVisible" :title="editingUser ? '編輯使用者' : '新增使用者'" width="500px" append-to-body>
      <UserForm v-if="dialogVisible" ref="formRef" :initial="editingUser" @submit="handleSubmit" />
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="formRef?.handleSubmit()">儲存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>
