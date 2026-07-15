<script setup lang="ts">
import ChatPanel from '../components/ChatPanel.vue'
import GameCanvas from '../components/GameCanvas.vue'
import LeaderboardPanel from '../components/LeaderboardPanel.vue'
</script>

<template>
  <section class="game-view">
    <GameCanvas />
    <aside class="game-overlay">
      <LeaderboardPanel />
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
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 12px;
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
