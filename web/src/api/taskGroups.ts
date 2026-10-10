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

// 下拉選單需要「全部群組」，用大 limit 取代真正的全量 API；群組數量大到 1000 筆時再改成搜尋式下拉
export const listAllTaskGroups = async () => (await listTaskGroups({ limit: 1000 })).groups

export const getTaskGroup =(id: string) => http.get<unknown, TaskGroup>(`/tasks/group/${id}`)

export const createTaskGroup = (payload: TaskGroupWriteDTO) => http.post<unknown, TaskGroup>('/tasks/group', payload)

export const updateTaskGroup = (id: string, payload: TaskGroupWriteDTO) =>
  http.put<unknown, TaskGroup>(`/tasks/group/${id}`, payload)

export const deleteTaskGroup = (id: string) => http.delete<unknown, DeleteTaskGroupResult>(`/tasks/group/${id}`)
