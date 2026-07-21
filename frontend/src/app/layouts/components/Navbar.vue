<script setup lang="ts">
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/features/auth/stores/auth.store'
import { useGameStore } from '@/features/game/stores/game.store'

const router = useRouter()
const authStore = useAuthStore()
const gameStore = useGameStore()

async function handleLogout() {
  await authStore.logout()
  // gameStore.characterId không tự mất khi logout (Pinia store sống suốt vòng đời SPA, không theo
  // route) — nếu không reset, lần đăng nhập tiếp theo (tài khoản khác, cùng tab) GameCanvas.vue và
  // ChatPanel.vue thấy characterId đã có sẵn nên bỏ qua loadMyCharacter(), tiếp tục dùng nhầm
  // characterId của tài khoản cũ làm "local player" — kết quả là thấy 2 nhân vật (character thật
  // của tài khoản mới bị hiểu nhầm thành remote), và di chuyển bằng bàn phím lại cập nhật đúng
  // character thật (do server tự xác thực qua JWT) khiến nó "đi theo" như một remote player.
  gameStore.$reset()
  router.push({ name: 'login' })
}
</script>

<template>
  <header class="navbar">
    <span class="navbar-brand">BigTown</span>
    <div v-if="authStore.isAuthenticated" class="navbar-user">
      <span class="pixel-badge pixel-badge--ok">Online</span>
      <button class="pixel-button pixel-button--sm" @click="handleLogout">Đăng xuất</button>
    </div>
  </header>
</template>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 20px;
  background: linear-gradient(180deg, var(--pixel-wood) 0%, var(--pixel-wood-dark) 100%);
  border-bottom: 4px solid var(--pixel-ink);
  box-shadow: 0 4px 0 rgba(0, 0, 0, 0.25);
}

.navbar-brand {
  font-family: var(--pixel-font);
  font-size: 28px;
  letter-spacing: 1px;
  color: var(--pixel-parchment);
  text-shadow: 2px 2px 0 var(--pixel-ink);
}

.navbar-user {
  display: flex;
  align-items: center;
  gap: 14px;
}
</style>
