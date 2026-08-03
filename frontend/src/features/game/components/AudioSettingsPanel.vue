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
      ⚙️
    </button>

    <section v-if="open" class="settings-panel" aria-label="Cài đặt âm thanh">
      <header>
        <span class="title">ÂM THANH</span>
        <button type="button" class="close-button" aria-label="Đóng cài đặt âm thanh" @click="open = false">×</button>
      </header>

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

      <button type="button" class="mute-button" @click="toggleMusicMuted">
        {{ musicMuted ? 'Bật nhạc' : 'Tắt nhạc' }}
      </button>
    </section>
  </div>
</template>

<style scoped>
.audio-settings {
  position: fixed;
  top: 70px;
  left: 16px;
  z-index: 1010;
  pointer-events: auto;
  font-family: var(--pixel-font);
}

.settings-button {
  width: 48px;
  height: 48px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: var(--pixel-parchment);
  border: 3px solid var(--pixel-wood-dark);
  border-radius: 4px;
  cursor: pointer;
  box-shadow:
    0 4px 0 var(--pixel-wood-dark),
    inset -3px -3px 0 var(--pixel-parchment-dark);
  transition: transform 0.1s ease, box-shadow 0.1s ease;
  font-size: 24px;
  color: var(--pixel-wood-dark);
  line-height: 1;
}

.settings-button:hover {
  background: var(--pixel-parchment-dark);
}

.settings-button:active,
.settings-button[aria-expanded="true"] {
  transform: translateY(2px);
  background: var(--pixel-accent);
  box-shadow:
    0 2px 0 var(--pixel-wood-dark),
    inset -3px -3px 0 var(--pixel-accent-dark);
}

.settings-panel {
  position: absolute;
  top: 56px;
  left: 0;
  width: 260px;
  background: var(--pixel-parchment);
  padding: 16px;
  z-index: 1009;
  box-shadow:
    0 0 0 4px var(--pixel-wood-dark),
    0 0 0 8px var(--pixel-wood),
    0 0 0 11px var(--pixel-wood-dark),
    0 16px 28px rgba(0, 0, 0, 0.45);
}

.settings-panel header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 3px solid var(--pixel-wood-dark);
  padding-bottom: 8px;
  margin-bottom: 12px;
}

.settings-panel header .title {
  font-family: var(--pixel-font);
  color: var(--pixel-wood-dark);
  font-size: 22px;
  font-weight: bold;
  letter-spacing: 0.5px;
  text-shadow: 1px 1px 0 rgba(255, 255, 255, 0.5);
}

.close-button {
  background: transparent;
  border: none;
  color: var(--pixel-wood-dark);
  font-size: 24px;
  cursor: pointer;
  line-height: 1;
  padding: 0 4px;
  font-family: var(--pixel-font);
  font-weight: bold;
}

.close-button:hover {
  color: var(--pixel-danger);
}

label {
  display: grid;
  gap: 6px;
  margin-top: 12px;
  font-size: 18px;
  font-family: var(--pixel-font);
  color: var(--pixel-ink);
}

input[type='range'] {
  width: 100%;
  accent-color: var(--pixel-accent);
  background: var(--pixel-parchment-dark);
  height: 8px;
  border: 2px solid var(--pixel-wood-dark);
  outline: none;
  cursor: pointer;
}

.mute-button {
  width: 100%;
  margin-top: 14px;
  background: var(--pixel-parchment-dark);
  border: 3px solid var(--pixel-wood-dark);
  color: var(--pixel-wood-dark);
  padding: 8px 12px;
  cursor: pointer;
  font-family: var(--pixel-font);
  font-size: 18px;
  font-weight: bold;
  box-shadow:
    0 4px 0 var(--pixel-wood-dark),
    inset -2px -2px 0 rgba(0, 0, 0, 0.1);
  transition: transform 0.1s ease, box-shadow 0.1s ease;
}

.mute-button:hover {
  background: #ffe0b2;
}

.mute-button:active {
  transform: translateY(2px);
  background: var(--pixel-accent);
  box-shadow:
    0 2px 0 var(--pixel-wood-dark),
    inset -2px -2px 0 rgba(0, 0, 0, 0.2);
}
</style>
