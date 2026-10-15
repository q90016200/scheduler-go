import http from './client'
import type { User } from '../types'

export const login = (payload: { user_name: string; password: string }) =>
  http.post<unknown, void>('/auth/login', payload)

export const logout = () => http.post<unknown, void>('/auth/logout')

export const getMe = () => http.get<unknown, User>('/auth/me')
