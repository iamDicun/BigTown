<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'

import { playMusic, stopMusic } from '@/shared/audio/audio.service'

import AudioSettingsPanel from '../components/AudioSettingsPanel.vue'
import ChatPanel from '../components/ChatPanel.vue'
import GameCanvas from '../components/GameCanvas.vue'

onMounted(() => {
  playMusic('/assets/sounds/bgm.mp3', { fadeMs: 2400, volume: 0.27 })
})

onBeforeUnmount(() => {
  stopMusic()
})
</script>

<template>
  <section class="game-view">
    <GameCanvas />
    <AudioSettingsPanel />
    <aside class="game-overlay">
      <ChatPanel />
    </aside>
  </section>
</template>

<style scoped>
.game-view {
  position: relative;
  min-height: calc(100vh - 54px);
  overflow: hidden;
  background: radial-gradient(circle at top, #2f3a2f 0%, #101610 58%);
}

.game-overlay {
  position: absolute;
  top: 16px;
  right: 16px;
  bottom: 16px;
  width: min(360px, calc(100vw - 32px));
  display: flex;
  flex-direction: column;
  pointer-events: none;
}

.game-overlay > * {
  pointer-events: auto;
}

@media (max-width: 760px) {
  .game-overlay {
    left: 12px;
    right: 12px;
    top: auto;
    bottom: 12px;
    width: auto;
    max-height: 48vh;
  }
}
</style>
