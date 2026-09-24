import axios from 'axios'
import type { ApiEnvelope, ValidationFieldError } from '../types'
import router from '../router'

const http = axios.create({
  baseURL: '/api',
  withCredentials: true,
})

function formatEnvelopeError(env?: ApiEnvelope<unknown>): string | undefined {
  const error = env?.error
  if (Array.isArray(error)) {
    return (error as ValidationFieldError[])
      .map((e) => [e.field, e.tag, e.param].filter(Boolean).join(' '))
      .join('；')
  }
  if (typeof error === 'string' && error) return error
  return env?.message
}

http.interceptors.response.use(
  ((res: { data: ApiEnvelope<unknown> }) => res.data.data) as any,
  (err) => {
    if (err.response?.status === 401 && router.currentRoute.value.path !== '/login') {
      router.push('/login')
    }
    const env = err.response?.data as ApiEnvelope<unknown> | undefined
    return Promise.reject(new Error(formatEnvelopeError(env) || err.message))
  },
)

export default http
