<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import CronInput from './CronInput.vue'
import StatusSwitch from './StatusSwitch.vue'
import { listAllTaskGroups } from '../api/taskGroups'
import { getNotifyConfig } from '../api/tasks'
import { useAuthStore } from '../stores/auth'
import type { NotifyConfig, Task, TaskGroup, TaskWriteDTO } from '../types'

const props = defineProps<{ initial?: Task }>()
const emit = defineEmits<{ submit: [payload: TaskWriteDTO] }>()

const auth = useAuthStore()
const groups = ref<TaskGroup[]>([])
const formRef = ref<FormInstance>()
const notifyConfig = ref<NotifyConfig>()

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
  retry_count: props.initial?.retry_count ?? 0,
  retry_delay_ms: props.initial?.retry_delay_ms ?? 0,
  timeout_seconds: props.initial?.timeout_seconds ?? 10,
  headers: props.initial?.headers ?? '{}',
  success_status_codes: props.initial?.success_status_codes ?? '',
  notify_fail_threshold: props.initial?.notify_fail_threshold ?? 0,
  notify_telegram: props.initial?.notify_telegram ?? false,
  notify_slack: props.initial?.notify_slack ?? false,
})

// headers 用 key-value 列表編輯，送出時再序列化成 JSON 字串存進 form.headers
function parseHeaderRows(raw?: string) {
  if (!raw) return []
  try {
    const obj = JSON.parse(raw) as Record<string, string>
    return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }))
  } catch {
    return []
  }
}
const headerRows = ref(parseHeaderRows(props.initial?.headers))

watch(
  () => form.type,
  (type) => {
    form.method = type === 'http' ? 'GET' : ''
  },
)

// 兩支分開處理：通知設定只用來顯示警告，失敗不該連帶讓群組下拉選單變空
onMounted(() => {
  listAllTaskGroups()
    .then((g) => (groups.value = g))
    .catch((e: Error) => ElMessage.error(e.message))
  getNotifyConfig()
    .then((c) => (notifyConfig.value = c))
    .catch(() => {})
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
  const headers = Object.fromEntries(
    headerRows.value.filter((r) => r.key.trim()).map((r) => [r.key.trim(), r.value]),
  )
  emit('submit', { ...form, headers: JSON.stringify(headers) })
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
        <el-option v-for="m in HTTP_METHODS" :key="m" :label="m" :value="m" />
      </el-select>
      <el-input v-else v-model="form.method" placeholder="package.Service/MethodName" />
    </el-form-item>
    <el-form-item label="Payload" prop="payload">
      <el-input v-model="form.payload" type="textarea" :rows="4" placeholder='{"key":"value"}' />
    </el-form-item>
    <el-form-item label="自訂 Headers">
      <div v-for="(row, index) in headerRows" :key="index" class="header-row">
        <el-input v-model="row.key" placeholder="Key，如 Authorization" style="width: 200px" />
        <el-input v-model="row.value" placeholder="Value" style="width: 260px" />
        <el-button :icon="Delete" circle @click="headerRows.splice(index, 1)" />
      </div>
      <el-button :icon="Plus" @click="headerRows.push({ key: '', value: '' })">新增 Header</el-button>
    </el-form-item>
    <el-form-item v-if="form.type === 'http'" label="成功狀態碼" prop="success_status_codes">
      <el-input v-model="form.success_status_codes" placeholder="如 200,201,2xx" style="width: 260px" />
      <span class="hint">留空則 HTTP 400 以下皆視為成功</span>
    </el-form-item>
    <el-form-item label="失敗重試">
      <el-input-number v-model="form.retry_count" :min="0" :max="5" />
      <span class="hint">次，間隔</span>
      <el-input-number v-model="form.retry_delay_ms" :min="0" :max="30000" :step="500" />
      <span class="hint">毫秒</span>
    </el-form-item>
    <el-form-item label="逾時秒數">
      <el-input-number v-model="form.timeout_seconds" :min="1" :max="120" />
    </el-form-item>
    <el-form-item label="失敗通知">
      <div>
        <el-input-number v-model="form.notify_fail_threshold" :min="0" :max="20" />
        <span class="hint">連續失敗達此次數才通知，0 表示不啟用</span>
      </div>
      <div>
        <el-checkbox v-model="form.notify_telegram">Telegram</el-checkbox>
        <el-checkbox v-model="form.notify_slack">Slack</el-checkbox>
      </div>
      <div v-if="notifyConfig">
        <span v-if="form.notify_telegram && !notifyConfig.telegram_configured" class="notify-warning">Telegram 尚未設定，通知不會送出</span>
        <span v-if="form.notify_slack && !notifyConfig.slack_configured" class="notify-warning">Slack 尚未設定，通知不會送出</span>
      </div>
    </el-form-item>
    <el-form-item label="狀態">
      <StatusSwitch v-model="form.status" />
    </el-form-item>
  </el-form>
</template>

<style scoped>
.header-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.hint {
  margin: 0 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.notify-warning {
  display: block;
  color: var(--el-color-danger);
  font-size: 13px;
}
</style>
