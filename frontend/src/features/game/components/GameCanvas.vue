<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type Phaser from 'phaser'
import { useRouter } from 'vue-router'

import { ApiError } from '@/shared/api/http'
import logo from '@/assets/images/logo.png'

import { createGame } from '../phaser/createGame'
import * as realtimeService from '../services/realtime.service'
import { useGameStore } from '../stores/game.store'

const containerEl = ref<HTMLElement | null>(null)
const error = ref('')
const loading = ref(true)
const progress = ref(0)
const phaserLoading = ref(false)

const CIRCLE_R = 58
const CIRCUMFERENCE = 2 * Math.PI * CIRCLE_R

const strokeDashOffset = computed(() => CIRCUMFERENCE * (1 - progress.value / 100))

const gameStore = useGameStore()
const router = useRouter()
let game: Phaser.Game | null = null
let readyHandler: ((e: Event) => void) | null = null
let progressHandler: ((e: Event) => void) | null = null

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

      phaserLoading.value = true

      progressHandler = (e: Event) => {
        progress.value = Math.round(((e as CustomEvent).detail as { value: number }).value * 100)
      }
      window.addEventListener('game:loadProgress', progressHandler)

      readyHandler = () => {
        loading.value = false
      }
      window.addEventListener('game:ready', readyHandler, { once: true })
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
  game?.destroy(true)
  game = null
})
</script>

<template>
  <div class="game-canvas-shell">
    <div ref="containerEl" class="game-canvas-mount" />
    <div v-if="loading" class="loader-overlay">
      <div class="loader-content">
        <svg class="loader-ring" :viewBox="`0 0 ${(CIRCLE_R + 6) * 2} ${(CIRCLE_R + 6) * 2}`">
          <circle class="loader-ring-bg" :cx="CIRCLE_R + 6" :cy="CIRCLE_R + 6" :r="CIRCLE_R" />
          <circle
            class="loader-ring-fg"
            :cx="CIRCLE_R + 6"
            :cy="CIRCLE_R + 6"
            :r="CIRCLE_R"
            :stroke-dasharray="CIRCUMFERENCE"
            :stroke-dashoffset="strokeDashOffset"
          />
        </svg>
        <img class="loader-logo" :src="logo" alt="BigTown" />
        <p class="loader-pct">{{ progress }}%</p>
        <p class="loader-label">Đang tải tài nguyên…</p>
      </div>
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
  background: linear-gradient(160deg, #2d5a27 0%, #3c6b34 40%, #5a9c4a 100%);
}

.game-canvas-mount {
  width: 100%;
  height: 100%;
}

.game-canvas-mount :deep(canvas) {
  display: block;
}

.loader-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: linear-gradient(160deg, #2d5a27 0%, #3c6b34 40%, #5a9c4a 100%);
}

.loader-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  position: relative;
  width: 160px;
  height: 160px;
}

.loader-ring {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.loader-ring-bg {
  fill: none;
  stroke: rgba(255, 255, 255, 0.12);
  stroke-width: 5;
}

.loader-ring-fg {
  fill: none;
  stroke: #fdf1d6;
  stroke-width: 5;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.15s linear;
}

.loader-logo {
  position: relative;
  z-index: 1;
  width: 64px;
  height: 64px;
  image-rendering: pixelated;
  margin-top: 24px;
}

.loader-pct {
  font-family: var(--pixel-font);
  font-size: 32px;
  color: #fdf1d6;
  text-shadow: 2px 2px 0 rgba(0, 0, 0, 0.3);
  margin: 0;
  line-height: 1;
}

.loader-label {
  font-family: var(--pixel-font);
  font-size: 22px;
  color: rgba(253, 241, 214, 0.75);
  text-shadow: 1px 1px 0 rgba(0, 0, 0, 0.25);
  margin: 0;
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
  font-family: var(--pixel-font);
  font-size: 22px;
  color: #ffb4a8;
}
</style>
