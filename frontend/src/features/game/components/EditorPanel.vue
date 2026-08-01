<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useGameStore } from '../stores/game.store'
import * as editorService from '../services/editor.service'
import type { DecorationItemDto, PlacementDto } from '../services/editor.service'

const props = defineProps<{
  mapCode: string
}>()

const gameStore = useGameStore()
const isOpen = ref(false)
const items = ref<DecorationItemDto[]>([])
const placements = ref<PlacementDto[]>([])
const activePlacementItemId = ref<string | null>(null)
const isDeleteMode = ref(false)
const loading = ref(false)
const errorMessage = ref<string | null>(null)
let errorTimeout: number | null = null

function showErrorMessage(msg: string) {
  errorMessage.value = msg
  if (errorTimeout) clearTimeout(errorTimeout)
  errorTimeout = window.setTimeout(() => {
    errorMessage.value = null
  }, 4000)
}

async function fetchEditorData() {
  if (!props.mapCode) return
  loading.value = true
  try {
    const res = await editorService.getEditorData(props.mapCode)
    items.value = res.items
    placements.value = res.placements || []
    gameStore.coins = res.coins
    
    // Dispatch placements to Phaser so they can be rendered
    window.dispatchEvent(new CustomEvent('game:loadPlacements', {
      detail: { placements: placements.value, items: items.value }
    }))
  } catch (err) {
    console.error('Failed to load editor data:', err)
  } finally {
    loading.value = false
  }
}

function toggleEditor() {
  isOpen.value = !isOpen.value
  window.dispatchEvent(new CustomEvent('game:toggleEditorMode', {
    detail: { active: isOpen.value }
  }))
  if (isOpen.value) {
    fetchEditorData()
  } else {
    cancelPlacement()
    toggleDeleteMode(false)
  }
}

function selectItem(item: DecorationItemDto) {
  if (gameStore.coins < item.price) return
  toggleDeleteMode(false) // Deactivate delete mode when choosing an item to place
  activePlacementItemId.value = item.id
  window.dispatchEvent(new CustomEvent('game:selectDecoration', {
    detail: { item }
  }))
}

interface OccupiedRect {
  x: number
  y: number
  w: number
  h: number
}

function intersects(ax: number, ay: number, aw: number, ah: number, bx: number, by: number, bw: number, bh: number): boolean {
  return ax < bx + bw && ax + aw > bx && ay < by + bh && ay + ah > by
}

function getOccupiedRects(item: DecorationItemDto, x: number, y: number): OccupiedRect[] {
  const rects: OccupiedRect[] = []
  let meta: any = {}
  try {
    meta = JSON.parse(item.metadata_json)
  } catch {}

  const hasFrames = meta.frameWidth !== undefined && meta.frameHeight !== undefined
  const spriteW = hasFrames ? meta.frameWidth : 48
  const spriteH = hasFrames ? meta.frameHeight : 48

  const bodyW = meta.collision_w ?? meta.w ?? spriteW
  const bodyH = meta.collision_h ?? meta.h ?? spriteH

  let offX = 0
  let offY = 0
  if (meta.collision_x !== undefined && meta.collision_y !== undefined) {
    offX = meta.collision_x
    offY = meta.collision_y
  } else {
    const anchorX = meta.anchorX ?? 0.5
    const anchorY = meta.anchorY ?? 1.0
    offX = -bodyW / 2 + spriteW * anchorX
    offY = -bodyH + spriteH * anchorY
  }

  if (meta.collides) {
    rects.push({
      x: x + offX,
      y: y + offY,
      w: bodyW,
      h: bodyH
    })
  }

  if (Array.isArray(meta.extra_colliders)) {
    for (const c of meta.extra_colliders) {
      rects.push({
        x: x + c.dx,
        y: y + c.dy,
        w: c.w,
        h: c.h
      })
    }
  } else {
    if (item.code.startsWith('deco_bridge_h_')) {
      rects.push({ x: x, y: y - 28, w: 48, h: 8 })
      rects.push({ x: x, y: y - 4, w: 48, h: 8 })
    } else if (item.code.startsWith('deco_bridge_v_')) {
      rects.push({ x: x - 20, y: y - 16, w: 8, h: 32 })
      rects.push({ x: x + 20, y: y - 16, w: 8, h: 32 })
    }
  }

  return rects
}

// Check if placement coordinates are occupied
window.addEventListener('game:checkOccupied', (e: Event) => {
  const detail = (e as CustomEvent).detail as {
    x: number
    y: number
    item: DecorationItemDto
    callback: (occupied: boolean) => void
  }

  if (!detail.item) {
    detail.callback(false)
    return
  }

  const previewRects = getOccupiedRects(detail.item, detail.x, detail.y)
  let isOccupied = false

  const itemMap = new Map<string, DecorationItemDto>()
  for (const item of items.value) {
    itemMap.set(item.id, item)
  }

  for (const p of placements.value) {
    const pItem = itemMap.get(p.item_id)
    if (!pItem) continue

    const existingRects = getOccupiedRects(pItem, p.x, p.y)

    for (const pr of previewRects) {
      for (const er of existingRects) {
        if (intersects(pr.x, pr.y, pr.w, pr.h, er.x, er.y, er.w, er.h)) {
          isOccupied = true
          break
        }
      }
      if (isOccupied) break
    }
    if (isOccupied) break
  }

  detail.callback(isOccupied)
})

function cancelPlacement() {
  activePlacementItemId.value = null
  window.dispatchEvent(new CustomEvent('game:cancelPlacement'))
}

function toggleDeleteMode(forceState?: boolean) {
  isDeleteMode.value = forceState !== undefined ? forceState : !isDeleteMode.value
  if (isDeleteMode.value) {
    cancelPlacement() // Deactivate placement mode when delete mode is active
  }
  window.dispatchEvent(new CustomEvent('game:toggleDeleteMode', {
    detail: { active: isDeleteMode.value }
  }))
}

// Event Listeners from Phaser & Realtime socket
let onPlacementDone: ((e: Event) => void) | null = null
let onPlacementCancel: ((e: Event) => void) | null = null
let onRealtimePlaced: ((e: Event) => void) | null = null
let onRealtimeDeleted: ((e: Event) => void) | null = null
let onPlacementError: ((e: Event) => void) | null = null

onMounted(() => {
  onPlacementDone = (e: Event) => {
    const detail = (e as CustomEvent).detail as { newCoins: number; placement?: PlacementDto; deletedId?: string }
    gameStore.coins = detail.newCoins
    activePlacementItemId.value = null
    
    if (detail.placement) {
      // Append the new placement locally
      placements.value.push(detail.placement)
    } else if (detail.deletedId) {
      // Remove the deleted placement locally
      placements.value = placements.value.filter(p => p.id !== detail.deletedId)
    }

    // Immediately trigger Phaser redraw with new list
    window.dispatchEvent(new CustomEvent('game:loadPlacements', {
      detail: { placements: placements.value, items: items.value }
    }))
  }
  window.addEventListener('game:placementDone', onPlacementDone)

  onPlacementCancel = () => {
    activePlacementItemId.value = null
  }
  window.addEventListener('game:placementCancel', onPlacementCancel)

  onRealtimePlaced = (e: Event) => {
    const detail = (e as CustomEvent).detail as { placement: PlacementDto }
    if (detail.placement.character_id !== gameStore.characterId && !placements.value.some(p => p.id === detail.placement.id)) {
      placements.value.push(detail.placement)
      window.dispatchEvent(new CustomEvent('game:loadPlacements', {
        detail: { placements: placements.value, items: items.value }
      }))
    }
  }
  window.addEventListener('game:realtimePlacementPlaced', onRealtimePlaced)

  onRealtimeDeleted = (e: Event) => {
    const detail = (e as CustomEvent).detail as { placementId: string }
    if (placements.value.some(p => p.id === detail.placementId)) {
      placements.value = placements.value.filter(p => p.id !== detail.placementId)
      window.dispatchEvent(new CustomEvent('game:loadPlacements', {
        detail: { placements: placements.value, items: items.value }
      }))
    }
  }
  window.addEventListener('game:realtimePlacementDeleted', onRealtimeDeleted)

  onPlacementError = (e: Event) => {
    const detail = (e as CustomEvent).detail as { message: string }
    showErrorMessage(detail.message)
  }
  window.addEventListener('game:placementError', onPlacementError)
  
  // Listen to map changes and game ready to reload placements
  window.addEventListener('game:mapChanged', fetchEditorData)
  window.addEventListener('game:ready', fetchEditorData)

  // Load editor data immediately on mount
  fetchEditorData()
})

onBeforeUnmount(() => {
  if (onPlacementDone) window.removeEventListener('game:placementDone', onPlacementDone)
  if (onPlacementCancel) window.removeEventListener('game:placementCancel', onPlacementCancel)
  if (onRealtimePlaced) window.removeEventListener('game:realtimePlacementPlaced', onRealtimePlaced)
  if (onRealtimeDeleted) window.removeEventListener('game:realtimePlacementDeleted', onRealtimeDeleted)
  if (onPlacementError) window.removeEventListener('game:placementError', onPlacementError)
  window.removeEventListener('game:mapChanged', fetchEditorData)
  window.removeEventListener('game:ready', fetchEditorData)
})

watch(() => props.mapCode, () => {
  if (isOpen.value) {
    fetchEditorData()
  }
})

function getItemPreviewStyle(item: DecorationItemDto): Record<string, any> {
  let meta: any = {}
  try {
    meta = JSON.parse(item.metadata_json)
  } catch {}

  if (meta.frameWidth !== undefined && meta.frameHeight !== undefined && meta.frame !== undefined) {
    let sheetW = 0
    let sheetH = 0
    
    const key = item.asset_key.toLowerCase()
    if (key.includes('fences.png')) {
      sheetW = 64
      sheetH = 64
    } else if (key.includes('oak_tree_small.png')) {
      sheetW = 96
      sheetH = 48
    } else if (key.includes('bridge_wood.png')) {
      sheetW = 144
      sheetH = 64
    }

    if (sheetW > 0 && sheetH > 0) {
      const cols = sheetW / meta.frameWidth
      const col = meta.frame % cols
      const row = Math.floor(meta.frame / cols)

      const posX = -col * meta.frameWidth
      const posY = -row * meta.frameHeight

      const maxDim = Math.max(meta.frameWidth, meta.frameHeight)
      const scale = maxDim > 0 ? Math.min(64 / maxDim, 1) : 1
      
      return {
        width: `${meta.frameWidth}px`,
        height: `${meta.frameHeight}px`,
        backgroundImage: `url(/assets/${item.asset_key})`,
        backgroundPosition: `${posX}px ${posY}px`,
        backgroundSize: `${sheetW}px ${sheetH}px`,
        imageRendering: 'pixelated',
        transform: `scale(${scale})`,
        transformOrigin: 'center'
      }
    }
  }

  return {
    width: '100%',
    height: '100%',
    backgroundImage: `url(/assets/${item.asset_key})`,
    backgroundPosition: 'center',
    backgroundSize: 'contain',
    backgroundRepeat: 'no-repeat',
    imageRendering: 'pixelated'
  }
}
</script>

<template>
  <div class="editor-panel-container">
    <!-- Toggle Button (Settings style) -->
    <button 
      class="btn-toggle-editor-pixel" 
      @click="toggleEditor" 
      :class="{ active: isOpen }"
      :aria-label="isOpen ? 'Đóng chế độ thiết kế' : 'Mở chế độ thiết kế'"
    >
      <span class="btn-icon">🛠️</span>
    </button>

    <!-- Editor Palette Overlay (Parchment styled like Login box) -->
    <transition name="pixel-slide">
      <div v-if="isOpen" class="editor-palette-pixel" aria-label="Editor Palette">
        <div class="palette-header">
          <span class="title">BẢN ĐỒ THIẾT KẾ</span>
          <div class="coins-display">
            <span class="coin-icon">🪙</span>
            <span class="coin-amount">{{ gameStore.coins }}</span>
          </div>
        </div>

        <div v-if="loading" class="palette-loading">
          Đang đọc bản thiết kế...
        </div>

        <div v-else class="palette-grid">
          <div 
            v-for="item in items" 
            :key="item.id"
            class="palette-item"
            :class="{ 
              selected: activePlacementItemId === item.id,
              disabled: gameStore.coins < item.price
            }"
            @click="selectItem(item)"
          >
            <div class="item-preview-box">
              <div :style="getItemPreviewStyle(item)" :aria-label="item.name"></div>
            </div>
            <div class="item-info">
              <span class="item-name">{{ item.name }}</span>
              <span class="item-price">🪙{{ item.price }}</span>
            </div>
          </div>
        </div>

        <!-- Delete Mode Toggle & Hint Footer -->
        <div class="palette-footer">
          <button 
            class="btn-delete-mode-pixel"
            :class="{ active: isDeleteMode }"
            @click="toggleDeleteMode()"
          >
            🗑️ {{ isDeleteMode ? 'Tắt Xóa' : 'Xóa Vật Phẩm' }}
          </button>
        </div>

        <div v-if="activePlacementItemId" class="placement-hint">
          <span class="hint-text">Chọn vị trí trên map để đặt. Nhấn ESC hoặc nút bên cạnh để hủy.</span>
          <button class="btn-cancel-pixel" @click="cancelPlacement">Hủy</button>
        </div>

        <div v-if="isDeleteMode" class="placement-hint delete-hint">
          <span class="hint-text text-danger">Click vào vật phẩm đã đặt trên map để xóa và được hoàn tiền 100%.</span>
        </div>

        <div v-if="errorMessage" class="placement-hint error-hint" aria-live="polite">
          <span class="hint-text text-danger">⚠️ {{ errorMessage }}</span>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.editor-panel-container {
  position: absolute;
  bottom: 0;
  right: 0;
  pointer-events: none;
}

/* 2D Settings Button Style */
.btn-toggle-editor-pixel {
  position: fixed;
  bottom: 16px;
  right: 16px;
  width: 48px;
  height: 48px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: var(--pixel-parchment);
  border: 3px solid var(--pixel-wood-dark);
  border-radius: 4px;
  cursor: pointer;
  z-index: 1010;
  pointer-events: auto;
  box-shadow:
    0 4px 0 var(--pixel-wood-dark),
    inset -3px -3px 0 var(--pixel-parchment-dark);
  transition: transform 0.1s ease, box-shadow 0.1s ease;
}

.btn-toggle-editor-pixel:hover {
  background: var(--pixel-parchment-dark);
}

.btn-toggle-editor-pixel:active,
.btn-toggle-editor-pixel.active {
  transform: translateY(2px);
  background: var(--pixel-accent);
  box-shadow:
    0 2px 0 var(--pixel-wood-dark),
    inset -3px -3px 0 var(--pixel-accent-dark);
}

.btn-icon {
  font-size: 20px;
  line-height: 1;
}

/* Editor Palette (Login Card vibe) */
.editor-palette-pixel {
  position: fixed;
  bottom: 76px;
  right: 16px;
  width: 350px;
  background: var(--pixel-parchment);
  padding: 20px;
  z-index: 1009;
  pointer-events: auto;
  box-shadow:
    0 0 0 4px var(--pixel-wood-dark),
    0 0 0 8px var(--pixel-wood),
    0 0 0 11px var(--pixel-wood-dark),
    0 16px 28px rgba(0, 0, 0, 0.45);
}

.palette-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 3px solid var(--pixel-wood-dark);
  padding-bottom: 8px;
  margin-bottom: 12px;
}

.palette-header .title {
  font-family: var(--pixel-font);
  color: var(--pixel-wood-dark);
  font-size: 24px;
  font-weight: bold;
  letter-spacing: 0.5px;
  text-shadow: 1px 1px 0 rgba(255, 255, 255, 0.5);
}

.coins-display {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--pixel-parchment-dark);
  border: 2px solid var(--pixel-wood-dark);
  border-radius: 2px;
  padding: 2px 8px;
  box-shadow: inset 1px 1px 0 rgba(255, 255, 255, 0.8);
}

.coin-amount {
  font-family: var(--pixel-font);
  color: var(--pixel-ink);
  font-weight: bold;
  font-size: 18px;
}

.palette-loading {
  font-family: var(--pixel-font);
  color: var(--pixel-wood-dark);
  text-align: center;
  padding: 24px;
  font-size: 20px;
}

.palette-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  max-height: 220px;
  overflow-y: auto;
  padding-right: 4px;
}

/* Custom Scrollbar for Retro feeling */
.palette-grid::-webkit-scrollbar {
  width: 6px;
}

.palette-grid::-webkit-scrollbar-track {
  background: var(--pixel-parchment-dark);
  border-left: 2px solid var(--pixel-wood-dark);
}

.palette-grid::-webkit-scrollbar-thumb {
  background: var(--pixel-wood-dark);
}

.palette-item {
  background: #fffaf0;
  border: 3px solid var(--pixel-wood-dark);
  border-radius: 2px;
  padding: 8px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  box-shadow: inset 1px 1px 0 rgba(255, 255, 255, 0.9);
  transition: transform 0.1s ease;
}

.palette-item:hover:not(.disabled) {
  background: var(--pixel-parchment-dark);
  transform: scale(1.03);
}

.palette-item.selected {
  background: var(--pixel-parchment-dark);
  border-color: var(--pixel-accent-dark);
  box-shadow: 
    0 0 0 2px var(--pixel-accent),
    inset 1px 1px 0 rgba(255, 255, 255, 0.9);
}

.palette-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.item-preview-box {
  width: 64px;
  height: 64px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: rgba(107, 66, 38, 0.08);
  border: 2px dashed rgba(107, 66, 38, 0.3);
  overflow: hidden;
}

.item-preview-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  image-rendering: pixelated;
}

.item-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
}

.item-name {
  font-family: var(--pixel-font);
  color: var(--pixel-ink);
  font-size: 18px;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

.item-price {
  font-family: var(--pixel-font);
  color: var(--pixel-accent-dark);
  font-size: 16px;
  font-weight: bold;
}

.palette-footer {
  margin-top: 12px;
  border-top: 2px dashed var(--pixel-wood-dark);
  padding-top: 12px;
  display: flex;
  justify-content: center;
}

.btn-delete-mode-pixel {
  width: 100%;
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

.btn-delete-mode-pixel:hover {
  background: #ffe0b2;
}

.btn-delete-mode-pixel:active,
.btn-delete-mode-pixel.active {
  transform: translateY(2px);
  background: var(--pixel-danger);
  color: #fff;
  box-shadow:
    0 2px 0 var(--pixel-wood-dark),
    inset -2px -2px 0 rgba(0, 0, 0, 0.2);
}

.placement-hint {
  margin-top: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fffde7;
  border: 2px dashed var(--pixel-wood-dark);
  padding: 6px 10px;
  border-radius: 2px;
  gap: 8px;
}

.placement-hint.delete-hint {
  background: #ffebee;
  border-color: var(--pixel-danger);
}

.placement-hint.error-hint {
  background: #ffebee;
  border-color: var(--pixel-danger);
  animation: shake 0.2s ease-in-out 2;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-4px); }
  75% { transform: translateX(4px); }
}

.hint-text {
  font-family: var(--pixel-font);
  color: var(--pixel-ink);
  font-size: 16px;
  line-height: 1.1;
}

.hint-text.text-danger {
  color: var(--pixel-danger);
  font-weight: bold;
}

.btn-cancel-pixel {
  background: var(--pixel-danger);
  border: 2px solid var(--pixel-ink);
  color: #fff;
  padding: 2px 8px;
  cursor: pointer;
  font-family: var(--pixel-font);
  font-size: 16px;
  box-shadow:
    0 2px 0 var(--pixel-ink),
    inset -1px -1px 0 rgba(0, 0, 0, 0.2);
}

.btn-cancel-pixel:active {
  transform: translateY(1px);
  box-shadow: 0 1px 0 var(--pixel-ink);
}

/* Animations */
.pixel-slide-enter-active,
.pixel-slide-leave-active {
  transition: transform 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275), opacity 0.2s ease;
}

.pixel-slide-enter-from,
.pixel-slide-leave-to {
  transform: translateY(20px);
  opacity: 0;
}
</style>
