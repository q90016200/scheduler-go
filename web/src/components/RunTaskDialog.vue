<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { cancelRunTask, getRunBatch, runTask } from '../api/tasks'
import ExecutionLogTable from './ExecutionLogTable.vue'
import type { ExecutionLog, Task } from '../types'

const props = defineProps<{ visible: boolean; task?: Task }>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()

const stage = ref<'form' | 'progress'>('form')
const submitting = ref(false)
const cancelling = ref(false)
const finished = ref(false)
// 連續輪詢失敗達上限就放棄：後端連得上時 running 一定會轉 false（重啟後也是），
// 會無限輪詢的只有「後端一直連不上」，所以上限加在連續失敗次數，而不是總等待時間
// （總時間上限得重算後端的逾時/重試規則，長批次也會被誤判）
const MAX_POLL_FAILURES = 30
const lostContact = ref(false)
let pollFailures = 0
const logs = ref<ExecutionLog[]>([])
const completedCount = computed(() => logs.value.filter((l) => l.is_final_attempt).length)

const form = reactive({ repeat: 1, intervalMs: 1000 })

let pollTimer: ReturnType<typeof setInterval> | undefined
let batchId = ''
let unmounted = false
// 上一輪 poll 尚未回來就跳過本輪，避免慢回應重疊、舊快照覆寫新資料
let polling = false

function stopPolling() {
  clearInterval(pollTimer)
  pollTimer = undefined
}

onBeforeUnmount(() => {
  unmounted = true
  stopPolling()
})

// 以後端的 batch_id 查批次進度：running=false 即代表整批結束且 logs 已完整（順序保證見後端 GetRunBatch）
async function poll() {
  if (!props.task || polling) return
  polling = true
  try {
    const batch = await getRunBatch(props.task.id, batchId)
    if (unmounted) return
    pollFailures = 0
    logs.value = batch.logs
    if (!batch.running) {
      finished.value = true
      stopPolling()
    }
  } catch {
    // 單次失敗不中斷，下一輪再試；連續失敗達上限才停止
    if (++pollFailures >= MAX_POLL_FAILURES) {
      lostContact.value = true
      stopPolling()
    }
  } finally {
    polling = false
  }
}

async function handleSubmit() {
  if (!props.task) return
  submitting.value = true
  try {
    const result = await runTask(props.task.id, { repeat: form.repeat, interval_ms: form.intervalMs })
    if (unmounted) return
    batchId = result.batch_id

    stage.value = 'progress'
    logs.value = []
    finished.value = false
    lostContact.value = false
    pollFailures = 0
    await poll()
    if (unmounted) return
    if (!finished.value) pollTimer = setInterval(poll, 1000)
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
    // 不立刻標記完成：繼續輪詢到 running=false，被取消那次嘗試的紀錄才會出現
    ElMessage.success('已送出取消，尚未開始的次數不會再執行')
  } catch (err) {
    ElMessage.error((err as Error).message || '取消失敗')
  } finally {
    cancelling.value = false
  }
}
</script>

<template>
  <el-dialog :model-value="visible" :title="task ? `立即執行：${task.name} #${task.id.slice(-6)}` : '立即執行'" width="600px" append-to-body @update:model-value="emit('update:visible', $event)">
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
      <el-progress :percentage="Math.min(100, Math.round((completedCount / form.repeat) * 100))" :status="finished ? 'success' : lostContact ? 'exception' : undefined" />
      <el-alert v-if="lostContact" type="warning" :closable="false" style="margin-top: 12px" title="已連續 30 秒無法取得批次進度，停止更新。批次可能仍在後端執行，請稍後到執行紀錄查看。" />
      <p style="margin: 12px 0">已收到 {{ completedCount }} / {{ form.repeat }} 筆執行結果</p>
      <ExecutionLogTable :logs="logs" :show-task="false" />
      <p style="color: var(--el-text-color-secondary); font-size: 13px; margin-top: 12px">關閉這個視窗不會停止尚未執行的次數，需按下方「取消批次」。</p>
    </template>

    <template #footer>
      <template v-if="stage === 'form'">
        <el-button @click="emit('update:visible', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">立即執行</el-button>
      </template>
      <template v-else>
        <el-button v-if="!finished && !lostContact" type="danger" :loading="cancelling" @click="handleCancelBatch">取消批次</el-button>
        <el-button type="primary" @click="emit('update:visible', false)">關閉</el-button>
      </template>
    </template>
  </el-dialog>
</template>
