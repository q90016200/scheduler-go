<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import StatusSwitch from '../components/StatusSwitch.vue'
import { createTaskGroup, getTaskGroup, updateTaskGroup } from '../api/taskGroups'
import { useAuthStore } from '../stores/auth'
import type { TaskGroupWriteDTO } from '../types'

const props = defineProps<{ id?: string }>()
const router = useRouter()
const auth = useAuthStore()
const loaded = ref(false)
const loading = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<TaskGroupWriteDTO>({
  name: '',
  description: '',
  status: 'disabled',
  user_id: auth.user?.id ?? '',
})

const rules: FormRules<TaskGroupWriteDTO> = {
  name: [{ required: true, message: '請輸入名稱', trigger: 'blur' }],
}

onMounted(async () => {
  if (props.id) {
    const group = await getTaskGroup(props.id)
    form.name = group.name
    form.description = group.description
    form.status = group.status
  }
  loaded.value = true
})

async function handleSubmit() {
  if (!(await formRef.value?.validate().catch(() => false))) return
  loading.value = true
  try {
    if (props.id) {
      await updateTaskGroup(props.id, form)
      ElMessage.success('群組已更新')
    } else {
      await createTaskGroup(form)
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
  <div class="page">
    <div class="page-head">
      <h1>{{ id ? '編輯任務群組' : '新增任務群組' }}</h1>
    </div>
    <el-card v-liquid-glass shadow="never" class="form-card">
      <el-form v-if="!id || loaded" ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名稱" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" />
        </el-form-item>
        <el-form-item label="狀態">
          <StatusSwitch v-model="form.status" />
        </el-form-item>
      </el-form>
      <el-button type="primary" :loading="loading" @click="handleSubmit">儲存</el-button>
    </el-card>
  </div>
</template>

<style scoped>
.form-card {
  max-width: 640px;
}
</style>
