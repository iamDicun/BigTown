<script setup lang="ts">
import { computed } from 'vue'
import { ref } from 'vue'

import { audioState, setMusicVolume, setSfxVolume, toggleMusicMuted } from '@/shared/audio/audio.service'
import PixelIcon from '@/shared/components/PixelIcon.vue'

const open = ref(false)
const musicVolume = computed(() => audioState.musicVolume.value)
const sfxVolume = computed(() => audioState.sfxVolume.value)
const musicVolumePercent = computed(() => Math.round(musicVolume.value * 100))
const sfxVolumePercent = computed(() => Math.round(sfxVolume.value * 100))
const musicMuted = computed(() => audioState.musicMuted.value)

function updateMusicVolume(event: Event) {
  setMusicVolume(Number((event.target as HTMLInputElement).value))
}

function updateSfxVolume(event: Event) {
  setSfxVolume(Number((event.target as HTMLInputElement).value))
}
</script>

<template>
  <div class="audio-settings">
    <button type="button" class="ui-btn ui-btn--icon settings-button" :aria-expanded="open" aria-label="Cài đặt âm thanh" @click="open = !open">
      <PixelIcon name="gear" :size="18" />
    </button>

    <section v-if="open" class="settings-panel ui-panel" aria-label="Cài đặt âm thanh">
      <span class="ui-banner">ÂM THANH</span>
      <div class="settings-body">
        <button type="button" class="ui-btn ui-btn--danger ui-btn--icon close-button" aria-label="Đóng cài đặt âm thanh" @click="open = false"><PixelIcon name="close" :size="14" /></button>

        <label>
          <span>Nhạc nền: {{ musicVolumePercent }}%</span>
          <input
            :value="musicVolume"
            min="0"
            max="1"
            step="0.01"
            type="range"
            @input="updateMusicVolume"
          />
        </label>

        <label>
          <span>Hiệu ứng: {{ sfxVolumePercent }}%</span>
          <input :value="sfxVolume" min="0" max="1" step="0.01" type="range" @input="updateSfxVolume" />
        </label>

        <button type="button" class="ui-btn ui-btn--ghost mute-button" @click="toggleMusicMuted">
          {{ musicMuted ? 'Bật nhạc' : 'Tắt nhạc' }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.audio-settings {
  position: fixed; top: 70px; left: 16px; z-index: 1010; pointer-events: auto;
  font-family: var(--pixel-font);
}
.settings-button { width: 36px; height: 36px; font-size: 18px; background: var(--pixel-parchment); color: var(--pixel-ink); text-shadow: none; }
.settings-button:hover { filter: brightness(1.06); }
.settings-button[aria-expanded="true"] { background: var(--pixel-accent); color: var(--pixel-text-inverse); text-shadow: 1px 1px 0 var(--pixel-accent-dark); }
.settings-panel {
  position: absolute; top: 44px; left: 0; width: 260px; z-index: 1009;
  padding: var(--sp-6) var(--sp-4) var(--sp-4);
}
.settings-body { display: grid; gap: var(--sp-3); }
.close-button { position: absolute; top: var(--sp-3); right: var(--sp-3); width: 32px; height: 32px; font-size: 16px; }
label {
  display: grid; gap: 6px; font-size: var(--fs-body); color: var(--pixel-ink);
}
input[type='range'] {
  width: 100%; accent-color: var(--pixel-accent);
  background: var(--pixel-parchment-dark); height: 8px;
  border: 2px solid var(--pixel-outline); outline: none; cursor: pointer;
}
.mute-button { width: 100%; }
</style>
