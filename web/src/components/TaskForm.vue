<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import CronInput from './CronInput.vue'
import { listTaskGroups } from '../api/taskGroups'
import { useAuthStore } from '../stores/auth'
import type { Task, TaskGroup, TaskWriteDTO } from '../types'

const props = defineProps<{ initial?: Task }>()
const emit = defineEmits<{ submit: [payload: TaskWriteDTO] }>()

const auth = useAuthStore()
const groups = ref<TaskGroup[]>([])

const form = reactive<TaskWriteDTO>({
  group_id: props.initial?.group_id ?? null,
  name: props.initial?.name ?? '',
  description: props.initial?.description ?? '',
  schedule: props.initial?.schedule ?? '',
  type: props.initial?.type ?? 'http',
  endpoint: props.initial?.endpoint ?? '',
  method: props.initial?.method ?? 'GET',
  payload: props.initial?.payload ?? '',
  status: props.initial?.status ?? 'inactive',
  user_id: auth.user?.id ?? '',
})

onMounted(async () => {
  groups.value = await listTaskGroups()
})

function validatePayload(): boolean {
  if (!form.payload.trim()) return true
  try {
    JSON.parse(form.payload)
    return true
  } catch {
    ElMessage.error('Payload 必須是合法的 JSON 字串')
    return false
  }
}

function handleSubmit() {
  if (!form.name || !form.schedule || !form.endpoint) {
    ElMessage.error('請填寫必填欄位')
    return
  }
  if (!validatePayload()) return
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
    <el-form-item label="群組">
      <el-select v-model="form.group_id" clearable placeholder="不選擇群組">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
    </el-form-item>
    <el-form-item label="排程 (cron)" required>
      <CronInput v-model="form.schedule" />
    </el-form-item>
    <el-form-item label="類型" required>
      <el-radio-group v-model="form.type">
        <el-radio value="http">HTTP</el-radio>
        <el-radio value="rpc">RPC</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="Endpoint" required>
      <el-input
        v-model="form.endpoint"
        :placeholder="form.type === 'http' ? 'https://example.com/api/...' : 'host:port'"
      />
    </el-form-item>
    <el-form-item label="Method">
      <el-select v-if="form.type === 'http'" v-model="form.method">
        <el-option label="GET" value="GET" />
        <el-option label="POST" value="POST" />
        <el-option label="PUT" value="PUT" />
        <el-option label="DELETE" value="DELETE" />
      </el-select>
      <el-input v-else v-model="form.method" placeholder="package.Service/MethodName" />
    </el-form-item>
    <el-form-item label="Payload">
      <el-input v-model="form.payload" type="textarea" :rows="4" placeholder="JSON 字串" />
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
