import http from './client'
import type { DeleteTaskGroupResult, ListTaskGroupsResult, SortOrder, TaskGroup, TaskGroupWriteDTO } from '../types'

export const listTaskGroups = (params?: {
  status?: string
  name?: string
  sort_field?: 'name' | 'created_at' | 'status'
  sort_order?: SortOrder
  limit?: number
  offset?: number
}) => http.get<unknown, ListTaskGroupsResult>('/tasks/group/', { params })

export const getTaskGroup = (id: string) => http.get<unknown, TaskGroup>(`/tasks/group/${id}`)

export const createTaskGroup = (payload: TaskGroupWriteDTO) => http.post<unknown, TaskGroup>('/tasks/group', payload)

export const updateTaskGroup = (id: string, payload: TaskGroupWriteDTO) =>
  http.put<unknown, TaskGroup>(`/tasks/group/${id}`, payload)

export const deleteTaskGroup = (id: string) => http.delete<unknown, DeleteTaskGroupResult>(`/tasks/group/${id}`)
