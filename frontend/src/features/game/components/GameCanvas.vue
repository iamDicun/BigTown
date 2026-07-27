<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type Phaser from 'phaser'
import { useRouter } from 'vue-router'

import { ApiError } from '@/shared/api/http'
import LoadingSplash from '@/shared/components/LoadingSplash.vue'

import { createGame } from '../phaser/createGame'
import * as realtimeService from '../services/realtime.service'
import { useGameStore } from '../stores/game.store'

const containerEl = ref<HTMLElement | null>(null)
const error = ref('')
const loading = ref(true)
const progress = ref<number | null>(null)

const gameStore = useGameStore()
const router = useRouter()
let game: Phaser.Game | null = null
let readyHandler: ((e: Event) => void) | null = null
let progressHandler: ((e: Event) => void) | null = null
let mapChangedHandler: ((e: Event) => void) | null = null

onMounted(async () => {
  try {
    const [, bootstrap] = await Promise.all([
      gameStore.characterId ? Promise.resolve() : gameStore.loadMyCharacter(),
      realtimeService.getBootstrap(),
    ])

    if (!gameStore.characterId || !gameStore.characterBaseAssetKey) {
      throw new Error('Không lấy được character của bạn')
    }

    if (containerEl.value) {
      game = createGame(
        containerEl.value,
        bootstrap,
        gameStore.characterId,
        gameStore.characterBaseAssetKey,
        gameStore.textureKey,
        gameStore.spritesheetConfig,
        gameStore.characterOptions,
      )

      progress.value = 0

      progressHandler = (e: Event) => {
        progress.value = Math.round(((e as CustomEvent).detail as { value: number }).value * 100)
      }
      window.addEventListener('game:loadProgress', progressHandler)

      readyHandler = () => {
        loading.value = false
      }
      window.addEventListener('game:ready', readyHandler)

      mapChangedHandler = () => {
        progress.value = 0
        loading.value = true
      }
      window.addEventListener('game:mapChanged', mapChangedHandler)
    } else {
      loading.value = false
    }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      await router.replace({ name: 'character-create' })
      return
    }
    error.value = err instanceof Error ? err.message : 'Không thể khởi tạo game'
    loading.value = false
  }
})

onBeforeUnmount(() => {
  if (progressHandler) {
    window.removeEventListener('game:loadProgress', progressHandler)
    progressHandler = null
  }
  if (readyHandler) {
    window.removeEventListener('game:ready', readyHandler)
    readyHandler = null
  }
  if (mapChangedHandler) {
    window.removeEventListener('game:mapChanged', mapChangedHandler)
    mapChangedHandler = null
  }
  game?.destroy(true)
  game = null
})
</script>

<template>
  <div class="game-canvas-shell">
    <div ref="containerEl" class="game-canvas-mount" />
    <LoadingSplash v-if="loading" :progress="progress" />
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
  background: linear-gradient(160deg, #2d5a27 0%, #3c6b34 40%, #5a9c4a 100%);
}

.error {
  font-family: var(--pixel-font);
  font-size: 22px;
  color: #ffb4a8;
}
</style>
