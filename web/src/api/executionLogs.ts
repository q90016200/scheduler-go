import http from './client'
import type { ExecutionLog, ExecutionStatus, ListLogsResult, SortOrder } from '../types'

export const listExecutionLogs = (params: {
  task_id?: string
  task_name?: string
  group_id?: string
  status?: ExecutionStatus
  sort_field?: 'started_at'
  sort_order?: SortOrder
  limit?: number
  offset?: number
}) => http.get<unknown, ListLogsResult>('/tasks/logs/', { params })

export const getExecutionLog = (id: string) => http.get<unknown, ExecutionLog>(`/tasks/logs/${id}`)
