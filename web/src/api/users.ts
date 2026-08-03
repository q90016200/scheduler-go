import http from './client'
import type { ListUsersResult, Status, User, UserWriteDTO } from '../types'

export const listUsers = (params?: { name?: string; limit?: number; offset?: number }) =>
  http.get<unknown, ListUsersResult>('/users/', { params })

export const getUser = (id: string) => http.get<unknown, User>(`/users/${id}`)

export const createUser = (payload: UserWriteDTO) => http.post<unknown, User>('/users/', payload)

export const updateUser = (id: string, payload: UserWriteDTO) => http.put<unknown, User>(`/users/${id}`, payload)

export const updateUserStatus = (id: string, status: Status) => http.patch<unknown, User>(`/users/${id}/status`, { status })

export const deleteUser = (id: string) => http.delete<unknown, unknown>(`/users/${id}`)
