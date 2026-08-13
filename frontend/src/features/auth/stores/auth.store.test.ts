import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from './auth.store'
import * as authService from '../services/auth.service'
import * as tokenStorage from '@/shared/api/tokenStorage'

vi.mock('../services/auth.service')
vi.mock('@/shared/api/tokenStorage', () => ({
  getAccessToken: vi.fn(() => null),
  setAccessToken: vi.fn(),
  clearAccessToken: vi.fn(),
}))

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('should initialize with default empty state', () => {
    const store = useAuthStore()
    expect(store.accessToken).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(store.role).toBeNull()
  })

  it('login success updates token and state', async () => {
    vi.mocked(authService.login).mockResolvedValueOnce({ access_token: 'fake.jwt.token', token_type: 'Bearer', expires_in: 900 })

    const store = useAuthStore()
    await store.login('test@test.com', 'pw')

    expect(store.accessToken).toBe('fake.jwt.token')
    expect(store.isAuthenticated).toBe(true)
    expect(tokenStorage.setAccessToken).toHaveBeenCalledWith('fake.jwt.token')
  })

  it('login failure leaves state untouched and sets error', async () => {
    vi.mocked(authService.login).mockRejectedValueOnce(new Error('Invalid credentials'))

    const store = useAuthStore()
    await expect(store.login('wrong@test.com', 'pw')).rejects.toThrow()

    expect(store.accessToken).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(store.error).toBe('Invalid credentials')
  })

  it('logout clears state and token storage', async () => {
    vi.mocked(authService.logout).mockResolvedValueOnce({ message: 'OK' })

    const store = useAuthStore()
    store.accessToken = 'existing.token'

    await store.logout()

    expect(store.accessToken).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(tokenStorage.clearAccessToken).toHaveBeenCalled()
  })
})
