<script setup lang="ts">
import { reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import type { TaskGroup, TaskGroupWriteDTO } from '../types'

const props = defineProps<{ initial?: TaskGroup }>()
const emit = defineEmits<{ submit: [payload: TaskGroupWriteDTO] }>()

const auth = useAuthStore()

const form = reactive<TaskGroupWriteDTO>({
  name: props.initial?.name ?? '',
  description: props.initial?.description ?? '',
  status: props.initial?.status ?? 'inactive',
  user_id: auth.user?.id ?? '',
})

function handleSubmit() {
  if (!form.name) {
    ElMessage.error('請填寫名稱')
    return
  }
  emit('submit', { ...form })
}

defineExpose({ handleSubmit })
</script>

<template>
  <el-form :model="form" label-width="100px">
    <el-form-item label="名稱" required>
      <el-input v-model="form.name" />
    </el-form-item>
    <el-form-item label="描述">
      <el-input v-model="form.description" type="textarea" />
    </el-form-item>
    <el-form-item label="狀態">
      <el-switch
        v-model="form.status"
        active-value="active"
        inactive-value="inactive"
        active-text="啟用"
        inactive-text="停用"
      />
    </el-form-item>
  </el-form>
</template>
