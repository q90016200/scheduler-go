import http from './client'
import type { ListTasksResult, NotifyConfig, RunTaskParams, SortField, SortOrder, Status, Task, TaskStats, TaskWriteDTO } from '../types'

export const listTasks = (params?: {
  status?: string
  group_id?: string
  name?: string
  sort_field?: SortField
  sort_order?: SortOrder
  limit?: number
  offset?: number
}) => http.get<unknown, ListTasksResult>('/tasks/', { params })

export const getTask = (id: string) => http.get<unknown, Task>(`/tasks/${id}`)

export const createTask = (payload: TaskWriteDTO) => http.post<unknown, Task>('/tasks/', payload)

export const updateTask = (id: string, payload: TaskWriteDTO) => http.put<unknown, Task>(`/tasks/${id}`, payload)

export const updateTaskStatus = (id: string, status: Status, userId: string) =>
  http.patch<unknown, Task>(`/tasks/${id}/status`, { status, user_id: userId })

export const deleteTask = (id: string) => http.delete<unknown, void>(`/tasks/${id}`)

export const runTask = (id: string, params: RunTaskParams) => http.post<unknown, void>(`/tasks/${id}/run`, params)

export const cancelRunTask = (id: string) => http.delete<unknown, void>(`/tasks/${id}/run`)

export const getTaskStats = (id: string) => http.get<unknown, TaskStats>(`/tasks/${id}/stats`)

export const getNotifyConfig = () => http.get<unknown, NotifyConfig>('/notify/config')
