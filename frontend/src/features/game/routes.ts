import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/game',
    name: 'game',
    component: () => import('./views/GameView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/character/create',
    name: 'character-create',
    component: () => import('./views/CharacterCreateView.vue'),
    meta: { requiresAuth: true },
  },
]

export default routes
