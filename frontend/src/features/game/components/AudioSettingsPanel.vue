<script setup lang="ts">
import { computed } from 'vue'
import { ref } from 'vue'

import { audioState, setMusicVolume, setSfxVolume, toggleMusicMuted } from '@/shared/audio/audio.service'

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
    <button type="button" class="settings-button" :aria-expanded="open" aria-label="Cài đặt âm thanh" @click="open = !open">
      ⚙
    </button>

    <section v-if="open" class="settings-panel" aria-label="Cài đặt âm thanh">
      <header>
        <strong>Âm thanh</strong>
        <button type="button" class="close-button" aria-label="Đóng cài đặt âm thanh" @click="open = false">×</button>
      </header>

      <label>
        <span>Nhạc nền {{ musicVolumePercent }}%</span>
        <input
          :value="musicVolume"
          min="0"
          max="1"
          step="0.05"
          type="range"
          @input="updateMusicVolume"
        />
      </label>

      <label>
        <span>Hiệu ứng {{ sfxVolumePercent }}%</span>
        <input :value="sfxVolume" min="0" max="1" step="0.05" type="range" @input="updateSfxVolume" />
      </label>

      <button type="button" class="mute-button" @click="toggleMusicMuted">
        {{ musicMuted ? 'Bật nhạc' : 'Tắt nhạc' }}
      </button>
    </section>
  </div>
</template>

<style scoped>
.audio-settings {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 5;
  pointer-events: auto;
  color: #f3e7c4;
  font-family: VT323, monospace;
}

.settings-button,
.close-button,
.mute-button {
  border: 2px solid #6f5630;
  background: #21180f;
  color: #f3e7c4;
  box-shadow: 3px 3px 0 #000;
  cursor: pointer;
  font: inherit;
}

.settings-button {
  width: 42px;
  height: 42px;
  font-size: 22px;
}

.settings-panel {
  width: 240px;
  margin-top: 10px;
  padding: 14px;
  border: 2px solid #6f5630;
  background: rgba(24, 17, 10, 0.94);
  box-shadow: 4px 4px 0 #000;
}

.settings-panel header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
  font-size: 22px;
}

.close-button {
  width: 26px;
  height: 26px;
  line-height: 1;
}

label {
  display: grid;
  gap: 6px;
  margin-top: 12px;
  font-size: 18px;
}

input[type='range'] {
  width: 100%;
  accent-color: #d9a441;
}

.mute-button {
  width: 100%;
  margin-top: 14px;
  padding: 8px 10px;
  font-size: 18px;
}

@media (max-width: 760px) {
  .audio-settings {
    top: 12px;
    left: 12px;
  }
}
</style>
