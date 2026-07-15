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

// Chỉ để trải nghiệm điều hướng mượt hơn (redirect sớm trước khi render trang sai quyền).
// Đây KHÔNG phải security boundary — backend middleware (RequireRoles, AuthMiddleware) mới là
// nơi thật sự chặn request, xem mục 14 frontend-architecture-it-asset-tracking.md.
export function attachAuthGuard(router: Router) {
  router.beforeEach((to) => {
    const authStore = useAuthStore()

    if (to.meta.guestOnly && authStore.isAuthenticated) {
      return { name: 'dashboard' }
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
