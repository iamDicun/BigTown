import { clearAccessToken, getAccessToken, setAccessToken } from './tokenStorage'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api'

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
  method?: string
  body?: unknown
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

async function rawRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, auth = true } = options

  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (auth) {
    const token = getAccessToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    credentials: 'include', // bắt buộc để browser gửi kèm cookie refresh_token HttpOnly
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const json = (await res.json().catch(() => null)) as ApiEnvelope<T> | null

  if (!res.ok || !json?.success) {
    throw new ApiError(json?.message ?? `Request failed: ${res.status}`, json?.code ?? 'UNKNOWN_ERROR', res.status)
  }

  return json.data as T
}

// Nhiều request có thể cùng nhận 401 một lúc — chỉ cho 1 lần gọi /auth/refresh chạy thật,
// các request còn lại chờ chung promise này rồi tự retry (mục 14.1 frontend-architecture doc).
let refreshPromise: Promise<string> | null = null

function refreshAccessTokenOnce(): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = rawRequest<{ access_token: string }>('/auth/refresh', {
      method: 'POST',
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

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const isAuthEndpoint = path.startsWith('/auth/')

  try {
    return await rawRequest<T>(path, options)
  } catch (err) {
    if (err instanceof ApiError && err.status === 401 && !options.skipRefresh && !isAuthEndpoint) {
      try {
        await refreshAccessTokenOnce()
      } catch {
        clearAccessToken()
        throw err
      }
      return rawRequest<T>(path, options)
    }
    throw err
  }
}

export const http = {
  get: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: 'GET' }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'POST', body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'PUT', body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'PATCH', body }),
  delete: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: 'DELETE' }),
}
