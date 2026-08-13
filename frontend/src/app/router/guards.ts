import type { Router } from 'vue-router'

import { useAuthStore } from '@/features/auth/stores/auth.store'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guestOnly?: boolean
    roles?: string[]
    layout?: 'auth' | 'app'
  }
}

export function attachAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const authStore = useAuthStore()

    if (!authStore.sessionReady) {
      await authStore.tryRestoreSession()
    }

    if (to.meta.guestOnly && authStore.isAuthenticated) {
      return { name: 'game' }
    }

    if (to.meta.requiresAuth && !authStore.isAuthenticated) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }

    if (to.meta.roles && to.meta.roles.length > 0) {
      if (!authStore.role || !to.meta.roles.includes(authStore.role)) {
        return { name: 'forbidden' }
      }
    }

    return true
  })
}
