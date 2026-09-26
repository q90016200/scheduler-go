import axios from 'axios'
import type { ApiEnvelope, ValidationFieldError } from '../types'
import router from '../router'

const http = axios.create({
  baseURL: '/api',
})

// 後端驗證錯誤的 field 是 json 欄位名，對應表單上的中文標籤；沒列到的原樣顯示
const FIELD_LABELS: Record<string, string> = {
  name: '名稱',
  description: '描述',
  schedule: '排程',
  type: '類型',
  endpoint: 'Endpoint',
  method: 'Method',
  payload: 'Payload',
  status: '狀態',
  headers: '自訂 Headers',
  retry_count: '重試次數',
  retry_delay_ms: '重試間隔',
  timeout_seconds: '逾時秒數',
  success_status_codes: '成功狀態碼',
  notify_fail_threshold: '失敗通知門檻',
  user_name: '使用者名稱',
  password: '密碼',
  repeat: '重複次數',
  interval_ms: '呼叫間隔',
}

const TAG_MESSAGES: Record<string, (param: string) => string> = {
  required: () => '為必填',
  min: (p) => `下限為 ${p}`,
  max: (p) => `上限為 ${p}`,
  oneof: (p) => `必須是 ${p.split(' ').join('/')} 其中之一`,
  json: () => '必須是合法的 JSON',
  cron: () => '不是合法的 cron 表達式',
  status_codes: () => '格式須如 200,201,2xx',
  rpc_method_format: () => '格式須為 package.Service/MethodName',
}

function formatFieldError(e: ValidationFieldError): string {
  if (!e.field) return `請求格式錯誤：${e.error ?? ''}`
  const label = FIELD_LABELS[e.field] ?? e.field
  const tag = e.tag ?? ''
  const msg = TAG_MESSAGES[tag]?.(e.param ?? '') ?? [tag, e.param].filter(Boolean).join(' ')
  return `${label} ${msg}`
}

function formatEnvelopeError(env?: ApiEnvelope<unknown>): string | undefined {
  const error = env?.error
  if (Array.isArray(error)) return (error as ValidationFieldError[]).map(formatFieldError).join('；')
  if (typeof error === 'string' && error) return error
  return env?.message
}

http.interceptors.response.use(
  ((res: { data: ApiEnvelope<unknown> }) => res.data.data) as any,
  (err) => {
    if (err.response?.status === 401 && router.currentRoute.value.path !== '/login') {
      router.push('/login')
    }
    const env = err.response?.data as ApiEnvelope<unknown> | undefined
    let message = formatEnvelopeError(env) || err.message
    // 5xx 後端刻意不回傳內部錯誤細節，附上 trace 讓使用者回報時可對應伺服器日誌
    if (err.response?.status >= 500 && env?.trace) message += `（trace: ${env.trace}）`
    return Promise.reject(new Error(message))
  },
)

export default http
