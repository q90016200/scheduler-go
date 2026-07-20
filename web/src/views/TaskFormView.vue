<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import TaskForm from '../components/TaskForm.vue'
import { createTask, getTask, updateTask } from '../api/tasks'
import type { Task, TaskWriteDTO } from '../types'

const props = defineProps<{ id?: string }>()
const router = useRouter()
const task = ref<Task>()
const loading = ref(false)

onMounted(async () => {
  if (props.id) {
    task.value = await getTask(props.id)
  }
})

async function handleSubmit(payload: TaskWriteDTO) {
  loading.value = true
  try {
    if (props.id) {
      await updateTask(props.id, payload)
      ElMessage.success('任務已更新')
    } else {
      await createTask(payload)
      ElMessage.success('任務已建立')
    }
    router.push('/tasks')
  } catch (err) {
    ElMessage.error((err as Error).message || '儲存失敗')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <el-card>
    <h2>{{ id ? '編輯任務' : '新增任務' }}</h2>
    <TaskForm v-if="!id || task" :initial="task" @submit="handleSubmit" ref="formRef" />
    <el-button type="primary" :loading="loading" @click="($refs.formRef as any).handleSubmit()">儲存</el-button>
  </el-card>
</template>
