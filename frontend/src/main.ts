import { createApp } from 'vue'
import './assets/css/style.css'
import App from './App.vue'
import router from './app/router'
import { pinia } from './app/providers/pinia'
import { useAuthStore } from './features/auth/stores/auth.store'
import { initButtonSfx } from './shared/audio/audio.service'

async function bootstrap() {
  initButtonSfx()

  const app = createApp(App)
  app.use(pinia)

  const authStore = useAuthStore()
  authStore.tryRestoreSession()

  app.use(router)
  app.mount('#app')

  await router.isReady()
}

void bootstrap()
