import { createApp } from 'vue'
import './assets/css/style.css'
import App from './App.vue'
import router from './app/router'
import { pinia } from './app/providers/pinia'
import { useAuthStore } from './features/auth/stores/auth.store'
import { initButtonSfx } from './shared/audio/audio.service'

const app = createApp(App)
app.use(pinia)

async function bootstrap() {
  // Phải restore trước khi install router: app.use(router) sẽ chạy initial navigation và auth guard.
  // Nếu guard chạy khi access token runtime còn null, reload ở route protected sẽ bị redirect /login.
  const authStore = useAuthStore()
  await authStore.tryRestoreSession()

  app.use(router)
  await router.isReady()
  initButtonSfx()
  app.mount('#app')
}

void bootstrap()
