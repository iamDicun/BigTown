<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import AuthCard from '../components/AuthCard.vue'
import RegisterForm from '../components/RegisterForm.vue'
import { useAuthStore } from '../stores/auth.store'

const router = useRouter()
const authStore = useAuthStore()
const errorMessage = ref('')
const successMessage = ref('')

async function handleSubmit(payload: { fullName: string; email: string; password: string }) {
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await authStore.register(payload.fullName, payload.email, payload.password)
    successMessage.value = 'Đăng ký thành công. Vui lòng đăng nhập.'
    setTimeout(() => router.push({ name: 'login' }), 1000)
  } catch {
    errorMessage.value = authStore.error
  }
}
</script>

<template>
  <AuthCard>
    <template #title>Đăng ký</template>
    <RegisterForm @submit="handleSubmit">
      <template #error>
        <p v-if="errorMessage" class="pixel-alert pixel-alert--error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="pixel-alert pixel-alert--success">{{ successMessage }}</p>
      </template>
    </RegisterForm>
    <p class="pixel-link-row">
      Đã có tài khoản?
      <router-link :to="{ name: 'login' }">Đăng nhập</router-link>
    </p>
  </AuthCard>
</template>
