<script setup lang="ts">
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/features/auth/stores/auth.store'

const router = useRouter()
const authStore = useAuthStore()

async function handleLogout() {
  await authStore.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <header class="navbar">
    <span class="navbar-brand">IT Asset Tracking</span>
    <div v-if="authStore.isAuthenticated" class="navbar-user">
      <span>Role: {{ authStore.role }}</span>
      <button @click="handleLogout">Đăng xuất</button>
    </div>
  </header>
</template>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  border-bottom: 1px solid #d0d7de;
  background: #fff;
}

.navbar-user {
  display: flex;
  align-items: center;
  gap: 12px;
}

button {
  font: inherit;
  padding: 6px 12px;
}
</style>
