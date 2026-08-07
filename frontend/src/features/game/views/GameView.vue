<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { stopMusic } from '@/shared/audio/audio.service'
import AudioSettingsPanel from '../components/AudioSettingsPanel.vue'
import ChatPanel from '../components/ChatPanel.vue'
import GameCanvas from '../components/GameCanvas.vue'
import EditorPanel from '../components/EditorPanel.vue'
import Hotbar from '../components/Hotbar.vue'
import InventoryModal from '../components/InventoryModal.vue'
import HelpHTMLOverlay from '../components/HelpHTMLOverlay.vue'
import { useGameStore } from '../stores/game.store'
import { useHotbarStore } from '../stores/hotbar.store'
import type { DecorationItemDto } from '../services/editor.service'

const gameStore = useGameStore()
const hotbarStore = useHotbarStore()
const invOpen = ref(false)

const catalogItems = computed<DecorationItemDto[]>(() => {
  return Object.values(hotbarStore.itemById)
})

function onKey(e: KeyboardEvent) {
  if ((e.target as HTMLElement)?.tagName === 'INPUT') return
  if (e.key === 'e' || e.key === 'E') {
    invOpen.value = !invOpen.value
    e.preventDefault()
  }
  if (e.key === 'Escape' && invOpen.value) {
    invOpen.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  stopMusic()
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <section class="game-view">
    <GameCanvas />
    <AudioSettingsPanel />
    <HelpHTMLOverlay />
    <EditorPanel :map-code="gameStore.mapCode" />
    <Hotbar />
    <InventoryModal v-if="invOpen" :items="catalogItems" @close="invOpen = false" />
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
