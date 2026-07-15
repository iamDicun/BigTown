import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'game',
    component: () => import('./views/GameView.vue'),
    meta: { requiresAuth: true },
  },
]

export default routes
