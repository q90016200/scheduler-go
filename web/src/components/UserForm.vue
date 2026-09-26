<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import StatusSwitch from './StatusSwitch.vue'
import type { User, UserWriteDTO } from '../types'

const props = defineProps<{ initial?: User }>()
const emit = defineEmits<{ submit: [payload: UserWriteDTO] }>()

const formRef = ref<FormInstance>()

const form = reactive<UserWriteDTO>({
  user_name: props.initial?.user_name ?? '',
  password: '',
  status: props.initial?.status ?? 'enabled',
})

const rules: FormRules<UserWriteDTO> = {
  user_name: [{ required: true, message: '請輸入使用者名稱', trigger: 'blur' }],
  password: [{ required: !props.initial, message: '請輸入密碼', trigger: 'blur' }],
}

async function handleSubmit() {
  if (!(await formRef.value?.validate().catch(() => false))) return
  const payload = { ...form }
  if (!payload.password) delete payload.password
  emit('submit', payload)
}

defineExpose({ handleSubmit })
</script>

<template>
  <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
    <el-form-item label="使用者名稱" prop="user_name">
      <el-input v-model="form.user_name" />
    </el-form-item>
    <el-form-item label="密碼" prop="password">
      <el-input v-model="form.password" type="password" show-password :placeholder="initial ? '留空代表不變更' : ''" />
    </el-form-item>
    <el-form-item label="狀態">
      <StatusSwitch v-model="form.status" />
    </el-form-item>
  </el-form>
</template>
