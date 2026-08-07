# BigTown — Nhà người chơi: interior qua "cutaway" (tách mái lúc đặt runtime)

Mục tiêu: người chơi **đặt** một căn nhà (qua hệ placement sẵn có), bước vào thì **mái mờ đi** để nhìn thấy bên trong + nội thất, mà **không** phải vẽ map nội thất riêng và **không** cần animation.

Vấn đề cốt lõi phải giải: nhà do người chơi đặt là **một sprite runtime, mái bị nướng chung vào texture**. Không có "layer mái" như nhà vẽ sẵn trong Tiled, nên không thể fade mái ở cấp map.

---

## 0. Bối cảnh & hai hướng thiết kế

| | Hướng A — Cutaway (khuyến nghị làm trước) | Hướng B — Instanced interior (nâng cấp sau) |
|---|---|---|
| Vào nhà | Nhìn xuyên xuống footprint, cùng map | Đổi sang map nội thất riêng |
| Chi phí chính | **Client render** (crop + fade), asset gần như 0 | **Asset** (bộ tường/sàn nội thất) |
| Backend | Không đổi (cùng map) | Gần như miễn phí — interior là `mapCode` mới, `RoomManager.Actor()` tự lo |
| Xã hội | Vẫn thấy người khác quanh nhà | Interior riêng tư/instanced |
| Kích thước trong | Bằng footprint nhà | To tuỳ ý |

**Chốt:** ship Hướng A trước (tài liệu này). Khi tính năng "xây nhà" được đón nhận, thêm Hướng B cho "nhà lớn/premium" (một interior kit auto-generate qua `generate_map.js`, mỗi nhà = một `mapCode` như `house_<placementId>`, cửa ra/vào = đúng `OnPlayerJoin/OnPlayerLeave`). Hai hướng bổ trợ: nhà nhỏ = cutaway, biệt thự = instanced.

> ⚠️ **License asset:** bản Cute Fantasy đang dùng là **Free = phi thương mại**. Dự án công ty phải nâng lên **bản trả phí** (cho phép commercial). Bản trả phí cũng **đã có sẵn interior/floor/wall/furniture cùng style** (các update Houses/Interiors) — giải quyết luôn bài toán asset nội thất, tĩnh, không cần animation.

---

## 1. Ý tưởng then chốt: tách ở ĐỊNH NGHĨA ITEM, không tách ở map

Một placement = **một itemID, một ô, một dòng DB**. Nhưng renderer sinh **2 sprite** từ **cùng một texture**:

- **base** — phần dưới (tường/sàn/cửa/cửa sổ): depth thường, có collision.
- **roof** — phần mái: depth cao, mang behavior `roof_fade`.

Việc "tách" khai báo **một lần** trong metadata của item nhà (toạ độ crop), nên đúng cho **mọi** căn nhà người chơi đặt runtime — kể cả căn vừa mới sinh.

Sprite nhà `House_1_Wood_Base_Blue` là 96×128, có đường mái rõ. Metadata:

```json
{
  "collides": true, "anchorX": 0.5, "anchorY": 1.0,
  "footprint_w": 96, "footprint_h": 80,
  "parts": [
    { "name": "base", "crop": [0, 48, 96, 80], "depth": 2,  "collides": true },
    { "name": "roof", "crop": [0, 0, 96, 56],  "depth": 30, "behaviors": ["roof_fade"] }
  ]
}
```
> Số crop chỉ minh hoạ (`[x, y, w, h]`); mái ~hàng trên, base ~hàng dưới, cho chồng nhẹ vài px là ổn.

**Vì sao khớp pixel-perfect miễn phí:** `setCrop` không đổi vị trí/anchor — nó chỉ giấu pixel ngoài vùng crop. Hai sprite tạo từ **cùng texture, cùng (x,y), cùng origin** sẽ tự căn chồng chính xác: `base` hiện phần dưới, `roof` hiện phần trên. Không cần cắt file, không cần layer Tiled.

*(Nếu muốn rendering tối giản hơn: cắt sẵn `house_base.png` + `house_roof.png` — cũng chỉ một lần, không phải mỗi nhà.)*

---

## 2. Mở rộng renderer để lặp qua `parts`

`renderPlacementsGroup` hiện tạo **1 sprite/placement**. Thêm nhánh: nếu có `meta.parts` thì lặp, mỗi part một sprite, cùng `placementId` (để reconcile/xoá vẫn chạy), thêm tag `part`. Không có `parts` → giữ nguyên đường cũ (backward-compatible).

```ts
// Bên trong renderPlacementsGroup, chỗ "Create new sprite":
const parts = meta.parts ?? [{ name: 'whole', depth: meta.depth, collides: meta.collides }]

for (const part of parts) {
  const sp = this.scene.add.image(p.x, p.y, item.asset_key)

  if (part.crop) sp.setCrop(part.crop[0], part.crop[1], part.crop[2], part.crop[3])

  sp.setOrigin(meta.anchorX ?? 0.5, meta.anchorY ?? 1.0)
  const baseDepth = typeof part.depth === 'number' ? part.depth : (meta.depth ?? 2)
  sp.setDepth(baseDepth + p.y / 10000.0)   // giữ đúng công thức y-sort hiện có

  sp.setData('placementId', p.id)
  sp.setData('part', part.name)

  // mỗi part có behavior riêng (roof mang roof_fade), tái dùng applyBehaviors*
  const partMeta = { ...meta, behaviors: part.behaviors ?? [] }
  sp.setData('meta', partMeta)
  applyBehaviorsOnCreate(sp, partMeta, {
    scene: this.scene, collisionGroup: this.collisionGroup, tileSize: this.tileSize,
  })

  // collision chỉ khi part.collides (xem mục 4 cho viền tường + ô cửa)
  if (part.collides) {
    this.scene.physics.add.existing(sp, true)
    // ... setSize/setOffset như code collision hiện tại
  }

  this.placementsGroup.add(sp)
}
```

**Lưu ý reconcile:** phần diff sprite theo `placementId` hiện dùng `Map<placementId, sprite>` giả định **1 sprite/placement**. Khi một placement có nhiều part, đổi khoá thành `${placementId}:${part}` (hoặc lưu mảng sprite theo placementId) để ret/xoá đúng từng part. Đây là thay đổi bắt buộc nhỏ ở vòng reconcile.

---

## 3. Behavior `roof_fade` — sao chép gần nguyên `fadeBehind`

Chỉ khác điều kiện kích hoạt: fade khi người chơi **ở trong footprint** (thay vì "đứng sau sprite"). Tween alpha giữ y hệt `fadeBehind` (150ms, Power1).

`frontend/src/features/game/systems/behaviors/roofFade.ts`:

```ts
import type { BehaviorHandler } from './types'

export const roofFade: BehaviorHandler = {
  onUpdate(sprite, meta, state) {
    const { player } = state
    const fw = meta.footprint_w ?? sprite.displayWidth
    const fh = meta.footprint_h ?? sprite.displayHeight
    const left = sprite.x - fw / 2
    const top  = sprite.y - fh                 // chân nhà tại sprite.y (origin y = 1)

    const inside =
      player.x >= left && player.x <= left + fw &&
      player.y >= top  && player.y <= sprite.y

    const targetAlpha = inside ? 0.15 : 1.0
    if (sprite.getData('targetAlpha') !== targetAlpha) {
      sprite.setData('targetAlpha', targetAlpha)
      sprite.scene.tweens.killTweensOf(sprite)
      sprite.scene.tweens.add({
        targets: sprite, alpha: targetAlpha, duration: 150, ease: 'Power1',
      })
    }
  },
}
```

Đăng ký trong `behaviors/index.ts`:

```ts
import { roofFade } from './roofFade'

export const BEHAVIORS: Record<string, BehaviorHandler> = {
  fade_behind: fadeBehind,
  glow_night: glowNight,
  bridge,
  roof_fade: roofFade,   // THÊM
}
```

> Tuỳ chọn hysteresis (chống nhấp nháy ở ranh giới): dùng 2 bán kính `inside`/`outside` như doc voice đã làm cho proximity — vào khi trong footprint, chỉ bật lại mái khi ra khỏi footprint + biên đệm vài px.

---

## 4. Va chạm: viền tường chừa ô cửa (tái dùng `ExtraCollider` của `bridge`)

Một body chữ nhật đặc **không** tả được tường có lỗ cửa. Bạn đã có sẵn khái niệm **`ExtraCollider`** trong behavior `bridge` — dùng lại đúng cơ chế đó để đặt **nhiều static body** thành viền tường, chừa khoảng trống cho cửa.

Ý tưởng: thêm behavior `house_walls` (hoặc dữ liệu `colliders: [...]` trong metadata) dựng 3–4 body (trái/phải/trên, và 2 đoạn dưới chừa cửa) theo footprint. Mỗi body là một static rect trong `collisionGroup`, offset theo anchor giống cách `bridge` đang làm. Không cần per-tile collision.

```jsonc
// ví dụ khai báo trong metadata nhà
"colliders": [
  { "x": 0,  "y": 0,  "w": 96, "h": 8  },   // tường trên
  { "x": 0,  "y": 8,  "w": 8,  "h": 64 },   // tường trái
  { "x": 88, "y": 8,  "w": 8,  "h": 64 },   // tường phải
  { "x": 0,  "y": 72, "w": 40, "h": 8  },   // đáy trái  (chừa cửa ở giữa)
  { "x": 56, "y": 72, "w": 40, "h": 8  }    // đáy phải
]
```

---

## 5. Mô hình dữ liệu

- **Đặt nhà** vẫn là **một dòng `map_placements`** (itemID nhà, một x/y). Multi-part chỉ là chuyện render phía client — DB không đổi.
- **Nội thất bên trong** = các placement **riêng** đặt trong footprint, gắn chủ sở hữu. Hai lựa chọn:
  - Cùng map, thêm cột `parent_placement_id` (FK tới placement nhà) để biết đồ này thuộc nhà nào → xoá nhà thì xoá cả cụm; quyền sửa nội thất = chủ nhà.
  - (Hướng B sau này) interior là `map_placements` trên `map_id` của map nội thất riêng, còn quyền sở hữu quản qua bảng `house_instances (owner, exterior_placement_id, interior_map_id)`.
- **Depth nội thất:** đồ trong nhà depth > sàn base nhưng < mái; vì mái fade khi người chơi vào nên vẫn thấy hết. Đây đúng là case same-cell-different-depth mà hệ stacking/depth đã xử lý.

---

## 6. Vì sao việc còn lại rất nhỏ (tái dùng thứ đã có)

- **Stacking 2 item/ô + depth fix**: base & roof là hai part cùng ô, khác `depth` — roof depth cao để vẽ đè lên người chơi khi họ đứng "dưới" mái.
- **Hệ behavior** (`applyBehaviorsOnCreate/OnUpdate`): chỉ thêm một handler ~15 dòng (`roof_fade`).
- **`ExtraCollider` của `bridge`**: dựng viền tường + ô cửa, không cần cơ chế collision mới.
- **Cơ chế tween fade**: bê nguyên từ `fadeBehind`.

Không bước nào cần vẽ asset mới hay animation.

---

## Checklist triển khai (Hướng A)

- [ ] Nâng Cute Fantasy lên bản trả phí (license commercial + có sẵn interior/floor/wall/furniture)
- [ ] Thêm `parts` (crop base/roof) + `footprint_w/h` vào metadata item nhà
- [ ] Mở rộng `renderPlacementsGroup` lặp qua `parts`; đổi khoá reconcile sang `placementId:part`
- [ ] Viết `roofFade.ts`, đăng ký `roof_fade` trong `BEHAVIORS`
- [ ] Dựng viền tường + ô cửa bằng `ExtraCollider` (behavior `house_walls` hoặc `colliders[]`)
- [ ] (Data) thêm `parent_placement_id` để gắn nội thất với nhà; quyền sửa = chủ nhà
- [ ] Đặt nội thất trong footprint bằng đúng hệ placement sẵn có
- [ ] (Sau) Hướng B cho nhà lớn: interior kit auto-gen + `mapCode` riêng qua `RoomManager.Actor()`
