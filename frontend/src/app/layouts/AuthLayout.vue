<template>
  <div class="auth-layout">
    <div class="auth-layout__scrim" />
    <div class="auth-layout__content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.auth-layout {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background-image: url('../../assets/images/login-background.png');
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  image-rendering: pixelated;
  image-rendering: -moz-crisp-edges;
  image-rendering: crisp-edges;
}

.auth-layout__scrim {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at center, rgba(20, 15, 8, 0.25) 0%, rgba(20, 15, 8, 0.55) 75%),
    linear-gradient(180deg, rgba(20, 15, 8, 0.35) 0%, rgba(20, 15, 8, 0.45) 100%);
}

.auth-layout__content {
  position: relative;
  z-index: 1;
  width: 100%;
  display: flex;
  justify-content: center;
}
</style>
