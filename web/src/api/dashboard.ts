import http from './client'
import type { DashboardStats } from '../types'

export const getDashboardStats = () => http.get<unknown, DashboardStats>('/dashboard/stats')
