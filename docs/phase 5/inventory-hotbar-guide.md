# Hướng dẫn: Inventory kiểu Minecraft + Hotbar + Sticky Brush cho BigTown

Tài liệu này gồm 4 phần, làm theo thứ tự:

1. **Sticky brush** — đặt liên tiếp cùng 1 item, không phải chọn lại (fix rẻ nhất, làm trước).
2. **Chọn bảng DB** — `player_items` vs `character_equipment` vs bảng mới. Kết luận rõ ràng.
3. **Hotbar** (9 ô ở đáy màn hình): phím số 1–9 / lăn chuột để chọn, chọn xong rê chuột đặt như hiện tại.
4. **Inventory modal** (bấm `E`/nút mở): chia nhóm theo danh mục + ô tìm kiếm, kéo/click đồ vào ô hotbar.

Luồng cuối cùng đúng như bạn mô tả:
> Bật tab → chọn đồ trong inventory → gán vào ô hotbar của player → tắt tab → lăn chuột / bấm số để chọn item → rê chuột tới điểm cần đặt (giữ nguyên cơ chế đặt hiện tại) → sticky brush cho đặt liên tiếp.

---

## Phần 0 — Bối cảnh code hiện tại (để hiểu ta đang cắm vào đâu)

Cơ chế đặt item hiện đi qua **CustomEvent** giữa Vue (`EditorPanel.vue`) và Phaser (`editorSystem.ts`):

- `EditorPanel.selectItem(item)` → phát `game:selectDecoration` `{ item }`.
- `editorSystem` bắt event → tạo `previewSprite` bám theo con trỏ; click trái = `confirmPlacement()`.
- `confirmPlacement()` gọi API `placeItem`, rồi `upsertPlacement()` (thêm đúng 1 sprite — đã tối ưu ở PR#36), phát `game:placementDone`, rồi **`clearPreview()`** (huỷ preview + `activeDecorationItem = null`).
- `EditorPanel.onPlacementDone` đặt `activePlacementItemId = null` → palette bỏ chọn.

Điểm quan trọng: **đặt item đang tính bằng coins, KHÔNG trừ trong kho**. Server (`editor_usecase.PlaceItem`) chỉ tra item trong catalog `items` và trừ coins. Nghĩa là "kho đồ" của người chơi bản chất là **danh mục vật thể đặt được** (catalog `items` where `type='decoration'`), số lượng không giới hạn. Điều này ảnh hưởng tới lựa chọn bảng DB ở Phần 2.

Ta sẽ **giữ nguyên** cơ chế `game:selectDecoration`/`confirmPlacement`. Hotbar chỉ là lớp UI mới phát đúng event đó.

---

## Phần 1 — Sticky brush (đặt liên tiếp)

Nguyên nhân phải chọn lại: `confirmPlacement()` gọi `clearPreview()` và `onPlacementDone` set `activePlacementItemId = null`. Chỉ cần **giữ preview + giữ item đang chọn** sau khi đặt thành công.

### 1.1 `editorSystem.ts` — thêm cờ sticky và không clear khi đặt xong

```ts
// (khai báo cạnh các field khác, ~dòng 29)
private stickyBrush = true   // true = đặt liên tiếp; có thể toggle từ UI

// thêm setter để UI bật/tắt
public setStickyBrush(on: boolean) {
  this.stickyBrush = on
}
```

Sửa `confirmPlacement()` — thay `this.clearPreview()` ở nhánh **thành công** bằng logic sticky:

```ts
private async confirmPlacement() {
  if (!this.activeDecorationItem || !this.previewSprite) return

  const x = this.previewSprite.x
  const y = this.previewSprite.y
  const itemId = this.activeDecorationItem.id

  try {
    const result = await editorService.placeItem({
      item_id: itemId,
      map_code: this.mapCode,
      x,
      y,
      rotation: this.placementRotation || undefined,
    })

    this.upsertPlacement(result.placement)

    window.dispatchEvent(new CustomEvent('game:placementDone', {
      detail: { newCoins: result.new_coins, placement: result.placement, sticky: this.stickyBrush }
    }))

    // --- STICKY: giữ item + preview để đặt tiếp; chỉ clear khi tắt sticky ---
    if (!this.stickyBrush) {
      this.clearPreview()
    }
    // preview vẫn còn -> update() tiếp tục cho nó bám con trỏ, click lại là đặt tiếp
  } catch (err: any) {
    console.error('Failed to place item:', err)
    window.dispatchEvent(new CustomEvent('game:placementError', {
      detail: { message: err.message || 'Lỗi không xác định khi đặt vật phẩm' }
    }))
    // Hết coins / lỗi -> nên dừng để người chơi thấy rõ, dù đang sticky
    window.dispatchEvent(new CustomEvent('game:placementCancel'))
    this.clearPreview()
  }
}
```

> Lưu ý: `update()` đã tự cập nhật `canPlace`/tint đỏ khi ô bị chiếm. Vì preview còn sống, sau khi đặt xong người chơi chỉ cần rê sang ô mới và click — không cần thao tác gì thêm.

### 1.2 `EditorPanel.vue` — đừng bỏ chọn khi sticky

```ts
onPlacementDone = (e: Event) => {
  const detail = (e as CustomEvent).detail as {
    newCoins: number; placement?: PlacementDto; deletedId?: string; sticky?: boolean
  }
  gameStore.coins = detail.newCoins

  // Giữ nguyên lựa chọn nếu đang sticky; chỉ bỏ chọn khi KHÔNG sticky
  if (!detail.sticky) {
    activePlacementItemId.value = null
  }

  if (detail.placement) {
    if (!placements.value.some(p => p.id === detail.placement!.id)) {
      placements.value.push(detail.placement)
    }
  } else if (detail.deletedId) {
    placements.value = placements.value.filter(p => p.id !== detail.deletedId)
  }
}
```

### 1.3 (tuỳ chọn) Nút bật/tắt sticky trong palette footer

```vue
<button
  class="btn-delete-mode-pixel"
  :class="{ active: stickyBrush }"
  @click="toggleSticky()"
>🖌️ {{ stickyBrush ? 'Đặt liên tiếp: BẬT' : 'Đặt liên tiếp: TẮT' }}</button>
```

```ts
const stickyBrush = ref(true)
function toggleSticky() {
  stickyBrush.value = !stickyBrush.value
  window.dispatchEvent(new CustomEvent('game:setStickyBrush', { detail: { on: stickyBrush.value } }))
}
```

Trong `editorSystem` setup listeners (chỗ đăng ký các `window.addEventListener`):

```ts
window.addEventListener('game:setStickyBrush', (e: Event) => {
  this.setStickyBrush((e as CustomEvent).detail.on)
})
```

Nhớ `removeEventListener` tương ứng trong `destroy()`.

Sau bước này, bạn đã đặt liên tiếp cùng 1 item bằng cách click liên tục — bấm ESC/Hủy để dừng.

---

## Phần 2 — Chọn bảng DB: `player_items` hay `character_equipment`?

**Kết luận ngắn: không dùng bảng nào trong hai bảng đó cho hotbar. Tạo bảng mới `character_hotbar`.** Lý do:

### `character_equipment` — KHÔNG hợp
```sql
CREATE TABLE character_equipment (
  character_id UUID, slot VARCHAR(50), item_id UUID,
  PRIMARY KEY(character_id, slot),          -- slot là TÊN ('weapon','head'), không phải chỉ số 0..8
  ... ON DELETE RESTRICT                     -- ngăn xoá item đang "mặc"
);
```
Đây là bảng **trang bị chiến đấu** (kiếm/giáp cho `attack_bonus`/`hp_bonus`). Slot là chuỗi tên, không có thứ tự 0–8, và `RESTRICT` mang ngữ nghĩa "đang mặc". Nhét hotbar xây dựng vào đây sẽ **đụng độ khi sau này bạn làm trang bị thật** và trộn hai khái niệm khác nhau.

### `player_items` — gần đúng nhưng vẫn lệch
```sql
CREATE TABLE player_items (
  character_id UUID, item_id UUID, quantity INTEGER,
  UNIQUE(character_id, item_id)              -- 1 item chỉ 1 dòng, không có slot/thứ tự
);
```
Đây là **kho sở hữu + số lượng**. Nhưng trong BigTown, decoration **đặt bằng coins, không tiêu số lượng** → người chơi không "sở hữu" số lượng hữu hạn. Ràng buộc `UNIQUE(character_id, item_id)` cũng cấm cùng 1 item nằm ở 2 ô hotbar và không có cột thứ tự ô. Nó chỉ hợp nếu **sau này** bạn đổi mô hình sang "mua item vào kho rồi đặt sẽ trừ kho".

### Bảng mới `character_hotbar` — khớp Minecraft chính xác
```sql
-- migration: add_character_hotbar.sql
CREATE TABLE IF NOT EXISTS character_hotbar (
  character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  slot_index   SMALLINT NOT NULL CHECK (slot_index >= 0 AND slot_index <= 8),
  item_id      UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (character_id, slot_index)
);
CREATE INDEX IF NOT EXISTS idx_character_hotbar_char ON character_hotbar(character_id);
```
Ưu điểm: có **thứ tự ô** (0–8), cho phép **trùng item** ở nhiều ô, `CASCADE` khi item bị gỡ khỏi catalog, và **không đụng** hai bảng kia — để dành `player_items` cho nền kinh tế tương lai và `character_equipment` cho trang bị thật.

### Lộ trình đề xuất
- **v1 (ship nhanh, không cần backend):** lưu hotbar **client-side** trong Pinia + `localStorage`. Vì nguồn "đặt được gì" vẫn là catalog + coins, hotbar chỉ là tiện ích chọn nhanh. Người chơi chơi 1 máy là đủ.
- **v2 (đồng bộ nhiều thiết bị):** thêm bảng `character_hotbar` + 2 endpoint `GET/PUT /editor/hotbar` để lưu server. Store gọi API thay cho localStorage.

Phần code dưới đây viết theo **v1** (localStorage) và có sẵn chỗ cắm v2.

---

## Phần 3 — Hotbar (9 ô ở đáy màn hình)

### 3.1 Pinia store cho hotbar — `stores/hotbar.store.ts`

```ts
import { defineStore } from 'pinia'
import type { DecorationItemDto } from '../services/editor.service'

const SLOTS = 9
const LS_KEY = 'bigtown:hotbar'

export const useHotbarStore = defineStore('hotbar', {
  state: () => ({
    // mảng 9 phần tử: itemId hoặc null
    slots: Array<string | null>(SLOTS).fill(null) as (string | null)[],
    activeIndex: 0,
    // cache item theo id để render ô mà không cần tra lại catalog
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
      // v2: đổi thành gọi PUT /editor/hotbar { slots }
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
      // lăn chuột: +1 / -1 vòng tròn
      this.activeIndex = (this.activeIndex + delta + SLOTS) % SLOTS
      this.persist()
      this.emitSelection()
    },
    // Phát đúng event mà editorSystem đang nghe
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
```

### 3.2 Component `Hotbar.vue` — hiển thị 9 ô + phím số + lăn chuột

```vue
<script setup lang="ts">
import { onMounted, onBeforeUnmount, computed } from 'vue'
import { useHotbarStore } from '../stores/hotbar.store'
import { useGameStore } from '../stores/game.store'
import { getItemPreviewStyle } from '../utils/itemPreview' // tách hàm getItemPreviewStyle ra file dùng chung

const hotbar = useHotbarStore()
const gameStore = useGameStore()

const slots = computed(() => hotbar.slots.map((id, i) => ({
  index: i,
  item: id ? hotbar.itemById[id] ?? null : null,
})))

function onKeydown(e: KeyboardEvent) {
  // 1..9 -> chọn ô. Bỏ qua khi đang gõ trong input.
  if ((e.target as HTMLElement)?.tagName === 'INPUT') return
  const n = parseInt(e.key, 10)
  if (n >= 1 && n <= 9) {
    hotbar.setActive(n - 1)
    e.preventDefault()
  }
}

function onWheel(e: WheelEvent) {
  // Chỉ cuộn hotbar khi KHÔNG mở inventory (tránh xung đột cuộn danh sách)
  hotbar.cycle(e.deltaY > 0 ? 1 : -1)
  e.preventDefault()
}

onMounted(() => {
  hotbar.load()
  window.addEventListener('keydown', onKeydown)
  // gắn wheel lên canvas game để không chặn cuộn UI khác:
  const canvas = document.getElementById('game-root') // đổi theo id thật của bạn
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
    <div
      v-for="slot in slots"
      :key="slot.index"
      class="hotbar-slot"
      :class="{ active: hotbar.activeIndex === slot.index, disabled: slot.item && gameStore.coins < slot.item.price }"
      @click="hotbar.setActive(slot.index)"
      @contextmenu.prevent="hotbar.clearSlot(slot.index)"
      :title="slot.item ? slot.item.name : 'Ô trống — mở kho (E) để gán'"
    >
      <span class="slot-key">{{ slot.index + 1 }}</span>
      <div v-if="slot.item" class="slot-icon" :style="getItemPreviewStyle(slot.item)"></div>
      <span v-if="slot.item" class="slot-price">🪙{{ slot.item.price }}</span>
    </div>
  </div>
</template>

<style scoped>
.hotbar {
  position: fixed; bottom: 12px; left: 50%; transform: translateX(-50%);
  display: flex; gap: 4px; z-index: 1005; pointer-events: auto;
  padding: 6px; background: var(--pixel-parchment);
  border: 3px solid var(--pixel-wood-dark); border-radius: 4px;
  box-shadow: 0 4px 0 var(--pixel-wood-dark);
}
.hotbar-slot {
  position: relative; width: 52px; height: 52px; cursor: pointer;
  background: #fffaf0; border: 3px solid var(--pixel-wood-dark);
  display: flex; align-items: center; justify-content: center;
  image-rendering: pixelated;
}
.hotbar-slot.active {
  border-color: var(--pixel-accent-dark);
  box-shadow: 0 0 0 2px var(--pixel-accent), inset 0 0 0 2px #fff;
}
.hotbar-slot.disabled { opacity: 0.45; }
.slot-key {
  position: absolute; top: 1px; left: 3px; font-family: var(--pixel-font);
  font-size: 12px; color: var(--pixel-wood-dark);
}
.slot-icon { width: 40px; height: 40px; }
.slot-price {
  position: absolute; bottom: 0; right: 2px; font-family: var(--pixel-font);
  font-size: 11px; color: var(--pixel-accent-dark);
}
</style>
```

> `getItemPreviewStyle` hiện đang nằm trong `EditorPanel.vue`. Tách nó ra `utils/itemPreview.ts` để cả Hotbar và Inventory dùng chung (tránh copy code).

Sau bước này: gán tay vài item vào slots (tạm thời) rồi test — bấm 1–9 hoặc lăn chuột trên canvas là item được chọn và preview hiện ra để đặt.

---

## Phần 4 — Inventory modal (chia nhóm + tìm kiếm + gán vào hotbar)

### 4.1 Chia nhóm danh mục

Hiện mọi item đều `type='decoration'`, nên **suy ra danh mục từ tiền tố `code`** (không cần đổi DB ngay):

```ts
// utils/itemCategory.ts
export type Category = 'tree' | 'house' | 'fence' | 'bridge' | 'flower' | 'rock' | 'ground' | 'other'

export function categoryOf(code: string): Category {
  if (code.includes('tree') || code === 'deco_stump') return 'tree'
  if (code.includes('house')) return 'house'
  if (code.startsWith('deco_fence')) return 'fence'
  if (code.startsWith('deco_bridge')) return 'bridge'
  if (code.includes('flower')) return 'flower'
  if (code.includes('rock')) return 'rock'
  if (code.startsWith('deco_grass')) return 'ground'
  return 'other'
}

export const CATEGORY_LABELS: Record<Category, string> = {
  tree: '🌳 Cây', house: '🏠 Nhà', fence: '🚧 Hàng rào', bridge: '🌉 Cầu',
  flower: '🌷 Hoa', rock: '🪨 Đá', ground: '🌿 Nền cỏ', other: '📦 Khác',
}
```

> **v2 (khuyến nghị khi kho lớn dần):** thêm khoá `"category"` vào `metadata_json` của mỗi item (hoặc cột `category` trong bảng `items`) để phân nhóm chuẩn thay vì đoán theo tên. Khi đó `categoryOf` đọc `meta.category` trước, fallback về đoán tên.

### 4.2 Component `InventoryModal.vue`

```vue
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
const pickedItemId = ref<string | null>(null) // item đang "cầm" để gán vào ô

// Các tab thực sự có item
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
  return tab === 'all' ? '🗂️ Tất cả' : CATEGORY_LABELS[tab]
}

// Click item: "cầm" nó lên; nếu đã cầm sẵn 1 cái, click ô hotbar sẽ gán.
function pickItem(it: DecorationItemDto) {
  pickedItemId.value = pickedItemId.value === it.id ? null : it.id
}

// Gán item đang cầm vào ô hotbar index
function assignToSlot(index: number) {
  if (!pickedItemId.value) return
  hotbar.assign(index, pickedItemId.value)
  pickedItemId.value = null
}

// Cho phép click đúp để gán nhanh vào ô đang active
function quickAssign(it: DecorationItemDto) {
  hotbar.assign(hotbar.activeIndex, it.id)
}
</script>

<template>
  <div class="inv-overlay" @click.self="emit('close')">
    <div class="inv-panel">
      <div class="inv-header">
        <span class="inv-title">KHO VẬT THỂ</span>
        <input v-model="search" class="inv-search" placeholder="Tìm... (tên hoặc mã)" />
        <button class="inv-close" @click="emit('close')">✕</button>
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
          :title="it.name + ' — click để cầm, click đúp để gán vào ô đang chọn'"
        >
          <div class="inv-icon" :style="getItemPreviewStyle(it)"></div>
          <span class="inv-name">{{ it.name }}</span>
          <span class="inv-price">🪙{{ it.price }}</span>
        </div>
        <p v-if="filtered.length === 0" class="inv-empty">Không có vật thể khớp.</p>
      </div>

      <!-- Hàng hotbar ngay trong modal để kéo/gán trực quan -->
      <div class="inv-hotbar-row">
        <span class="inv-hint">
          {{ pickedItemId ? 'Chọn ô bên dưới để gán ↓' : 'Click 1 vật thể ở trên để cầm' }}
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
                 :style="getItemPreviewStyle(hotbar.itemById[id])"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.inv-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center; z-index: 1100;
}
.inv-panel {
  width: min(680px, 92vw); max-height: 82vh; display: flex; flex-direction: column;
  background: var(--pixel-parchment); padding: 16px;
  box-shadow: 0 0 0 4px var(--pixel-wood-dark), 0 0 0 8px var(--pixel-wood), 0 16px 28px rgba(0,0,0,.45);
}
.inv-header { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; }
.inv-title { font-family: var(--pixel-font); font-size: 24px; color: var(--pixel-wood-dark); }
.inv-search {
  flex: 1; font-family: var(--pixel-font); font-size: 16px; padding: 6px 10px;
  border: 3px solid var(--pixel-wood-dark); background: #fffaf0;
}
.inv-close { border: 3px solid var(--pixel-wood-dark); background: var(--pixel-danger); color:#fff;
  width: 34px; height: 34px; cursor: pointer; }
.inv-tabs { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 10px; }
.inv-tab {
  font-family: var(--pixel-font); font-size: 14px; padding: 4px 10px; cursor: pointer;
  border: 2px solid var(--pixel-wood-dark); background: #fffaf0;
}
.inv-tab.active { background: var(--pixel-accent); color: #fff; }
.inv-grid {
  flex: 1; overflow-y: auto; display: grid;
  grid-template-columns: repeat(auto-fill, minmax(84px, 1fr)); gap: 8px; padding-right: 4px;
}
.inv-item {
  display: flex; flex-direction: column; align-items: center; gap: 3px; padding: 6px;
  border: 3px solid var(--pixel-wood-dark); background: #fffaf0; cursor: pointer;
}
.inv-item.picked { box-shadow: 0 0 0 3px var(--pixel-accent); background: var(--pixel-parchment-dark); }
.inv-item.disabled { opacity: .45; }
.inv-icon { width: 48px; height: 48px; image-rendering: pixelated; }
.inv-name { font-family: var(--pixel-font); font-size: 12px; text-align: center;
  max-width: 80px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.inv-price { font-family: var(--pixel-font); font-size: 12px; color: var(--pixel-accent-dark); }
.inv-empty { grid-column: 1 / -1; text-align: center; font-family: var(--pixel-font); padding: 20px; }
.inv-hotbar-row { margin-top: 12px; border-top: 2px dashed var(--pixel-wood-dark); padding-top: 10px; }
.inv-hint { font-family: var(--pixel-font); font-size: 14px; color: var(--pixel-wood-dark); }
.inv-hotbar { display: flex; gap: 4px; justify-content: center; margin-top: 6px; }
.inv-slot { position: relative; width: 46px; height: 46px; border: 3px solid var(--pixel-wood-dark);
  background: #fffaf0; cursor: pointer; }
.inv-slot.active { border-color: var(--pixel-accent-dark); box-shadow: 0 0 0 2px var(--pixel-accent); }
.inv-slot.fillable { animation: pulse .8s infinite alternate; }
@keyframes pulse { to { box-shadow: 0 0 0 2px var(--pixel-accent); } }
.inv-slot-key { position: absolute; top: 1px; left: 3px; font-size: 11px; font-family: var(--pixel-font); }
.inv-slot-icon { width: 38px; height: 38px; margin: 3px auto; }
</style>
```

### 4.3 Ráp vào màn game

Trong component chứa game (nơi đang render `EditorPanel`), thêm Hotbar + Inventory và phím `E`:

```vue
<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useHotbarStore } from '../stores/hotbar.store'
import Hotbar from './Hotbar.vue'
import InventoryModal from './InventoryModal.vue'
import * as editorService from '../services/editor.service'
import type { DecorationItemDto } from '../services/editor.service'

const hotbar = useHotbarStore()
const items = ref<DecorationItemDto[]>([])
const invOpen = ref(false)

async function loadCatalog(mapCode: string) {
  const res = await editorService.getEditorData(mapCode)
  items.value = res.items
  hotbar.hydrateCatalog(res.items)   // để hotbar render icon theo id
}

function onKey(e: KeyboardEvent) {
  if ((e.target as HTMLElement)?.tagName === 'INPUT') return
  if (e.key === 'e' || e.key === 'E') { invOpen.value = !invOpen.value; e.preventDefault() }
  if (e.key === 'Escape' && invOpen.value) { invOpen.value = false }
}

onMounted(() => {
  hotbar.load()
  window.addEventListener('keydown', onKey)
  // gọi loadCatalog(mapCode) khi map sẵn sàng — tái dùng event game:ready sẵn có
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <!-- ...game canvas... -->
  <Hotbar />
  <InventoryModal v-if="invOpen" :items="items" @close="invOpen = false" />
</template>
```

**Luồng hoàn chỉnh:** `E` mở kho → chọn tab / gõ tìm → click item để "cầm" → click ô hotbar (1–9) để gán → `E`/Esc đóng kho → lăn chuột hoặc bấm số chọn ô → `game:selectDecoration` phát ra → preview hiện, rê chuột, click đặt → sticky brush cho đặt liên tiếp.

---

## Phần 5 — (v2 tuỳ chọn) Lưu hotbar lên server

Khi muốn đồng bộ nhiều thiết bị, thêm 2 endpoint và bảng `character_hotbar` (SQL ở Phần 2).

Backend (phác thảo, đặt cạnh editor module):
```
GET  /editor/hotbar            -> { slots: [item_id|null x9] }
PUT  /editor/hotbar            body { slots: [item_id|null x9] }
```
- `GET`: `SELECT slot_index, item_id FROM character_hotbar WHERE character_id=$1` → dựng mảng 9 phần tử.
- `PUT`: transaction `DELETE ... WHERE character_id=$1` rồi `INSERT` các ô != null. Validate `item_id` tồn tại trong `items` và `slot_index` 0..8.

Frontend: trong store, đổi `load()` → gọi `GET /editor/hotbar`, `persist()` → `PUT /editor/hotbar` (debounce ~500ms để không spam khi kéo thả nhiều).

---

## Checklist triển khai

- [ ] Phần 1: sticky brush (`editorSystem.ts` + `EditorPanel.vue`) — test đặt liên tiếp.
- [ ] Tách `getItemPreviewStyle` → `utils/itemPreview.ts`; thêm `utils/itemCategory.ts`.
- [ ] Phần 3: `hotbar.store.ts` + `Hotbar.vue` — test phím 1–9 và lăn chuột.
- [ ] Phần 4: `InventoryModal.vue` + ráp phím `E` — test gán item vào ô.
- [ ] Chạy `reencode-audio-opus.sh` để .ogg nhỏ hơn .mp3 (không liên quan hotbar nhưng nên làm luôn).
- [ ] (v2) Bảng `character_hotbar` + 2 endpoint khi cần đồng bộ.
