<script setup lang="ts">
import { onBeforeUnmount } from 'vue'

import { stopMusic } from '@/shared/audio/audio.service'

import AudioSettingsPanel from '../components/AudioSettingsPanel.vue'
import ChatPanel from '../components/ChatPanel.vue'
import GameCanvas from '../components/GameCanvas.vue'
import EditorPanel from '../components/EditorPanel.vue'
import { useGameStore } from '../stores/game.store'

const gameStore = useGameStore()

onBeforeUnmount(() => {
  stopMusic()
})
</script>

<template>
  <section class="game-view">
    <GameCanvas />
    <div class="game-top-left">
      <AudioSettingsPanel />
    </div>
    <EditorPanel :map-code="gameStore.mapCode" />
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
  bottom: 16px;
  right: 16px;
  width: min(360px, calc(100vw - 32px));
  max-height: 250px;
  display: flex;
  flex-direction: column;
  pointer-events: none;
  z-index: 5;
}

.game-overlay > * {
  pointer-events: auto;
}

.game-top-left {
  position: absolute;
  top: 16px;
  left: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 5;
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
