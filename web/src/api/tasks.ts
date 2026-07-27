import http from './client'
import type { ListTasksResult, SortField, SortOrder, Task, TaskWriteDTO } from '../types'

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

export const deleteTask = (id: string) => http.delete<unknown, void>(`/tasks/${id}`)

export const runTask = (id: string) => http.post<unknown, Task>(`/tasks/${id}/run`)
