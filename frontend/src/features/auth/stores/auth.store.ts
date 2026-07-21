import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { toErrorMessage } from '@/shared/api/errorMapper'
import { clearAccessToken, getAccessToken, setAccessToken } from '@/shared/api/tokenStorage'
import { decodeJwtPayload } from '@/shared/utils/jwt'

import * as authService from '../services/auth.service'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(getAccessToken())
  const loading = ref(false)
  const error = ref('')

  const payload = computed(() => (accessToken.value ? decodeJwtPayload(accessToken.value) : null))
  const isAuthenticated = computed(() => Boolean(accessToken.value))
  const role = computed(() => payload.value?.role ?? null)
  const userId = computed(() => payload.value?.user_id ?? null)

  async function login(email: string, password: string) {
    loading.value = true
    error.value = ''
    try {
      const data = await authService.login({ email, password })
      setAccessToken(data.access_token)
      accessToken.value = data.access_token
    } catch (err) {
      error.value = toErrorMessage(err, 'Đăng nhập thất bại')
      throw err
    } finally {
      loading.value = false
    }
  }

  async function loginWithTeams(ssoToken: string) {
    loading.value = true
    error.value = ''
    try {
      const data = await authService.loginWithTeams({ sso_token: ssoToken })
      setAccessToken(data.access_token)
      accessToken.value = data.access_token
    } catch (err) {
      error.value = toErrorMessage(err, 'Đăng nhập Teams thất bại')
      throw err
    } finally {
      loading.value = false
    }
  }

  async function register(fullName: string, email: string, password: string) {
    loading.value = true
    error.value = ''
    try {
      await authService.register({ full_name: fullName, email, password })
    } catch (err) {
      error.value = toErrorMessage(err, 'Đăng ký thất bại')
      throw err
    } finally {
      loading.value = false
    }
  }

  // Gọi lúc app khởi động: nếu browser còn cookie refresh_token hợp lệ thì tự đăng nhập lại
  // mà không cần user nhập lại email/password. Thất bại thì coi như chưa đăng nhập, không throw.
  async function tryRestoreSession() {
    if (accessToken.value) return
    try {
      const data = await authService.refresh()
      setAccessToken(data.access_token)
      accessToken.value = data.access_token
    } catch {
      clearAccessToken()
      accessToken.value = null
    }
  }

  async function logout() {
    try {
      await authService.logout()
    } catch {
      // Best-effort: nếu API logout lỗi (mất mạng, thiếu cookie refresh_token do lệch SameSite khi
      // deploy cross-site, access token vừa hết hạn...), vẫn phải xoá session local — không được để
      // lỗi ở đây chặn code gọi sau (Navbar.vue điều hướng về /login), nếu không user tưởng đã đăng
      // xuất nhưng GameView/Centrifuge connection vẫn sống nguyên.
    } finally {
      clearAccessToken()
      accessToken.value = null
    }
  }

  return {
    accessToken,
    loading,
    error,
    isAuthenticated,
    role,
    userId,
    login,
    loginWithTeams,
    register,
    logout,
    tryRestoreSession,
  }
})
