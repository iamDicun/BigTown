import { createApp } from 'vue'
import './assets/css/style.css'
import App from './App.vue'
import router from './app/router'
import { pinia } from './app/providers/pinia'
import { useAuthStore } from './features/auth/stores/auth.store'

const app = createApp(App)
app.use(pinia)
app.use(router)

// Thử phục hồi session bằng cookie refresh_token trước khi mount, để tránh nháy màn hình login
// rồi mới redirect vào game nếu user thực ra vẫn còn đăng nhập.
const authStore = useAuthStore()
authStore.tryRestoreSession().finally(() => {
  app.mount('#app')
})
