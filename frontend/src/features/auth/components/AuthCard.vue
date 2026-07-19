<script setup lang="ts">
import PixelPanel from '@/shared/components/PixelPanel.vue'
import logo from '@/assets/images/logo.png'
</script>

<template>
  <div class="auth-card">
    <img class="auth-card__logo" :src="logo" alt="BigTown" />
    <PixelPanel>
      <h1 class="auth-card__title"><slot name="title" /></h1>
      <slot />
    </PixelPanel>
  </div>
</template>

<style scoped>
.auth-card {
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.auth-card__logo {
  width: min(280px, 80%);
  image-rendering: -webkit-optimize-contrast;
  filter: drop-shadow(0 6px 10px rgba(0, 0, 0, 0.5));
  margin-bottom: -14px;
  position: relative;
  z-index: 1;
}

.auth-card__title {
  margin: 0 0 18px;
  text-align: center;
  font-family: var(--pixel-font);
  font-size: 30px;
  letter-spacing: 1px;
  color: var(--pixel-wood-dark);
  text-shadow: 1px 1px 0 rgba(255, 255, 255, 0.5);
}
</style>
