import { defineStore } from 'pinia'
import type { DecorationItemDto } from '../services/editor.service'

const SLOTS = 5
const LS_KEY = 'bigtown:hotbar'

export const useHotbarStore = defineStore('hotbar', {
  state: () => ({
    slots: Array<string | null>(SLOTS).fill(null) as (string | null)[],
    activeIndex: 0,
    itemById: {} as Record<string, DecorationItemDto>,
  }),
  getters: {
    activeItem(state): DecorationItemDto | null {
      const id = state.slots[state.activeIndex]
      return id ? state.itemById[id] ?? null : null
    },
  },
  actions: {
    hydrateCatalog(items: DecorationItemDto[]) {
      for (const it of items) this.itemById[it.id] = it
    },
    load() {
      try {
        const raw = localStorage.getItem(LS_KEY)
        if (raw) {
          const parsed = JSON.parse(raw)
          if (Array.isArray(parsed.slots)) this.slots = parsed.slots.slice(0, SLOTS)
          if (typeof parsed.activeIndex === 'number') this.activeIndex = parsed.activeIndex
        }
      } catch { /* ignore */ }
    },
    persist() {
      try {
        localStorage.setItem(LS_KEY, JSON.stringify({ slots: this.slots, activeIndex: this.activeIndex }))
      } catch { /* ignore */ }
    },
    assign(index: number, itemId: string | null) {
      if (index < 0 || index >= SLOTS) return
      this.slots[index] = itemId
      this.persist()
    },
    clearSlot(index: number) {
      this.assign(index, null)
    },
    setActive(index: number) {
      if (index < 0 || index >= SLOTS) return
      this.activeIndex = index
      this.persist()
      this.emitSelection()
    },
    cycle(delta: number) {
      this.activeIndex = (this.activeIndex + delta + SLOTS) % SLOTS
      this.persist()
      this.emitSelection()
    },
    emitSelection() {
      const item = this.activeItem
      if (item) {
        window.dispatchEvent(new CustomEvent('game:selectDecoration', { detail: { item } }))
      } else {
        window.dispatchEvent(new CustomEvent('game:cancelPlacement'))
      }
    },
  },
})
