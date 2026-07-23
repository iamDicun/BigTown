import axios from 'axios'

import { clearAccessToken, getAccessToken, setAccessToken } from './tokenStorage'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL
if (!API_BASE_URL) {
  throw new Error('Thiếu biến môi trường VITE_API_BASE_URL — kiểm tra file .env (xem .env.example).')
}

export class ApiError extends Error {
  code: string
  status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

interface RequestOptions {
  /** Có gắn Authorization: Bearer <access_token> hay không. Mặc định true. */
  auth?: boolean
  /** Dùng nội bộ để chặn refresh-loop, không set tay khi gọi từ service. */
  skipRefresh?: boolean
}

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  code?: string
  message?: string
}

const client = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true, // bắt buộc để browser gửi kèm cookie refresh_token HttpOnly
  headers: { 'Content-Type': 'application/json' },
})

async function rawRequest<T>(
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  path: string,
  body: unknown,
  options: RequestOptions,
): Promise<T> {
  const headers: Record<string, string> = {}
  if (options.auth !== false) {
    const token = getAccessToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  try {
    const res = await client.request<ApiEnvelope<T>>({ url: path, method, data: body, headers })

    if (!res.data?.success) {
      throw new ApiError(res.data?.message ?? `Request failed: ${res.status}`, res.data?.code ?? 'UNKNOWN_ERROR', res.status)
    }

    return res.data.data as T
  } catch (err) {
    if (err instanceof ApiError) throw err

    if (axios.isAxiosError(err)) {
      const envelope = err.response?.data as ApiEnvelope<T> | undefined
      throw new ApiError(
        envelope?.message ?? err.message ?? `Request failed: ${err.response?.status ?? 0}`,
        envelope?.code ?? 'UNKNOWN_ERROR',
        err.response?.status ?? 0,
      )
    }

    throw err
  }
}

// Nhiều request có thể cùng nhận 401 một lúc — chỉ cho 1 lần gọi /auth/refresh chạy thật,
// các request còn lại chờ chung promise này rồi tự retry (mục 14.1 frontend-architecture doc).
let refreshPromise: Promise<string> | null = null

export function refreshAccessTokenOnce(): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = rawRequest<{ access_token: string }>('POST', '/auth/refresh', undefined, {
      auth: false,
      skipRefresh: true,
    })
      .then((data) => {
        setAccessToken(data.access_token)
        return data.access_token
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

async function request<T>(
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  path: string,
  body?: unknown,
  options: RequestOptions = {},
): Promise<T> {
  const isAuthEndpoint = path.startsWith('/auth/')

  try {
    return await rawRequest<T>(method, path, body, options)
  } catch (err) {
    if (err instanceof ApiError && err.status === 401 && !options.skipRefresh && !isAuthEndpoint) {
      try {
        await refreshAccessTokenOnce()
      } catch {
        clearAccessToken()
        throw err
      }
      return rawRequest<T>(method, path, body, options)
    }
    throw err
  }
}

export const http = {
  get: <T>(path: string, options?: RequestOptions) => request<T>('GET', path, undefined, options),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) => request<T>('POST', path, body, options),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) => request<T>('PUT', path, body, options),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) => request<T>('PATCH', path, body, options),
  delete: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, undefined, options),
}
