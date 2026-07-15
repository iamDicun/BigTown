<template>
  <div class="game-canvas-shell">
    <div class="game-canvas-placeholder">
      <p class="eyebrow">BigTown MVP</p>
      <h1>Phaser canvas mount point</h1>
      <p>
        Khu vực này sẽ được thay bằng canvas Phaser để render map, player, NPC, animation,
        HP bar và chat bubble.
      </p>
    </div>
  </div>
</template>

<style scoped>
.game-canvas-shell {
  min-height: calc(100vh - 54px);
  display: grid;
  place-items: center;
  padding: 24px;
}

.game-canvas-placeholder {
  width: min(960px, 100%);
  min-height: 540px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  padding: 32px;
  border: 2px solid #566744;
  border-radius: 18px;
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px),
    linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
    #1d2a1d;
  background-size: 32px 32px;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.35);
  text-align: center;
}

.eyebrow {
  color: #9fd47f;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

p {
  max-width: 560px;
  color: #c9c1aa;
}
</style>
