<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useGameStore } from '../stores/game.store'
import { useHotbarStore } from '../stores/hotbar.store'
import * as editorService from '../services/editor.service'
import type { DecorationItemDto, PlacementDto } from '../services/editor.service'
import PixelIcon from '@/shared/components/PixelIcon.vue'

const props = defineProps<{
  mapCode: string
}>()

const gameStore = useGameStore()
const hotbarStore = useHotbarStore()
const items = ref<DecorationItemDto[]>([])
const placements = ref<PlacementDto[]>([])
const loading = ref(false)
const errorMessage = ref<string | null>(null)
let errorTimeout: number | null = null
const isResourceLoading = ref(true)

function setResourceLoadingTrue() {
  isResourceLoading.value = true
}

function setResourceLoadingFalse() {
  isResourceLoading.value = false
}

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
    hotbarStore.hydrateCatalog(res.items)

    window.dispatchEvent(new CustomEvent('game:loadPlacements', {
      detail: {
        placements: placements.value,
        items: items.value,
        spawned_coins: res.spawned_coins || []
      }
    }))
  } catch (err) {
    console.error('Failed to load editor data:', err)
  } finally {
    loading.value = false
  }
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

function getOccupiedRects(item: DecorationItemDto, x: number, y: number, tileSize = 16): OccupiedRect[] {
  const rects: OccupiedRect[] = []
  let meta: any = {}
  try {
    meta = JSON.parse(item.metadata_json)
  } catch {}

  const hasFrames = meta.frameWidth !== undefined && meta.frameHeight !== undefined
  const spriteW = hasFrames ? meta.frameWidth : 48
  const spriteH = hasFrames ? meta.frameHeight : 48

  const assetW = hasFrames ? meta.frameWidth : spriteW
  const scale = tileSize / assetW

  const bodyW = (meta.collision_w ?? meta.w ?? spriteW) * scale
  const bodyH = (meta.collision_h ?? meta.h ?? spriteH) * scale

  let offX = 0
  let offY = 0
  if (meta.collision_x !== undefined && meta.collision_y !== undefined) {
    offX = meta.collision_x * scale
    offY = meta.collision_y * scale
  } else {
    const anchorX = meta.anchorX ?? 0.5
    const anchorY = meta.anchorY ?? 1.0
    offX = -bodyW / 2 + (spriteW * scale) * anchorX
    offY = -bodyH + (spriteH * scale) * anchorY
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
        x: x + c.dx * scale,
        y: y + c.dy * scale,
        w: c.w * scale,
        h: c.h * scale
      })
    }
  } else {
    if (item.code && item.code.startsWith('deco_bridge_h_')) {
      rects.push({ x: x, y: y - 28 * scale, w: 48 * scale, h: 8 * scale })
      rects.push({ x: x, y: y - 4 * scale, w: 48 * scale, h: 8 * scale })
    } else if (item.code && item.code.startsWith('deco_bridge_v_')) {
      rects.push({ x: x - 20 * scale, y: y - 16 * scale, w: 8 * scale, h: 32 * scale })
      rects.push({ x: x + 20 * scale, y: y - 16 * scale, w: 8 * scale, h: 32 * scale })
    }
  }

  return rects
}

window.addEventListener('game:checkOccupied', (e: Event) => {
  const detail = (e as CustomEvent).detail as {
    x: number
    y: number
    item: DecorationItemDto
    tileSize: number
    callback: (occupied: boolean) => void
  }

  if (!detail.item) {
    detail.callback(false)
    return
  }

  const previewRects = getOccupiedRects(detail.item, detail.x, detail.y, detail.tileSize)
  let isOccupied = false

  const itemMap = new Map<string, DecorationItemDto>()
  for (const item of items.value) {
    itemMap.set(item.id, item)
  }

  for (const p of placements.value) {
    const pItem = itemMap.get(p.item_id)
    if (!pItem) continue

    const existingRects = getOccupiedRects(pItem, p.x, p.y, detail.tileSize)

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

let onPlacementDone: ((e: Event) => void) | null = null
let onRealtimePlaced: ((e: Event) => void) | null = null
let onRealtimeDeleted: ((e: Event) => void) | null = null
let onPlacementError: ((e: Event) => void) | null = null

onMounted(() => {
  onPlacementDone = (e: Event) => {
    const detail = (e as CustomEvent).detail as { newCoins: number; placement?: PlacementDto; deletedId?: string; sticky?: boolean }
    gameStore.coins = detail.newCoins

    if (detail.placement) {
      if (!placements.value.some(p => p.id === detail.placement!.id)) {
        placements.value.push(detail.placement)
      }
    } else if (detail.deletedId) {
      placements.value = placements.value.filter(p => p.id !== detail.deletedId)
    }
  }
  window.addEventListener('game:placementDone', onPlacementDone)

  onRealtimePlaced = (e: Event) => {
    const detail = (e as CustomEvent).detail as { placement: PlacementDto }
    if (detail.placement.character_id !== gameStore.characterId && !placements.value.some(p => p.id === detail.placement.id)) {
      placements.value.push(detail.placement)
    }
  }
  window.addEventListener('game:realtimePlacementPlaced', onRealtimePlaced)

  onRealtimeDeleted = (e: Event) => {
    const detail = (e as CustomEvent).detail as { placementId: string }
    placements.value = placements.value.filter(p => p.id !== detail.placementId)
  }
  window.addEventListener('game:realtimePlacementDeleted', onRealtimeDeleted)

  onPlacementError = (e: Event) => {
    const detail = (e as CustomEvent).detail as { message: string }
    showErrorMessage(detail.message)
  }
  window.addEventListener('game:placementError', onPlacementError)

  window.addEventListener('game:mapChanged', fetchEditorData)
  window.addEventListener('game:ready', fetchEditorData)
  window.addEventListener('game:mapChanged', setResourceLoadingTrue)
  window.addEventListener('game:ready', setResourceLoadingFalse)
})

onBeforeUnmount(() => {
  if (onPlacementDone) window.removeEventListener('game:placementDone', onPlacementDone)
  if (onRealtimePlaced) window.removeEventListener('game:realtimePlacementPlaced', onRealtimePlaced)
  if (onRealtimeDeleted) window.removeEventListener('game:realtimePlacementDeleted', onRealtimeDeleted)
  if (onPlacementError) window.removeEventListener('game:placementError', onPlacementError)
  window.removeEventListener('game:mapChanged', fetchEditorData)
  window.removeEventListener('game:ready', fetchEditorData)
  window.removeEventListener('game:mapChanged', setResourceLoadingTrue)
  window.removeEventListener('game:ready', setResourceLoadingFalse)
})

watch(() => props.mapCode, (newMapCode) => {
  if (newMapCode === 'winter' || newMapCode === 'dark_village') {
    window.dispatchEvent(new CustomEvent('game:cancelPlacement'))
  }
  fetchEditorData()
})
</script>

<template>
  <div class="editor-panel-container">
    <div v-if="!isResourceLoading" class="coins-display-global ui-badge--coin" aria-label="Player Coins">
      <PixelIcon name="coin" :size="22" />
      <span>{{ gameStore.coins }}</span>
    </div>

    <div v-if="errorMessage" class="error-toast ui-toast" aria-live="polite">
      <PixelIcon name="warning" :size="18" /> {{ errorMessage }}
    </div>
  </div>
</template>

<style scoped>
.editor-panel-container {
  position: absolute;
  bottom: 0;
  right: 0;
  pointer-events: none;
}

.coins-display-global {
  position: fixed;
  top: 70px;
  right: 16px;
  z-index: 9999;
  font-size: var(--fs-head);
  padding: var(--sp-2) var(--sp-3);
}

.error-toast {
  position: fixed;
  bottom: 90px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1010;
  pointer-events: auto;
}
</style>
