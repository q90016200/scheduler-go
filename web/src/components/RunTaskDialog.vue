<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { cancelRunTask, runTask } from '../api/tasks'
import { listExecutionLogs } from '../api/executionLogs'
import ExecutionLogTable from './ExecutionLogTable.vue'
import type { ExecutionLog, Task } from '../types'

const props = defineProps<{ visible: boolean; task?: Task }>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()

type Stage = 'form' | 'progress'

const stage = ref<Stage>('form')
const submitting = ref(false)
const cancelling = ref(false)
const finished = ref(false)
const timedOut = ref(false)
const logs = ref<ExecutionLog[]>([])
const completedCount = computed(() => logs.value.filter((l) => l.is_final_attempt).length)

const form = reactive({ repeat: 1, intervalMs: 1000 })

let pollTimer: ReturnType<typeof setInterval> | undefined
let baselineLogId = ''
let deadline = 0
let unmounted = false

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }
}

function resetDialog() {
  stopPolling()
  stage.value = 'form'
  logs.value = []
  finished.value = false
  timedOut.value = false
  form.repeat = 1
  form.intervalMs = 1000
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) resetDialog()
  },
)

onBeforeUnmount(() => {
  unmounted = true
  stopPolling()
})

async function poll() {
  if (!props.task) return
  const retryCount = props.task.retry_count ?? 0
  try {
    const result = await listExecutionLogs({
      task_id: props.task.id,
      sort_field: 'started_at',
      sort_order: 'desc',
      limit: form.repeat * (1 + retryCount) + 5,
    })
    logs.value = result.logs
      .filter((l) => l.id > baselineLogId && l.source === 'manual')
      .reverse()
  } catch {
    // 輪詢失敗不中斷，下一輪再試
  }

  if (completedCount.value >= form.repeat) {
    finished.value = true
    stopPolling()
    return
  }
  if (Date.now() > deadline) {
    timedOut.value = true
    stopPolling()
  }
}

async function handleSubmit() {
  if (!props.task) return
  submitting.value = true
  try {
    const baseline = await listExecutionLogs({
      task_id: props.task.id,
      sort_field: 'started_at',
      sort_order: 'desc',
      limit: 1,
    })
    if (unmounted) return
    baselineLogId = baseline.logs[0]?.id ?? ''

    await runTask(props.task.id, { repeat: form.repeat, interval_ms: form.intervalMs })
    if (unmounted) return

    stage.value = 'progress'
    logs.value = []
    finished.value = false
    timedOut.value = false
    const timeoutSeconds = props.task.timeout_seconds || 10
    const retryCount = props.task.retry_count ?? 0
    const retryDelayMs = props.task.retry_delay_ms ?? 0
    const perCallWorstMs = timeoutSeconds * 1000 * (1 + retryCount) + retryDelayMs * retryCount + 2000
    deadline = Date.now() + form.repeat * (form.intervalMs + perCallWorstMs) + 5000
    await poll()
    if (unmounted) return
    pollTimer = setInterval(poll, 1000)
  } catch (err) {
    ElMessage.error((err as Error).message || '觸發執行失敗')
  } finally {
    submitting.value = false
  }
}

async function handleCancelBatch() {
  if (!props.task) return
  cancelling.value = true
  try {
    await cancelRunTask(props.task.id)
    ElMessage.success('已送出取消，尚未開始的次數不會再執行')
    stopPolling()
    finished.value = true
  } catch (err) {
    ElMessage.error((err as Error).message || '取消失敗')
  } finally {
    cancelling.value = false
  }
}

function handleClose() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog :model-value="visible" title="立即執行" width="600px" append-to-body @update:model-value="emit('update:visible', $event)" @close="resetDialog">
    <template v-if="stage === 'form'">
      <el-form label-width="100px">
        <el-form-item label="重複次數">
          <el-input-number v-model="form.repeat" :min="1" :max="50" />
        </el-form-item>
        <el-form-item label="呼叫間隔">
          <el-input-number v-model="form.intervalMs" :min="100" :max="60000" :step="100" />
          <span style="margin-left: 8px; color: var(--el-text-color-secondary)">毫秒</span>
        </el-form-item>
      </el-form>
      <p style="color: var(--el-text-color-secondary); font-size: 13px">
        預估總耗時約 {{ Math.round((form.repeat * form.intervalMs) / 1000) }} 秒（不含每次呼叫實際花費的時間）。
      </p>
    </template>

    <template v-else>
      <el-progress :percentage="Math.min(100, Math.round((completedCount / form.repeat) * 100))" :status="finished ? 'success' : undefined" />
      <p style="margin: 12px 0">已收到 {{ completedCount }} / {{ form.repeat }} 筆執行結果</p>
      <p v-if="timedOut && !finished" style="color: var(--el-color-warning)">等待逾時，請至執行紀錄頁查看完整結果。</p>
      <ExecutionLogTable :logs="logs" :show-task="false" />
      <p style="color: var(--el-text-color-secondary); font-size: 13px; margin-top: 12px">關閉這個視窗不會停止尚未執行的次數，需按下方「取消批次」。</p>
    </template>

    <template #footer>
      <template v-if="stage === 'form'">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">立即執行</el-button>
      </template>
      <template v-else>
        <el-button v-if="!finished" type="danger" :loading="cancelling" @click="handleCancelBatch">取消批次</el-button>
        <el-button type="primary" @click="handleClose">關閉</el-button>
      </template>
    </template>
  </el-dialog>
</template>
