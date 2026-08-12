<script setup lang="ts">
import { ref, computed } from 'vue'
import type { DecorationItemDto } from '../services/editor.service'
import { useHotbarStore } from '../stores/hotbar.store'
import { useGameStore } from '../stores/game.store'
import { categoryOf, CATEGORY_LABELS, type Category } from '../utils/itemCategory'
import { getItemPreviewStyle } from '../utils/itemPreview'
import PixelIcon from '@/shared/components/PixelIcon.vue'

const props = defineProps<{ items: DecorationItemDto[] }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const hotbar = useHotbarStore()
const gameStore = useGameStore()

const search = ref('')
const activeTab = ref<Category | 'all'>('all')
const pickedItemId = ref<string | null>(null)

const tabs = computed<(Category | 'all')[]>(() => {
  const set = new Set<Category>()
  for (const it of props.items) set.add(categoryOf(it.code))
  return ['all', ...Array.from(set)]
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return props.items.filter(it => {
    const okTab = activeTab.value === 'all' || categoryOf(it.code) === activeTab.value
    const okSearch = !q || it.name.toLowerCase().includes(q) || it.code.toLowerCase().includes(q)
    return okTab && okSearch
  })
})

function labelOf(tab: Category | 'all') {
  return tab === 'all' ? 'Tất cả' : CATEGORY_LABELS[tab]
}

function pickItem(it: DecorationItemDto) {
  pickedItemId.value = pickedItemId.value === it.id ? null : it.id
}

function assignToSlot(index: number) {
  if (!pickedItemId.value) return
  hotbar.assign(index, pickedItemId.value)
  pickedItemId.value = null
}

function quickAssign(it: DecorationItemDto) {
  hotbar.assign(hotbar.activeIndex, it.id)
}
function onInputKeydown(e: KeyboardEvent) {
  e.stopPropagation()
}

function onInputFocus() {
  window.dispatchEvent(new CustomEvent('game:chatFocus', { detail: { focused: true } }))
}

function onInputBlur() {
  window.dispatchEvent(new CustomEvent('game:chatFocus', { detail: { focused: false } }))
}
</script>

<template>
  <div class="ui-overlay" @click.self="emit('close')">
    <div class="inv-panel ui-panel ui-pop">
      <span class="ui-banner">KHO ĐỒ VẬT</span>

      <div class="inv-header">
        <div class="inv-search-wrap">
          <input
            ref="searchInput"
            v-model="search"
            class="ui-input inv-search"
            placeholder="Tìm tên hoặc mã vật phẩm..."
            @focus="onInputFocus"
            @blur="onInputBlur"
            @keydown="onInputKeydown"
          />
        </div>
        <button class="ui-btn ui-btn--danger ui-btn--icon" @click="emit('close')"><PixelIcon name="close" :size="16" /></button>
      </div>

      <div class="inv-tabs">
        <button
          v-for="tab in tabs" :key="tab"
          class="ui-tab" :class="{ 'is-active': activeTab === tab }"
          @click="activeTab = tab"
        >{{ labelOf(tab) }}</button>
      </div>

      <div class="inv-grid ui-scroll">
        <div
          v-for="it in filtered" :key="it.id"
          class="inv-item"
          :class="{ picked: pickedItemId === it.id, disabled: gameStore.coins < it.price }"
          @click="pickItem(it)"
          @dblclick="quickAssign(it)"
        >
          <span class="inv-item-price">{{ it.price }}</span>
          <div class="inv-icon-wrap">
            <div class="inv-icon" :style="getItemPreviewStyle(it, 100)"></div>
          </div>
          <span class="inv-name">{{ it.name }}</span>
        </div>
        <p v-if="filtered.length === 0" class="inv-empty">Không tìm thấy vật phẩm nào.</p>
      </div>

      <div class="inv-hotbar-row">
        <span class="inv-hint">
          {{ pickedItemId ? 'Chọn 1 ô bên dưới để gán vật phẩm ↓' : 'Chọn 1 vật phẩm bên trên rồi gán vào hotbar' }}
        </span>
        <div class="inv-hotbar-slots">
          <div
            v-for="(id, i) in hotbar.slots" :key="i"
            class="ui-slot"
            :class="{ 'is-active': hotbar.activeIndex === i, 'is-fillable': !!pickedItemId }"
            @click="assignToSlot(i)"
            @contextmenu.prevent="hotbar.clearSlot(i)"
          >
            <span class="ui-slot-key">{{ i + 1 }}</span>
            <div v-if="id && hotbar.itemById[id]" class="ui-slot-icon"
                 :style="getItemPreviewStyle(hotbar.itemById[id], 40)"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.inv-panel {
  width: min(620px, 94vw); max-height: 86vh; display: flex; flex-direction: column;
  padding: var(--sp-5) var(--sp-4) var(--sp-4);
  overflow: visible;
}
.inv-header { display: flex; align-items: center; gap: var(--sp-3); margin-bottom: var(--sp-3); margin-top: var(--sp-2); }
.inv-search-wrap {
  flex: 1; background: var(--pixel-wood-dark); border-radius: var(--r-lg);
  padding: 2px; box-shadow: inset 0 3px 5px rgba(0,0,0,0.4);
}
.inv-search { width: 100%; }
.inv-tabs { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: var(--sp-3); }
.inv-grid {
  flex: 1; overflow-y: auto; display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--sp-3);
  padding: var(--sp-3); background: var(--pixel-wood-dark); border-radius: var(--r-lg);
  box-shadow: inset 0 4px 10px rgba(0,0,0,0.5);
}
.inv-item {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: var(--sp-3) var(--sp-2) var(--sp-2); cursor: pointer; position: relative;
  background: linear-gradient(180deg, var(--pixel-wood) 0%, var(--pixel-wood-dark) 100%);
  border: var(--bw) solid var(--pixel-outline); border-radius: var(--r);
  box-shadow: var(--bevel-in), var(--lift);
  transition: transform .05s ease, box-shadow .05s ease;
}
.inv-item:active:not(.disabled) { transform: translateY(3px); box-shadow: var(--bevel-in), 0 1px 0 var(--pixel-outline); }
.inv-item.picked {
  border-color: var(--pixel-accent);
  box-shadow: var(--bevel-in), 0 0 0 3px var(--pixel-accent), var(--lift);
}
.inv-item.disabled { opacity: 0.4; cursor: not-allowed; }
.inv-item-price {
  position: absolute; top: 4px; right: 6px;
  font-family: var(--pixel-font); font-size: var(--fs-cap);
  color: #ffd700; text-shadow: 1px 1px 0 #3a2b1a;
  z-index: 2;
}
.inv-icon-wrap {
  width: 100px; height: 100px;
  display: flex; align-items: center; justify-content: center;
  overflow: hidden;
}
.inv-icon {
  image-rendering: pixelated;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.inv-name {
  font-family: var(--pixel-font); font-size: 13px; color: var(--pixel-text-inverse);
  text-align: center; width: 100%; line-height: 1.2;
  text-shadow: 1px 1px 0 rgba(0,0,0,0.4);
  word-break: break-word;
}
.inv-empty {
  grid-column: 1 / -1; text-align: center;
  font-family: var(--pixel-font); font-size: var(--fs-body); color: var(--pixel-wood-light); padding: var(--sp-6);
}
.inv-hotbar-row {
  margin-top: var(--sp-3); border-top: 2px dashed var(--pixel-wood-dark); padding-top: var(--sp-3);
}
.inv-hint { font-family: var(--pixel-font); font-size: var(--fs-label); color: var(--pixel-ink); }
.inv-hotbar-slots { display: flex; gap: 4px; justify-content: center; margin-top: var(--sp-2); }
</style>
