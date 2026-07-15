import { http } from '@/shared/api/http'

export interface LoginPayload {
  email: string
  password: string
}

export interface RegisterPayload {
  full_name: string
  email: string
  password: string
}

export interface AuthTokenResponse {
  access_token: string
  token_type: string
  expires_in: number
}

export interface RegisterResponse {
  id: string
  full_name: string
  email: string
  role: string
}

// Endpoint/payload khớp đúng backend/internal/module/auth/delivery/dto.go — nếu backend đổi
// field, chỉ sửa ở đây, không phải trong từng component.
export function login(payload: LoginPayload) {
  return http.post<AuthTokenResponse>('/auth/login', payload, { auth: false })
}

export function register(payload: RegisterPayload) {
  return http.post<RegisterResponse>('/auth/register', payload, { auth: false })
}

export function refresh() {
  return http.post<AuthTokenResponse>('/auth/refresh', undefined, { auth: false })
}

export function logout() {
  return http.post<{ message: string }>('/auth/logout')
}
