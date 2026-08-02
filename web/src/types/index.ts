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
  status: boolean
}

export type TaskType = 'http' | 'rpc'
export type ActiveStatus = 'active' | 'inactive'
export type ExecutionStatus = 'success' | 'failed'
export type SortField = 'name' | 'created_at' | 'status' | 'schedule'
export type SortOrder = 'asc' | 'desc'

export interface TaskGroup {
  id: string
  name: string
  description: string
  status: ActiveStatus
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
  status: ActiveStatus
  created_by: string
  updated_by: string
  created_at: number
  updated_at: number
  group?: TaskGroup | null
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
  status?: ActiveStatus
  user_id: string
}

export type TaskGroupWriteDTO = {
  name: string
  description: string
  status?: ActiveStatus
  user_id: string
}

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
