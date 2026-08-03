<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import CronInput from './CronInput.vue'
import { listTaskGroups } from '../api/taskGroups'
import { useAuthStore } from '../stores/auth'
import type { Task, TaskGroup, TaskWriteDTO } from '../types'

const props = defineProps<{ initial?: Task }>()
const emit = defineEmits<{ submit: [payload: TaskWriteDTO] }>()

const auth = useAuthStore()
const groups = ref<TaskGroup[]>([])
const formRef = ref<FormInstance>()

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

const form = reactive<TaskWriteDTO>({
  group_id: props.initial?.group_id ?? null,
  name: props.initial?.name ?? '',
  description: props.initial?.description ?? '',
  schedule: props.initial?.schedule ?? '',
  type: props.initial?.type ?? 'http',
  endpoint: props.initial?.endpoint ?? '',
  method: props.initial?.method ?? 'GET',
  payload: props.initial?.payload ?? '',
  status: props.initial?.status ?? 'disabled',
  user_id: auth.user?.id ?? '',
})

watch(
  () => form.type,
  (type) => {
    form.method = type === 'http' ? 'GET' : ''
  },
)

onMounted(async () => {
  // ponytail: 下拉選單需要「全部群組」，用大 limit 取代真正的全量 API；群組數量大到 1000 筆時再改成搜尋式下拉
  groups.value = (await listTaskGroups({ limit: 1000 })).groups
})

const rules: FormRules<TaskWriteDTO> = {
  name: [{ required: true, message: '請輸入名稱', trigger: 'blur' }],
  schedule: [{ required: true, message: '請輸入排程 (cron)', trigger: 'blur' }],
  endpoint: [{ required: true, message: '請輸入 Endpoint', trigger: 'blur' }],
  method: [
    {
      required: true,
      validator: (_rule, value: string, callback) => {
        if (!value) return callback(new Error('請輸入 Method'))
        if (form.type === 'http' && !HTTP_METHODS.includes(value)) return callback(new Error(`Method 必須是 ${HTTP_METHODS.join('/')} 其中之一`))
        if (form.type === 'rpc' && !/^[\w.]+\/\w+$/.test(value)) return callback(new Error('Method 格式須為 package.Service/MethodName'))
        callback()
      },
      trigger: ['blur', 'change'],
    },
  ],
  payload: [
    {
      validator: (_rule, value: string, callback) => {
        if (!value.trim()) return callback()
        try {
          JSON.parse(value)
          callback()
        } catch {
          callback(new Error('Payload 必須是合法的 JSON 字串'))
        }
      },
      trigger: 'blur',
    },
  ],
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
    <el-form-item label="群組">
      <el-select v-model="form.group_id" clearable placeholder="不選擇群組">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
    </el-form-item>
    <el-form-item label="排程 (cron)" prop="schedule">
      <CronInput v-model="form.schedule" />
    </el-form-item>
    <el-form-item label="類型" required>
      <el-radio-group v-model="form.type">
        <el-radio value="http">HTTP</el-radio>
        <el-radio value="rpc">RPC</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="Endpoint" prop="endpoint">
      <el-input
        v-model="form.endpoint"
        :placeholder="form.type === 'http' ? 'https://example.com/api/...' : 'host:port'"
      />
    </el-form-item>
    <el-form-item label="Method" prop="method">
      <el-select v-if="form.type === 'http'" v-model="form.method">
        <el-option label="GET" value="GET" />
        <el-option label="POST" value="POST" />
        <el-option label="PUT" value="PUT" />
        <el-option label="DELETE" value="DELETE" />
        <el-option label="PATCH" value="PATCH" />
      </el-select>
      <el-input v-else v-model="form.method" placeholder="package.Service/MethodName" />
    </el-form-item>
    <el-form-item label="Payload" prop="payload">
      <el-input v-model="form.payload" type="textarea" :rows="4" placeholder='{"key":"value"}' />
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
