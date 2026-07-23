import { createRouter, createWebHistory } from 'vue-router'

import authRoutes from '@/features/auth/routes'
import gameRoutes from '@/features/game/routes'
import { useAuthStore } from '@/features/auth/stores/auth.store'

import { attachAuthGuard } from './guards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      redirect: () => {
        const authStore = useAuthStore()
        return { name: authStore.isAuthenticated ? 'game' : 'login' }
      },
    },
    ...authRoutes,
    ...gameRoutes,
    {
      path: '/403',
      name: 'forbidden',
      component: () => import('@/app/views/ForbiddenView.vue'),
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/app/views/NotFoundView.vue'),
    },
  ],
})

attachAuthGuard(router)

export default router
