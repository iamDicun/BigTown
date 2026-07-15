import { createRouter, createWebHistory } from 'vue-router'

import authRoutes from '@/features/auth/routes'
import gameRoutes from '@/features/game/routes'

import { attachAuthGuard } from './guards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
