<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import TaskGroupForm from '../components/TaskGroupForm.vue'
import { createTaskGroup, getTaskGroup, updateTaskGroup } from '../api/taskGroups'
import type { TaskGroup, TaskGroupWriteDTO } from '../types'

const props = defineProps<{ id?: string }>()
const router = useRouter()
const group = ref<TaskGroup>()
const loading = ref(false)

onMounted(async () => {
  if (props.id) {
    group.value = await getTaskGroup(props.id)
  }
})

async function handleSubmit(payload: TaskGroupWriteDTO) {
  loading.value = true
  try {
    if (props.id) {
      await updateTaskGroup(props.id, payload)
      ElMessage.success('群組已更新')
    } else {
      await createTaskGroup(payload)
      ElMessage.success('群組已建立')
    }
    router.push('/groups')
  } catch (err) {
    ElMessage.error((err as Error).message || '儲存失敗')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <el-card>
    <h2>{{ id ? '編輯任務群組' : '新增任務群組' }}</h2>
    <TaskGroupForm v-if="!id || group" :initial="group" @submit="handleSubmit" ref="formRef" />
    <el-button type="primary" :loading="loading" @click="($refs.formRef as any).handleSubmit()">儲存</el-button>
  </el-card>
</template>
