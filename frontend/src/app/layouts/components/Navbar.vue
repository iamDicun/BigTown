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
    <span class="navbar-brand">BigTown</span>
    <div v-if="authStore.isAuthenticated" class="navbar-user">
      <span class="navbar-status">Online</span>
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
  border-bottom: 1px solid #2f3a2f;
  background: #172017;
  color: #f3ead7;
}

.navbar-brand {
  font-weight: 700;
  letter-spacing: 0.04em;
}

.navbar-user {
  display: flex;
  align-items: center;
  gap: 12px;
}

button {
  font: inherit;
  padding: 6px 12px;
  border: 1px solid #6f7f55;
  border-radius: 6px;
  background: #2f3a2f;
  color: #f3ead7;
  cursor: pointer;
}

.navbar-status {
  color: #9fd47f;
}
</style>
