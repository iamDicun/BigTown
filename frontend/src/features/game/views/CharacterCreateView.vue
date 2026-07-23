<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { ApiError } from '@/shared/api/http'

import * as characterService from '../services/character.service'
import type { CharacterOptionDto } from '../services/character.service'
import { useGameStore } from '../stores/game.store'

const router = useRouter()
const gameStore = useGameStore()

const options = ref<CharacterOptionDto[]>([])
const selectedIndex = ref(0)
const name = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref('')

const selectedOption = computed(() => options.value[selectedIndex.value] ?? null)

const previewStyle = computed(() => {
  const option = selectedOption.value
  if (!option) return {}
  const cfg = option.spritesheet
  const targetSize = 96
  const scale = targetSize / cfg.frame_height
  return {
    width: targetSize + 'px',
    height: targetSize + 'px',
    backgroundImage: `url(${option.preview_url})`,
    backgroundSize: cfg.frame_width * cfg.columns * scale + 'px auto',
    backgroundPosition: '0 0',
  }
})

const canGoPrev = computed(() => selectedIndex.value > 0)
const canGoNext = computed(() => selectedIndex.value < options.value.length - 1)

function goPrev() {
  if (canGoPrev.value) selectedIndex.value--
}

function goNext() {
  if (canGoNext.value) selectedIndex.value++
}

onMounted(async () => {
  try {
    const existing = await characterService.getMe()
    gameStore.setMyCharacter(existing)
    await router.replace({ name: 'game' })
    return
  } catch (err) {
    if (!(err instanceof ApiError) || err.status !== 404) {
      error.value = err instanceof Error ? err.message : 'Không kiểm tra được nhân vật hiện tại'
      loading.value = false
      return
    }
  }

  try {
    options.value = await characterService.getOptions()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Không tải được danh sách nhân vật'
  } finally {
    loading.value = false
  }
})

async function createCharacter() {
  const option = selectedOption.value
  if (!option) {
    error.value = 'Không có nhân vật để chọn'
    return
  }

  const characterName = name.value.trim()
  if (!characterName) {
    error.value = 'Vui lòng nhập tên nhân vật'
    return
  }

  saving.value = true
  error.value = ''
  try {
    const character = await characterService.createCharacter({
      name: characterName,
      base_asset_key: option.base_asset_key,
    })
    gameStore.setMyCharacter(character)
    gameStore.setSpritesheetConfig(option.spritesheet)
    await router.replace({ name: 'game' })
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Không tạo được nhân vật'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="character-create-view">
    <div class="creator-card">
      <p class="eyebrow">BigTown</p>
      <h1>Tạo nhân vật</h1>
      <p class="intro">Chọn một nhân vật, đặt tên rồi vào thị trấn.</p>

      <p v-if="error" class="pixel-alert pixel-alert--error">{{ error }}</p>
      <p v-if="loading" class="loading">Đang tải danh sách nhân vật...</p>

  <template v-else-if="options.length > 0">
    <div class="carousel">
      <button
        type="button"
        class="arrow arrow-left"
        :disabled="!canGoPrev"
        aria-label="Nhân vật trước"
        @click="goPrev"
      >
        ◀
      </button>

      <div class="preview-area">
        <span
          class="sprite-viewer"
          :style="previewStyle"
        />
        <strong class="selected-name">{{ selectedOption?.name }}</strong>
        <span class="selected-counter">{{ selectedIndex + 1 }} / {{ options.length }}</span>
      </div>

      <button
        type="button"
        class="arrow arrow-right"
        :disabled="!canGoNext"
        aria-label="Nhân vật tiếp theo"
        @click="goNext"
      >
        ▶
      </button>
    </div>

        <form class="creator-form" @submit.prevent="createCharacter">
          <label class="name-field">
            <span>Tên nhân vật</span>
            <input v-model="name" maxlength="80" placeholder="Ví dụ: BigCat" type="text" />
          </label>

          <button type="submit" class="create-button" :disabled="saving">
            {{ saving ? 'Đang tạo...' : 'Tạo nhân vật' }}
          </button>
        </form>
      </template>

      <p v-else class="loading">Không có nhân vật nào để chọn.</p>
    </div>
  </section>
</template>

<style scoped>
.character-create-view {
  min-height: calc(100vh - 54px);
  display: grid;
  place-items: center;
  padding: 24px;
  background:
    radial-gradient(circle at 20% 10%, rgba(214, 158, 63, 0.22), transparent 34%),
    linear-gradient(180deg, #172315 0%, #0d130d 100%);
  color: #f5e9ca;
  font-family: VT323, monospace;
}

.creator-card {
  width: min(560px, 100%);
  padding: 28px;
  border: 3px solid #7d5b2b;
  background: rgba(29, 21, 13, 0.94);
  box-shadow: 6px 6px 0 #000;
}

.eyebrow,
h1,
.intro {
  margin: 0;
  text-align: center;
}

.eyebrow {
  color: #d9a441;
  font-size: 20px;
  letter-spacing: 2px;
}

h1 {
  margin-top: 4px;
  font-size: 42px;
}

.intro {
  margin-top: 8px;
  margin-bottom: 20px;
  color: #d8c69b;
  font-size: 20px;
}

.carousel {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 16px;
}

.arrow {
  width: 40px;
  height: 40px;
  border: 2px solid #7d5b2b;
  background: #21180f;
  color: #f3e7c4;
  box-shadow: 3px 3px 0 #000;
  cursor: pointer;
  font-size: 18px;
  display: grid;
  place-items: center;
}

.arrow:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.preview-area {
  display: grid;
  place-items: center;
  gap: 10px;
  min-width: 180px;
}

.sprite-viewer {
  background-repeat: no-repeat;
  background-position: 0 0;
  image-rendering: pixelated;
  border: 2px solid #514024;
}

.selected-name {
  font-size: 24px;
  color: #d9a441;
}

.selected-counter {
  font-size: 18px;
  color: #9e8b6c;
}

.creator-form {
  display: grid;
  gap: 18px;
}

.name-field {
  display: grid;
  gap: 8px;
  font-size: 20px;
}

input[type='text'] {
  width: 100%;
  box-sizing: border-box;
  padding: 12px 14px;
  border: 2px solid #7d5b2b;
  background: #110d09;
  color: #f5e9ca;
  font: inherit;
  font-size: 22px;
  outline: none;
}

.create-button {
  justify-self: center;
  min-width: 200px;
  padding: 12px 18px;
  border: 2px solid #6f5630;
  background: #d9a441;
  color: #21180f;
  box-shadow: 4px 4px 0 #000;
  cursor: pointer;
  font: inherit;
  font-size: 24px;
}

.create-button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.loading {
  margin: 22px 0 0;
  text-align: center;
  color: #d8c69b;
  font-size: 20px;
}

@media (max-width: 640px) {
  .creator-card {
    padding: 20px;
  }
}
</style>
