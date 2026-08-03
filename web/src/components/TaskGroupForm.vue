<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import type { TaskGroup, TaskGroupWriteDTO } from '../types'

const props = defineProps<{ initial?: TaskGroup }>()
const emit = defineEmits<{ submit: [payload: TaskGroupWriteDTO] }>()

const auth = useAuthStore()
const formRef = ref<FormInstance>()

const form = reactive<TaskGroupWriteDTO>({
  name: props.initial?.name ?? '',
  description: props.initial?.description ?? '',
  status: props.initial?.status ?? 'disabled',
  user_id: auth.user?.id ?? '',
})

const rules: FormRules<TaskGroupWriteDTO> = {
  name: [{ required: true, message: '請輸入名稱', trigger: 'blur' }],
}

async function handleSubmit() {
  if (!(await formRef.value?.validate().catch(() => false))) return
  emit('submit', { ...form })
}

defineExpose({ handleSubmit })
</script>

<template>
  <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
    <el-form-item label="名稱" prop="name">
      <el-input v-model="form.name" />
    </el-form-item>
    <el-form-item label="描述">
      <el-input v-model="form.description" type="textarea" />
    </el-form-item>
    <el-form-item label="狀態">
      <el-switch
        v-model="form.status"
        active-value="enabled"
        inactive-value="disabled"
        active-text="啟用"
        inactive-text="停用"
      />
    </el-form-item>
  </el-form>
</template>
