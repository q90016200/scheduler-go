import http from './client'
import type { SortField, SortOrder, Task, TaskWriteDTO } from '../types'

export const listTasks = (params?: { status?: string; sort_field?: SortField; sort_order?: SortOrder }) =>
  http.get<unknown, { tasks: Task[] }>('/tasks/', { params }).then((res) => res.tasks)

export const getTask = (id: string) => http.get<unknown, Task>(`/tasks/${id}`)

export const createTask = (payload: TaskWriteDTO) => http.post<unknown, Task>('/tasks/', payload)

export const updateTask = (id: string, payload: TaskWriteDTO) => http.put<unknown, Task>(`/tasks/${id}`, payload)

export const deleteTask = (id: string) => http.delete<unknown, void>(`/tasks/${id}`)
