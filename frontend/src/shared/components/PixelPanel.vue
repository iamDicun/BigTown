<template>
  <div class="pixel-panel">
    <slot />
  </div>
</template>

<style scoped>
.pixel-panel {
  width: 100%;
  background: var(--pixel-parchment);
  padding: 28px 26px 26px;
  position: relative;
  box-shadow:
    0 0 0 4px var(--pixel-wood-dark),
    0 0 0 8px var(--pixel-wood),
    0 0 0 11px var(--pixel-wood-dark),
    0 16px 28px rgba(0, 0, 0, 0.45);
}
</style>
