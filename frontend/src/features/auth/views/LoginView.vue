<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import LoginForm from '../components/LoginForm.vue'
import { useAuthStore } from '../stores/auth.store'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const errorMessage = ref('')

async function handleSubmit(payload: { email: string; password: string }) {
  errorMessage.value = ''
  try {
    await authStore.login(payload.email, payload.password)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.push(redirect)
  } catch {
    errorMessage.value = authStore.error
  }
}
</script>

<template>
  <div class="auth-view">
    <h1>Đăng nhập</h1>
    <LoginForm @submit="handleSubmit">
      <template #error>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
      </template>
    </LoginForm>
    <p>
      Chưa có tài khoản?
      <router-link :to="{ name: 'register' }">Đăng ký</router-link>
    </p>
  </div>
</template>

<style scoped>
.auth-view {
  background: #fff;
  padding: 32px;
  border: 1px solid #d0d7de;
  border-radius: 8px;
}

.error {
  color: #b91c1c;
}
</style>
