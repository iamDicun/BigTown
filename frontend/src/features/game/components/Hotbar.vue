<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useHotbarStore } from '../stores/hotbar.store'
import { useGameStore } from '../stores/game.store'
import { getItemPreviewStyle } from '../utils/itemPreview'

const hotbar = useHotbarStore()
const gameStore = useGameStore()

const isDeleteMode = ref(false)
const isSticky = ref(true)

const slots = computed(() => hotbar.slots.map((id, i) => ({
  index: i,
  item: id ? hotbar.itemById[id] ?? null : null,
})))

function toggleDeleteMode() {
  isDeleteMode.value = !isDeleteMode.value
  if (isDeleteMode.value) {
    isSticky.value = false
    window.dispatchEvent(new CustomEvent('game:setStickyBrush', { detail: { on: false } }))
    window.dispatchEvent(new CustomEvent('game:cancelPlacement'))
  }
  window.dispatchEvent(new CustomEvent('game:toggleDeleteMode', {
    detail: { active: isDeleteMode.value }
  }))
}

function toggleSticky() {
  isSticky.value = !isSticky.value
  if (isSticky.value) {
    isDeleteMode.value = false
    window.dispatchEvent(new CustomEvent('game:toggleDeleteMode', {
      detail: { active: false }
    }))
  }
  window.dispatchEvent(new CustomEvent('game:setStickyBrush', { detail: { on: isSticky.value } }))
}

function onKeydown(e: KeyboardEvent) {
  if ((e.target as HTMLElement)?.tagName === 'INPUT') return
  const n = parseInt(e.key, 10)
  if (n >= 1 && n <= 5) {
    hotbar.setActive(n - 1)
    e.preventDefault()
    return
  }
  if (e.key === 'Delete') {
    toggleDeleteMode()
    e.preventDefault()
  }
  if (e.key === 't' || e.key === 'T') {
    toggleSticky()
    e.preventDefault()
  }
}

function onWheel(e: WheelEvent) {
  hotbar.cycle(e.deltaY > 0 ? 1 : -1)
  e.preventDefault()
}

onMounted(() => {
  hotbar.load()
  window.addEventListener('keydown', onKeydown)
  const canvas = document.getElementById('game-root')
  canvas?.addEventListener('wheel', onWheel, { passive: false })
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  const canvas = document.getElementById('game-root')
  canvas?.removeEventListener('wheel', onWheel)
})
</script>

<template>
  <div class="hotbar">
    <button
      class="hotbar-btn"
      :class="{ active: isDeleteMode }"
      @click="toggleDeleteMode()"
      title="Xoa vat pham (Del)"
    >🗑️</button>
    <button
      class="hotbar-btn"
      :class="{ active: isSticky }"
      @click="toggleSticky()"
      title="Dat lien tiep (T)"
    >🖌️</button>

    <div class="hotbar-divider"></div>

    <div
      v-for="slot in slots"
      :key="slot.index"
      class="hotbar-slot"
      :class="{ active: hotbar.activeIndex === slot.index, disabled: slot.item && gameStore.coins < slot.item.price }"
      @click="hotbar.setActive(slot.index)"
      @contextmenu.prevent="hotbar.clearSlot(slot.index)"
      :title="slot.item ? slot.item.name : 'O trong — mo kho (E) de gan'"
    >
      <span class="slot-key">{{ slot.index + 1 }}</span>
      <div v-if="slot.item" class="slot-icon" :style="getItemPreviewStyle(slot.item, 40)"></div>
      <span v-if="slot.item" class="slot-price">{{ slot.item.price }}</span>
    </div>
  </div>
</template>

<style scoped>
.hotbar {
  position: fixed; bottom: 12px; left: 50%; transform: translateX(-50%);
  display: flex; align-items: center; gap: 4px; z-index: 1005; pointer-events: auto;
  padding: 6px 8px;
  background: linear-gradient(180deg, #c98a4b 0%, #b07838 100%);
  border: 4px solid var(--pixel-wood-dark); border-radius: 8px;
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.12),
    0 0 0 2px #4a2a10,
    0 5px 0 #4a2a10,
    0 7px 10px rgba(0,0,0,0.35);
}
.hotbar-btn {
  width: 40px; height: 40px; cursor: pointer;
  font-size: 18px; display: flex; align-items: center; justify-content: center;
  background: linear-gradient(180deg, #8a5a34 0%, #6b4226 100%);
  border: 2px solid #4a2a10; border-radius: 5px;
  box-shadow:
    inset 0 2px 3px rgba(0,0,0,0.3),
    0 3px 0 #4a2a10;
  transition: transform 0.1s;
  line-height: 1;
}
.hotbar-btn:active { transform: translateY(2px); box-shadow: inset 0 2px 3px rgba(0,0,0,0.3), 0 1px 0 #4a2a10; }
.hotbar-btn.active {
  background: linear-gradient(180deg, #c94a3c 0%, #a03028 100%);
  border-color: #6b1a10;
  box-shadow:
    inset 0 2px 3px rgba(0,0,0,0.3),
    0 3px 0 #6b1a10;
}
.hotbar-divider {
  width: 3px; height: 40px; background: #4a2a10; border-radius: 2px;
  margin: 0 4px;
}
.hotbar-slot {
  position: relative; width: 52px; height: 52px; cursor: pointer;
  background: linear-gradient(180deg, #8a5a34 0%, #6b4226 100%);
  border: 3px solid #4a2a10; border-radius: 5px;
  display: flex; align-items: center; justify-content: center;
  box-shadow:
    inset 0 3px 5px rgba(0,0,0,0.35),
    0 4px 0 #4a2a10;
  transition: transform 0.1s;
}
.hotbar-slot:active { transform: translateY(2px); box-shadow: inset 0 3px 5px rgba(0,0,0,0.35), 0 2px 0 #4a2a10; }
.hotbar-slot.active {
  border-color: var(--pixel-accent);
  box-shadow:
    inset 0 3px 5px rgba(0,0,0,0.35),
    0 0 0 2px var(--pixel-accent),
    0 4px 0 #4a2a10;
}
.hotbar-slot.disabled { opacity: 0.4; }
.slot-key {
  position: absolute; top: 1px; left: 3px; font-family: var(--pixel-font);
  font-size: 12px; color: #fdf1d6; text-shadow: 1px 1px 0 rgba(0,0,0,0.4);
}
.slot-icon { width: 40px; height: 40px; image-rendering: pixelated; overflow: hidden; }
.slot-price {
  position: absolute; bottom: 0; right: 2px; font-family: var(--pixel-font);
  font-size: 11px; color: #ffd700; text-shadow: 1px 1px 0 #3a2b1a;
}
</style>
