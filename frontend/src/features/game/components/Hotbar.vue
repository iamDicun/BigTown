<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useHotbarStore } from '../stores/hotbar.store'
import { useGameStore } from '../stores/game.store'
import { getItemPreviewStyle } from '../utils/itemPreview'
import PixelIcon from '@/shared/components/PixelIcon.vue'

const hotbar = useHotbarStore()
const gameStore = useGameStore()

const isDeleteMode = ref(false)

const slots = computed(() => hotbar.slots.map((id, i) => ({
  index: i,
  item: id ? hotbar.itemById[id] ?? null : null,
})))

function toggleDeleteMode() {
  isDeleteMode.value = !isDeleteMode.value
  if (isDeleteMode.value) {
    window.dispatchEvent(new CustomEvent('game:cancelPlacement'))
  }
  window.dispatchEvent(new CustomEvent('game:toggleDeleteMode', {
    detail: { active: isDeleteMode.value }
  }))
}

function onKeydown(e: KeyboardEvent) {
  if ((e.target as HTMLElement)?.tagName === 'INPUT') return
  const n = parseInt(e.key, 10)
  if (n >= 1 && n <= 5) {
    hotbar.setActive(n - 1)
    e.preventDefault()
    return
  }
  if (e.key === 'q' || e.key === 'Q') {
    toggleDeleteMode()
    e.preventDefault()
  }
}

onMounted(() => {
  hotbar.load()
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="hotbar ui-panel">
    <button
      class="ui-btn ui-btn--icon hotbar-mode-btn"
      :class="{ 'ui-btn--danger': isDeleteMode }"
      @click="toggleDeleteMode()"
      title="Xóa vật phẩm (Q)"
    ><PixelIcon name="trash" :size="18" /></button>

    <div class="hotbar-divider"></div>

    <div
      v-for="slot in slots"
      :key="slot.index"
      class="ui-slot"
      :class="{ 'is-active': hotbar.activeIndex === slot.index, 'is-disabled': slot.item && gameStore.coins < slot.item.price }"
      @click="hotbar.setActive(slot.index)"
      @contextmenu.prevent="hotbar.clearSlot(slot.index)"
      :title="slot.item ? slot.item.name : 'Ô trống — mở kho (E) để gán'"
    >
      <span class="ui-slot-key">{{ slot.index + 1 }}</span>
      <div v-if="slot.item" class="ui-slot-icon" :style="getItemPreviewStyle(slot.item, 40)"></div>
      <span v-if="slot.item" class="ui-slot-price">{{ slot.item.price }}</span>
    </div>
  </div>
</template>

<style scoped>
.hotbar {
  position: fixed; bottom: 12px; left: 50%; transform: translateX(-50%);
  display: flex; align-items: center; gap: var(--sp-1); z-index: 1005; pointer-events: auto;
  padding: var(--sp-2) var(--sp-2);
  background: linear-gradient(180deg, var(--pixel-wood-light) 0%, #b07838 100%);
}
.hotbar-mode-btn { width: 40px; height: 40px; background: var(--pixel-wood); text-shadow: none; }
.hotbar-mode-btn.ui-btn--danger { background: var(--pixel-danger); text-shadow: 1px 1px 0 var(--pixel-danger-dark); }
.hotbar-divider {
  width: 3px; height: 40px; background: var(--pixel-outline); border-radius: 2px;
  margin: 0 var(--sp-1);
}
</style>
