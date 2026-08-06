<script setup lang="ts">
import { ref, computed } from 'vue'
import type { DecorationItemDto } from '../services/editor.service'
import { useHotbarStore } from '../stores/hotbar.store'
import { useGameStore } from '../stores/game.store'
import { categoryOf, CATEGORY_LABELS, type Category } from '../utils/itemCategory'
import { getItemPreviewStyle } from '../utils/itemPreview'

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
  return tab === 'all' ? 'Tat ca' : CATEGORY_LABELS[tab]
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
</script>

<template>
  <div class="inv-overlay" @click.self="emit('close')">
    <div class="inv-panel">
      <div class="inv-header">
        <span class="inv-title">KHO DO VAT</span>
        <div class="inv-search-wrap">
          <input v-model="search" class="inv-search" placeholder="Tim ten hoac ma vat pham..." />
        </div>
        <button class="inv-close" @click="emit('close')">X</button>
      </div>

      <div class="inv-tabs">
        <button
          v-for="tab in tabs" :key="tab"
          class="inv-tab" :class="{ active: activeTab === tab }"
          @click="activeTab = tab"
        >{{ labelOf(tab) }}</button>
      </div>

      <div class="inv-grid">
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
        <p v-if="filtered.length === 0" class="inv-empty">Khong tim thay vat pham nao.</p>
      </div>

      <div class="inv-hotbar-row">
        <span class="inv-hint">
          {{ pickedItemId ? 'Chon 1 o ben duoi de gan vat pham ↓' : 'Chon 1 vat pham ben tren roi gan vao hotbar' }}
        </span>
        <div class="inv-hotbar">
          <div
            v-for="(id, i) in hotbar.slots" :key="i"
            class="inv-slot" :class="{ active: hotbar.activeIndex === i, fillable: !!pickedItemId }"
            @click="assignToSlot(i)"
            @contextmenu.prevent="hotbar.clearSlot(i)"
          >
            <span class="inv-slot-key">{{ i + 1 }}</span>
            <div v-if="id && hotbar.itemById[id]" class="inv-slot-icon"
                 :style="getItemPreviewStyle(hotbar.itemById[id], 40)"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.inv-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.55);
  display: flex; align-items: center; justify-content: center; z-index: 1100;
}
.inv-panel {
  width: min(620px, 94vw); max-height: 86vh; display: flex; flex-direction: column;
  background: linear-gradient(180deg, #c98a4b 0%, #b07838 100%);
  padding: 14px;
  border: 5px solid var(--pixel-wood-dark);
  border-radius: 10px;
  box-shadow:
    inset 0 2px 0 rgba(255,255,255,0.15),
    0 0 0 3px #4a2a10,
    0 12px 32px rgba(0,0,0,0.55);
}
.inv-header { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; }
.inv-title {
  font-family: var(--pixel-font); font-size: 26px; color: #3a2b1a;
  text-shadow: 1px 1px 0 rgba(255,255,255,0.2);
  white-space: nowrap;
}
.inv-search-wrap {
  flex: 1; background: #5c3820; border-radius: 5px; padding: 2px;
  box-shadow: inset 0 3px 5px rgba(0,0,0,0.4);
}
.inv-search {
  width: 100%; font-family: var(--pixel-font); font-size: 16px;
  padding: 6px 10px; border: none; border-radius: 4px;
  background: #fdf1d6; color: #3a2b1a;
}
.inv-search::placeholder { color: #a08060; }
.inv-close {
  border: 3px solid #4a2a10; background: #c94a3c; color: #fff;
  width: 34px; height: 34px; cursor: pointer; border-radius: 5px;
  font-family: var(--pixel-font); font-size: 18px; line-height: 1;
  box-shadow: 0 3px 0 #4a2a10;
}
.inv-close:active { transform: translateY(2px); box-shadow: 0 1px 0 #4a2a10; }
.inv-tabs { display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 10px; }
.inv-tab {
  font-family: var(--pixel-font); font-size: 15px; padding: 5px 12px; cursor: pointer;
  border: 2px solid var(--pixel-wood-dark); border-radius: 5px;
  background: linear-gradient(180deg, #fdf1d6 0%, #e8d5a8 100%);
  color: #3a2b1a; box-shadow: 0 3px 0 var(--pixel-wood-dark);
  transition: transform 0.1s;
}
.inv-tab:active { transform: translateY(2px); box-shadow: 0 1px 0 var(--pixel-wood-dark); }
.inv-tab.active {
  background: linear-gradient(180deg, var(--pixel-accent) 0%, var(--pixel-accent-dark) 100%);
  color: #fff; border-color: #8a3a10;
  box-shadow: 0 3px 0 #8a3a10;
}
.inv-grid {
  flex: 1; overflow-y: auto; display: grid;
  grid-template-columns: repeat(4, 1fr); gap: 10px;
  padding: 10px; background: #5c3820; border-radius: 6px;
  box-shadow: inset 0 4px 10px rgba(0,0,0,0.5);
}
.inv-item {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 10px 8px 8px; cursor: pointer; position: relative;
  background: linear-gradient(180deg, #8a5a34 0%, #6b4226 100%);
  border: 3px solid #4a2a10; border-radius: 6px;
  box-shadow:
    inset 0 3px 6px rgba(0,0,0,0.35),
    0 4px 0 #4a2a10,
    0 5px 3px rgba(0,0,0,0.2);
  transition: transform 0.1s, box-shadow 0.1s;
}
.inv-item:active:not(.disabled) {
  transform: translateY(2px);
  box-shadow:
    inset 0 3px 6px rgba(0,0,0,0.35),
    0 2px 0 #4a2a10,
    0 3px 3px rgba(0,0,0,0.2);
}
.inv-item.picked {
  border-color: var(--pixel-accent);
  box-shadow:
    inset 0 3px 6px rgba(0,0,0,0.35),
    0 0 0 3px var(--pixel-accent),
    0 4px 0 #4a2a10,
    0 5px 3px rgba(0,0,0,0.2);
}
.inv-item.disabled { opacity: 0.4; cursor: not-allowed; }
.inv-item-price {
  position: absolute; top: 4px; right: 6px;
  font-family: var(--pixel-font); font-size: 14px;
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
  font-family: var(--pixel-font); font-size: 13px; color: #fdf1d6;
  text-align: center; width: 100%; line-height: 1.2;
  text-shadow: 1px 1px 0 rgba(0,0,0,0.4);
  word-break: break-word;
}
.inv-empty {
  grid-column: 1 / -1; text-align: center;
  font-family: var(--pixel-font); font-size: 18px; color: #c98a4b; padding: 30px;
}
.inv-hotbar-row {
  margin-top: 12px; border-top: 2px dashed var(--pixel-wood-dark); padding-top: 10px;
}
.inv-hint { font-family: var(--pixel-font); font-size: 15px; color: #3a2b1a; }
.inv-hotbar { display: flex; gap: 5px; justify-content: center; margin-top: 8px; }
.inv-slot {
  position: relative; width: 48px; height: 48px; cursor: pointer;
  background: linear-gradient(180deg, #8a5a34 0%, #6b4226 100%);
  border: 3px solid #4a2a10; border-radius: 5px;
  display: flex; align-items: center; justify-content: center;
  box-shadow:
    inset 0 2px 5px rgba(0,0,0,0.4),
    0 3px 0 #4a2a10;
  transition: transform 0.1s;
}
.inv-slot:active { transform: translateY(2px); box-shadow: inset 0 2px 5px rgba(0,0,0,0.4), 0 1px 0 #4a2a10; }
.inv-slot.active {
  border-color: var(--pixel-accent);
  box-shadow:
    inset 0 2px 5px rgba(0,0,0,0.4),
    0 0 0 2px var(--pixel-accent),
    0 3px 0 #4a2a10;
}
.inv-slot.fillable { animation: pulse .7s infinite alternate; }
@keyframes pulse { to { box-shadow: inset 0 2px 5px rgba(0,0,0,0.4), 0 0 0 3px var(--pixel-accent), 0 3px 0 #4a2a10; } }
.inv-slot-key {
  position: absolute; top: 1px; left: 4px; font-size: 11px;
  font-family: var(--pixel-font); color: #fdf1d6;
  text-shadow: 1px 1px 0 rgba(0,0,0,0.4);
}
.inv-slot-icon { width: 38px; height: 38px; }
</style>
