import { describe, expect, it, vi } from 'vitest'
import { http } from '@/shared/api/http'
import { login, loginWithTeams, logout, refresh, register } from './auth.service'

vi.mock('@/shared/api/http', () => ({
  http: {
    post: vi.fn(),
  },
}))

describe('Auth Service', () => {
  it('should call login endpoint with payload', async () => {
    vi.mocked(http.post).mockResolvedValueOnce({ access_token: 'token-123', token_type: 'Bearer', expires_in: 900 })

    const res = await login({ email: 'test@test.com', password: 'password123' })
    expect(http.post).toHaveBeenCalledWith('/auth/login', { email: 'test@test.com', password: 'password123' }, { auth: false })
    expect(res.access_token).toBe('token-123')
  })

  it('should call loginWithTeams endpoint', async () => {
    vi.mocked(http.post).mockResolvedValueOnce({ access_token: 'teams-token', token_type: 'Bearer', expires_in: 900 })

    const res = await loginWithTeams({ sso_token: 'sso-123' })
    expect(http.post).toHaveBeenCalledWith('/auth/teams', { sso_token: 'sso-123' }, { auth: false })
    expect(res.access_token).toBe('teams-token')
  })

  it('should call register endpoint', async () => {
    vi.mocked(http.post).mockResolvedValueOnce({ id: 'u1', full_name: 'Test', email: 't@t.com', role: 'User' })

    const res = await register({ full_name: 'Test', email: 't@t.com', password: 'pw' })
    expect(http.post).toHaveBeenCalledWith('/auth/register', { full_name: 'Test', email: 't@t.com', password: 'pw' }, { auth: false })
    expect(res.id).toBe('u1')
  })

  it('should call refresh endpoint', async () => {
    vi.mocked(http.post).mockResolvedValueOnce({ access_token: 'new-token', token_type: 'Bearer', expires_in: 900 })

    const res = await refresh()
    expect(http.post).toHaveBeenCalledWith('/auth/refresh', undefined, { auth: false })
    expect(res.access_token).toBe('new-token')
  })

  it('should call logout endpoint', async () => {
    vi.mocked(http.post).mockResolvedValueOnce({ message: 'Success' })

    const res = await logout()
    expect(http.post).toHaveBeenCalledWith('/auth/logout')
    expect(res.message).toBe('Success')
  })
})
