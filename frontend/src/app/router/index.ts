import { createRouter, createWebHistory } from 'vue-router'

import authRoutes from '@/features/auth/routes'
import dashboardRoutes from '@/features/dashboard/routes'

import { attachAuthGuard } from './guards'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    ...authRoutes,
    ...dashboardRoutes,
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
