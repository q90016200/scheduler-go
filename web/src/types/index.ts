export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  error: unknown
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
  notify_fail_threshold: number
  notify_telegram: boolean
  notify_slack: boolean
  notify_email: boolean
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
  notify_fail_threshold: number
  notify_telegram: boolean
  notify_slack: boolean
  notify_email: boolean
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
  attempt_number: number
  is_final_attempt: boolean
}

export interface ListLogsResult {
  logs: ExecutionLog[]
  total: number
  limit: number
  offset: number
}

export interface ListTasksResult {
  tasks: Task[]
  total: number
  limit: number
  offset: number
}

export interface ListTaskGroupsResult {
  groups: TaskGroup[]
  total: number
  limit: number
  offset: number
}

export interface DeactivatedTaskRef {
  id: string
  name: string
}

export interface DeleteTaskGroupResult {
  deactivated_tasks: DeactivatedTaskRef[]
}

export interface RunTaskParams {
  repeat: number
  interval_ms: number
}

export interface RunTaskResult {
  task_id: string
  repeat: number
  interval_ms: number
}

export type UserWriteDTO = {
  user_name: string
  password?: string
  status?: Status
}

export interface ListUsersResult {
  users: User[]
  total: number
  limit: number
  offset: number
}
