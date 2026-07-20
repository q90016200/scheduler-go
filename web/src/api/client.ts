import axios from 'axios'
import type { ApiEnvelope } from '../types'
import router from '../router'

const http = axios.create({
  baseURL: '/api',
  withCredentials: true,
})

http.interceptors.response.use(
  ((res: { data: ApiEnvelope<unknown> }) => res.data.data) as any,
  (err) => {
    if (err.response?.status === 401 && router.currentRoute.value.path !== '/login') {
      router.push('/login')
    }
    const env = err.response?.data as ApiEnvelope<unknown> | undefined
    return Promise.reject(new Error((env?.error as string) || env?.message || err.message))
  },
)

export default http
