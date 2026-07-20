import http from './client'
import type { TaskGroup, TaskGroupWriteDTO } from '../types'

export const listTaskGroups = () => http.get<unknown, { groups: TaskGroup[] }>('/tasks/group/').then((res) => res.groups)

export const getTaskGroup = (id: string) => http.get<unknown, TaskGroup>(`/tasks/group/${id}`)

export const createTaskGroup = (payload: TaskGroupWriteDTO) => http.post<unknown, TaskGroup>('/tasks/group', payload)

export const updateTaskGroup = (id: string, payload: TaskGroupWriteDTO) =>
  http.put<unknown, TaskGroup>(`/tasks/group/${id}`, payload)

export const deleteTaskGroup = (id: string) => http.delete<unknown, void>(`/tasks/group/${id}`)
