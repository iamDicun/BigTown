<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

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
  <div class="auth-view">
    <h1>Đăng ký</h1>
    <RegisterForm @submit="handleSubmit">
      <template #error>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="success">{{ successMessage }}</p>
      </template>
    </RegisterForm>
    <p>
      Đã có tài khoản?
      <router-link :to="{ name: 'login' }">Đăng nhập</router-link>
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

.success {
  color: #15803d;
}
</style>
