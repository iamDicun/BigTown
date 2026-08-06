<script setup lang="ts">
defineProps<{ name: string; size?: number }>()
</script>

<template>
  <svg
    class="pixel-icon"
    :class="'pixel-icon--' + name"
    :style="{ width: (size || 20) + 'px', height: (size || 20) + 'px' }"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <!-- trash -->
    <template v-if="name === 'trash'">
      <path d="M5 7h14v14a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V7Z" stroke="currentColor" stroke-width="2"/>
      <path d="M3 5h18v2H3z" stroke="currentColor" stroke-width="2"/>
      <path d="M9 3h6v2H9z" stroke="currentColor" stroke-width="2"/>
      <line x1="10" y1="10" x2="10" y2="18" stroke="currentColor" stroke-width="2"/>
      <line x1="14" y1="10" x2="14" y2="18" stroke="currentColor" stroke-width="2"/>
    </template>

    <!-- brush -->
    <template v-if="name === 'brush'">
      <path d="M4 20h4l12-12-4-4L4 16v4Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/>
      <line x1="14" y1="6" x2="18" y2="10" stroke="currentColor" stroke-width="2"/>
    </template>

    <!-- gear -->
    <template v-if="name === 'gear'">
      <circle cx="12" cy="12" r="7" stroke="currentColor" stroke-width="2"/>
      <circle cx="12" cy="12" r="3.5" stroke="currentColor" stroke-width="1.5"/>
      <rect x="10.5" y="3" width="3" height="3" rx="0.5" fill="currentColor"/>
      <rect x="10.5" y="18" width="3" height="3" rx="0.5" fill="currentColor"/>
      <rect x="3" y="10.5" width="3" height="3" rx="0.5" fill="currentColor"/>
      <rect x="18" y="10.5" width="3" height="3" rx="0.5" fill="currentColor"/>
      <rect x="5.5" y="5.5" width="3" height="3" rx="0.5" fill="currentColor" transform="rotate(45 7 7)"/>
      <rect x="15.5" y="5.5" width="3" height="3" rx="0.5" fill="currentColor" transform="rotate(45 17 7)"/>
      <rect x="5.5" y="15.5" width="3" height="3" rx="0.5" fill="currentColor" transform="rotate(-45 7 17)"/>
      <rect x="15.5" y="15.5" width="3" height="3" rx="0.5" fill="currentColor" transform="rotate(-45 17 17)"/>
    </template>

    <!-- coin -->
    <template v-if="name === 'coin'">
      <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
      <text x="12" y="17" text-anchor="middle" font-family="VT323, monospace" font-size="16" fill="currentColor">$</text>
    </template>

    <!-- warning -->
    <template v-if="name === 'warning'">
      <path d="M12 3 1 21h22L12 3Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/>
      <line x1="12" y1="9" x2="12" y2="14" stroke="currentColor" stroke-width="2" stroke-linecap="square"/>
      <circle cx="12" cy="17.5" r="1" fill="currentColor"/>
    </template>

    <!-- close -->
    <template v-if="name === 'close'">
      <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2.5" stroke-linecap="square"/>
      <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2.5" stroke-linecap="square"/>
    </template>

    <!-- chevron-down -->
    <template v-if="name === 'chevron-down'">
      <polyline points="6 9 12 15 18 9" stroke="currentColor" stroke-width="2.5" stroke-linecap="square" stroke-linejoin="round"/>
    </template>

    <!-- chevron-right -->
    <template v-if="name === 'chevron-right'">
      <polyline points="9 6 15 12 9 18" stroke="currentColor" stroke-width="2.5" stroke-linecap="square" stroke-linejoin="round"/>
    </template>

    <!-- map -->
    <template v-if="name === 'map'">
      <rect x="2" y="4" width="14" height="16" stroke="currentColor" stroke-width="2" rx="1"/>
      <path d="M16 8h5v14a1 1 0 0 1-1 1h-4" stroke="currentColor" stroke-width="2"/>
      <polyline points="9 4 9 20 16 20 19 22 20 8 16 8 16 4" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/>
    </template>

    <!-- graveyard -->
    <template v-if="name === 'graveyard'">
      <rect x="8" y="6" width="8" height="10" stroke="currentColor" stroke-width="2" rx="1"/>
      <path d="M6 6h12a2 2 0 0 1 0 4H6a2 2 0 0 1 0-4Z" stroke="currentColor" stroke-width="2"/>
      <line x1="12" y1="16" x2="12" y2="20" stroke="currentColor" stroke-width="2"/>
      <line x1="8" y1="20" x2="16" y2="20" stroke="currentColor" stroke-width="2"/>
    </template>

    <!-- snow -->
    <template v-if="name === 'snow'">
      <line x1="12" y1="2" x2="12" y2="22" stroke="currentColor" stroke-width="2"/>
      <line x1="2" y1="12" x2="22" y2="12" stroke="currentColor" stroke-width="2"/>
      <line x1="4.5" y1="4.5" x2="19.5" y2="19.5" stroke="currentColor" stroke-width="2"/>
      <line x1="19.5" y1="4.5" x2="4.5" y2="19.5" stroke="currentColor" stroke-width="2"/>
    </template>

    <!-- home -->
    <template v-if="name === 'home'">
      <path d="M3 10 12 3l9 7v9a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-9Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/>
      <rect x="9" y="13" width="6" height="7" stroke="currentColor" stroke-width="2"/>
    </template>
  </svg>
</template>

<style scoped>
.pixel-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
  color: var(--pixel-ink);
}
</style>
