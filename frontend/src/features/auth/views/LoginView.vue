<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AuthCard from '../components/AuthCard.vue'
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
  <AuthCard>
    <template #title>Đăng nhập</template>
    <LoginForm @submit="handleSubmit">
      <template #error>
        <p v-if="errorMessage" class="pixel-alert pixel-alert--error">{{ errorMessage }}</p>
      </template>
    </LoginForm>
    <p class="pixel-link-row">
      Chưa có tài khoản?
      <router-link :to="{ name: 'register' }">Đăng ký</router-link>
    </p>
  </AuthCard>
</template>
