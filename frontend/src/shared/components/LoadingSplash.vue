<script setup lang="ts">
import { computed } from 'vue'
import logo from '@/assets/images/logo.png'

const props = withDefaults(defineProps<{ progress?: number | null }>(), { progress: null })

const CIRCLE_R = 58
const CIRCUMFERENCE = 2 * Math.PI * CIRCLE_R
const VIEWBOX = (CIRCLE_R + 6) * 2

const hasProgress = computed(() => props.progress !== null && props.progress !== undefined)
const strokeDashOffset = computed(() =>
  hasProgress.value ? CIRCUMFERENCE * (1 - (props.progress! / 100)) : CIRCUMFERENCE * 0.25,
)
</script>

<template>
  <div class="splash">
    <div class="splash-ring-wrap">
      <svg class="splash-ring" :viewBox="`0 0 ${VIEWBOX} ${VIEWBOX}`">
        <circle class="splash-ring-bg" :cx="CIRCLE_R + 6" :cy="CIRCLE_R + 6" :r="CIRCLE_R" />
        <circle
          class="splash-ring-fg"
          :class="{ indeterminate: !hasProgress }"
          :cx="CIRCLE_R + 6"
          :cy="CIRCLE_R + 6"
          :r="CIRCLE_R"
          :stroke-dasharray="CIRCUMFERENCE"
          :stroke-dashoffset="strokeDashOffset"
        />
      </svg>
      <img class="splash-logo" :src="logo" alt="BigTown" />
    </div>
    <p v-if="hasProgress" class="splash-pct">{{ progress }}%</p>
    <p class="splash-label">Đang tải tài nguyên…</p>
  </div>
</template>

<style scoped>
.splash {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  background: linear-gradient(160deg, #2d5a27 0%, #3c6b34 40%, #5a9c4a 100%);
}

.splash-ring-wrap {
  position: relative;
  width: 136px;
  height: 136px;
  flex-shrink: 0;
}

.splash-ring {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.splash-ring-bg {
  fill: none;
  stroke: rgba(255, 255, 255, 0.12);
  stroke-width: 5;
}

.splash-ring-fg {
  fill: none;
  stroke: #fdf1d6;
  stroke-width: 5;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.15s linear;
}

.splash-ring-fg.indeterminate {
  animation: splash-spin 1.4s linear infinite;
  transform-origin: center;
}

@keyframes splash-spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.splash-logo {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 64px;
  height: 64px;
  image-rendering: pixelated;
}

.splash-pct {
  font-family: var(--pixel-font);
  font-size: 32px;
  color: #fdf1d6;
  text-shadow: 2px 2px 0 rgba(0, 0, 0, 0.3);
  margin: 0;
  line-height: 1;
  flex-shrink: 0;
}

.splash-label {
  font-family: var(--pixel-font);
  font-size: 22px;
  color: rgba(253, 241, 214, 0.75);
  text-shadow: 1px 1px 0 rgba(0, 0, 0, 0.25);
  margin: 0;
  white-space: nowrap;
  flex-shrink: 0;
}
</style>
