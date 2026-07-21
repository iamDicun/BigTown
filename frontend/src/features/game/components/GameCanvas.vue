<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type Phaser from 'phaser'

import { createGame } from '../phaser/createGame'
import * as realtimeService from '../services/realtime.service'
import { useGameStore } from '../stores/game.store'

const containerEl = ref<HTMLElement | null>(null)
const error = ref('')
const loading = ref(true)

const gameStore = useGameStore()
let game: Phaser.Game | null = null

onMounted(async () => {
  try {
    // characterId và bootstrap là 2 API call độc lập (bootstrap không cần characterId) — chạy song
    // song thay vì tuần tự để không cộng dồn 2 lần round-trip riêng biệt, nhất là khi backend có độ
    // trễ đáng kể (region xa, cold start...).
    const [, bootstrap] = await Promise.all([
      gameStore.characterId ? Promise.resolve() : gameStore.loadMyCharacter(),
      realtimeService.getBootstrap(),
    ])

    if (!gameStore.characterId) {
      throw new Error('Không lấy được character của bạn')
    }

    if (containerEl.value) {
      game = createGame(containerEl.value, bootstrap, gameStore.characterId)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Không thể khởi tạo game'
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  game?.destroy(true)
  game = null
})
</script>

<template>
  <div class="game-canvas-shell">
    <div ref="containerEl" class="game-canvas-mount" />
    <div v-if="loading" class="game-canvas-overlay">
      <p>Đang tải map...</p>
    </div>
    <div v-else-if="error" class="game-canvas-overlay">
      <p class="error">{{ error }}</p>
    </div>
  </div>
</template>

<style scoped>
.game-canvas-shell {
  position: relative;
  width: 100%;
  height: calc(100vh - 54px);
  overflow: hidden;
  background: #1d2a1d;
}

.game-canvas-mount {
  width: 100%;
  height: 100%;
}

.game-canvas-mount :deep(canvas) {
  display: block;
}

.game-canvas-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: #c9c1aa;
  pointer-events: none;
}

.error {
  color: #ffb4a8;
}
</style>
