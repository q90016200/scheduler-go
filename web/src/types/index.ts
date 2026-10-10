// 驗證失敗為 {field, tag, param}；JSON 本身解不開（型別不符等）時後端只帶 {error}
export interface ValidationFieldError {
  field?: string
  tag?: string
  param?: string
  error?: string
}

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  error: string | ValidationFieldError[] | unknown
  trace: string
}

export interface User {
  id: string
  user_name: string
  status: Status
}

export type TaskType = 'http' | 'rpc'
export type Status = 'enabled' | 'disabled'
export type ExecutionStatus = 'success' | 'failed'
export type SortField = 'name' | 'created_at' | 'status' | 'schedule'
export type SortOrder = 'asc' | 'desc'

export interface TaskGroup {
  id: string
  name: string
  description: string
  status: Status
  created_by: string
  updated_by: string
  created_at: number
  updated_at: number
}

export interface Task {
  id: string
  group_id: string | null
  name: string
  description: string
  schedule: string
  type: TaskType
  endpoint: string
  method: string
  payload: string
  status: Status
  created_by: string
  updated_by: string
  created_at: number
  updated_at: number
  group?: TaskGroup | null
  retry_count: number
  retry_delay_ms: number
  timeout_seconds: number
  headers: string
  success_status_codes: string
  notify_fail_threshold: number
  notify_telegram: boolean
  notify_slack: boolean
  consecutive_failures: number
  next_run_at?: number | null
}

export type TaskWriteDTO = {
  group_id: string | null
  name: string
  description: string
  schedule: string
  type: TaskType
  endpoint: string
  method: string
  payload: string
  status?: Status
  user_id: string
  retry_count: number
  retry_delay_ms: number
  timeout_seconds: number
  headers: string
  success_status_codes: string
  notify_fail_threshold: number
  notify_telegram: boolean
  notify_slack: boolean
}

export interface TaskStats {
  success_rate_30d: number
  avg_duration_ms_30d: number
  recent_fail_count: number
}

export type TaskGroupWriteDTO = {
  name: string
  description: string
  status?: Status
  user_id: string
}

export type ExecutionSource = 'cron' | 'manual'

export interface ExecutionLog {
  id: string
  task_id: string
  status: ExecutionStatus
  started_at: number
  finished_at: number
  output?: string
  error_message?: string
  created_at: number
  task?: Task
  source: ExecutionSource
  batch_id: string
  attempt_number: number
  is_final_attempt: boolean
}

export type Paged<K extends string, T> = Record<K, T[]> & { total: number; limit: number; offset: number }

export type ListLogsResult = Paged<'logs', ExecutionLog>
export type ListTasksResult = Paged<'tasks', Task>
export type ListTaskGroupsResult = Paged<'groups', TaskGroup>

export interface DeleteTaskGroupResult {
  deactivated_tasks: { id: string; name: string }[]
}

export interface RunTaskParams {
  repeat: number
  interval_ms: number
}

export interface RunTaskResult {
  task_id: string
  repeat: number
  interval_ms: number
  batch_id: string
}

export interface RunBatch {
  batch_id: string
  running: boolean
  logs: ExecutionLog[]
}

export type UserWriteDTO = {
  user_name: string
  password?: string
  status?: Status
}

export type ListUsersResult = Paged<'users', User>

export interface DashboardTaskRef {
  task_id: string
  task_name: string
}

export type DashboardFailingTask = DashboardTaskRef & { fail_count: number }

export interface DashboardStats {
  total_tasks_enabled: number
  total_tasks_disabled: number
  executions_24h: number
  success_rate_24h: number
  top_failing_tasks_24h: DashboardFailingTask[]
  currently_failing: DashboardTaskRef[]
}

export interface NotifyConfig {
  telegram_configured: boolean
  slack_configured: boolean
}
