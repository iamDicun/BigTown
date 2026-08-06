# Hướng dẫn: Animal System (idle-only) + Đặt con vật bằng Placement Item

Tài liệu này gộp 2 phần:

1. **AnimalSystem rút gọn** — con vật seed chỉ đứng idle, không di chuyển, không cần collision.
2. **Đặt con vật như placement item** — con vật đặt ra cũng idle, xoá được, đồng bộ realtime, tái dùng toàn bộ pipeline decoration sẵn có.

Phạm vi file đụng tới:

- `frontend/src/features/game/systems/animalSystem.ts` (thay toàn bộ)
- `frontend/src/features/game/systems/editorSystem.ts` (thêm 1 helper + sửa 1 đoạn tạo sprite)
- `backend/internal/database/seed.sql` (thêm item con vật)
- `frontend/src/features/game/phaser/GameScene.ts` — **không cần sửa**

---

## Bối cảnh & lý do chọn idle-only

Con vật seed hiện tại là `scene.add.sprite` thuần, mỗi frame bị `setPosition` tới toạ độ tính sẵn nên **không có physics body → không va chạm** với map, decoration, người chơi hay với nhau. Muốn nó di chuyển mà vẫn chặn tường/nước thì phải chuyển sang steering bằng velocity + collider (và đồng bộ vị trí qua server nếu multiplayer) — phức tạp.

Vì con vật ở đây chỉ mang tính trang trí, cho **idle-only** là gọn nhất: nó đứng đúng chỗ, không thể lội nước hay chồng vào nhà, nên bỏ được toàn bộ phần wander/flip/collision. Đồng thời cũng dứt điểm luôn 2 lỗi cũ ("đi lùi không quay đầu" do sprite vẽ quay trái nhưng code lật ngược, và "đứng yên" do random-walk tích luỹ bị clamp ghim vào góc).

---

## Phần 1 — AnimalSystem rút gọn (idle-only)

Thay **toàn bộ** nội dung `frontend/src/features/game/systems/animalSystem.ts` bằng:

```ts
import Phaser from 'phaser'
import type { NPCSpawnDto } from '../services/realtime.service'

interface AnimalMeta {
  frame_width: number
  frame_height: number
  columns: number
  row_idle: number
  idle_frame_rate: number
}

export class AnimalSystem {
  private scene: Phaser.Scene
  private sprites: Phaser.GameObjects.Sprite[] = []

  constructor(scene: Phaser.Scene) {
    this.scene = scene
  }

  spawnFromBootstrap(spawns: NPCSpawnDto[]): void {
    if (!spawns || spawns.length === 0) return

    // Gom asset chưa load, load 1 lần rồi tạo hết (không gọi load.start nhiều lần trong vòng lặp).
    const pending: NPCSpawnDto[] = []
    for (const s of spawns) {
      const meta = this.parseMeta(s.metadata_json)
      if (this.scene.textures.exists(s.asset_key)) {
        this.createAnimal(s.spawn_x, s.spawn_y, s.asset_key, meta)
      } else {
        this.scene.load.spritesheet(s.asset_key, `/assets/${s.asset_key}`, {
          frameWidth: meta.frame_width,
          frameHeight: meta.frame_height,
        })
        pending.push(s)
      }
    }

    if (pending.length > 0) {
      this.scene.load.once('complete', () => {
        for (const s of pending) {
          if (this.scene.textures.exists(s.asset_key)) {
            this.createAnimal(s.spawn_x, s.spawn_y, s.asset_key, this.parseMeta(s.metadata_json))
          }
        }
      })
      this.scene.load.start()
    }
  }

  private parseMeta(raw: string): AnimalMeta {
    try {
      const p = JSON.parse(raw)
      return {
        frame_width: p.frame_width || 32,
        frame_height: p.frame_height || 32,
        columns: p.columns || 2,
        row_idle: p.row_idle || 0,
        idle_frame_rate: p.idle_frame_rate || 4,
      }
    } catch {
      return { frame_width: 32, frame_height: 32, columns: 2, row_idle: 0, idle_frame_rate: 4 }
    }
  }

  private createAnimal(x: number, y: number, textureKey: string, meta: AnimalMeta): void {
    const animKey = textureKey.replace(/\//g, '_').replace(/\.png$/i, '')

    if (!this.scene.anims.exists(`${animKey}_idle`)) {
      const idleFrames = Array.from({ length: meta.columns }, (_, i) => meta.row_idle * meta.columns + i)
      this.scene.anims.create({
        key: `${animKey}_idle`,
        frames: this.scene.anims.generateFrameNumbers(textureKey, { frames: idleFrames }),
        frameRate: meta.idle_frame_rate,
        repeat: -1,
      })
    }

    const sprite = this.scene.add.sprite(x, y, textureKey, 0)
    sprite.setOrigin(0.5, 1.0)
    sprite.setDepth(2 + y / 10000.0)
    sprite.play(`${animKey}_idle`)
    this.sprites.push(sprite)
  }

  // Idle-only: không di chuyển nên không cần cập nhật mỗi frame.
  update(_time: number, _delta: number): void {}

  destroy(): void {
    for (const s of this.sprites) s.destroy()
    this.sprites.length = 0
  }
}
```

### Điểm cần biết

- `update()` để rỗng: animation idle có `repeat: -1` nên Phaser tự lặp, không cần cập nhật mỗi frame.
- `GameScene.ts` giữ nguyên — vẫn gọi `spawnFromBootstrap`, `update`, `destroy` như cũ.
- Không còn import `getGameTime`, không còn `detPosition`/wander/flip.
- `metadata_json` của `npc_types` trong seed vẫn dùng được nguyên (các key `wander_*`, `row_walk`, `walk_frame_rate` giờ bị bỏ qua, không sao).
- Depth base = 2 (nằm dưới người chơi) — giống hành vi cũ.

---

## Phần 2 — Đặt con vật bằng placement item

Ý tưởng: **không** đụng `AnimalSystem` cho con được đặt, mà cho nó đi qua đúng pipeline decoration trong `editorSystem.ts`. Nhờ đó con đặt tự động có preview, kiểm tra ô trống, `placementId`, click để xoá ở delete mode, và đồng bộ realtime — tất cả miễn phí. Khác biệt duy nhất với decoration thường: dùng `Sprite` + chạy anim idle thay vì `Image` tĩnh.

### Bước 1 — Seed item con vật

Thêm vào khối `INSERT INTO items (...) VALUES` trong `backend/internal/database/seed.sql`.
Lưu ý: dùng `frameWidth`/`frameHeight` (camelCase) để editor tự nhận và load dạng spritesheet; đặt `collides:false` để con vật không có va chạm.

```sql
('npc_chicken', 'Gà',  'decoration', 'animals/Chicken.png', 80,
  '{"is_animal":true,"frameWidth":32,"frameHeight":32,"columns":2,"row_idle":0,"idle_frame_rate":4,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
('npc_cow',     'Bò',  'decoration', 'animals/Cow.png',     120,
  '{"is_animal":true,"frameWidth":32,"frameHeight":32,"columns":2,"row_idle":0,"idle_frame_rate":4,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
('npc_pig',     'Heo', 'decoration', 'animals/Pig.png',     120,
  '{"is_animal":true,"frameWidth":32,"frameHeight":32,"columns":2,"row_idle":0,"idle_frame_rate":4,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
('npc_sheep',   'Cừu', 'decoration', 'animals/Sheep.png',   120,
  '{"is_animal":true,"frameWidth":32,"frameHeight":32,"columns":2,"row_idle":0,"idle_frame_rate":4,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
```

> Nhớ chèn trước dòng cuối của khối `VALUES` và giữ đúng dấu phẩy/`ON CONFLICT` như các item khác.

Các key trong metadata:

| Key | Ý nghĩa |
| --- | --- |
| `is_animal` | Cờ để editor biết đây là con vật (tạo Sprite + chạy anim idle) |
| `frameWidth` / `frameHeight` | Kích thước 1 frame; có 2 key này editor sẽ load asset dạng spritesheet |
| `columns` | Số cột trên sheet (số frame mỗi hàng) — dùng để tính frame idle |
| `row_idle` | Hàng chứa animation idle (0 = hàng đầu) |
| `idle_frame_rate` | Tốc độ khung hình idle |
| `anchorX` / `anchorY` | Điểm neo (0.5, 1.0 = giữa-dưới, đứng đúng trên ô) |
| `collides` | Để `false` → không va chạm (idle trang trí) |

### Bước 2 — Thêm helper `ensureIdleAnim` vào `EditorSystem`

Đặt cạnh các method private khác trong `editorSystem.ts`:

```ts
private ensureIdleAnim(animKey: string, textureKey: string, meta: any): void {
  if (this.scene.anims.exists(`${animKey}_idle`)) return
  const cols = meta.columns ?? 2
  const rowIdle = meta.row_idle ?? 0
  const frames = Array.from({ length: cols }, (_, i) => rowIdle * cols + i)
  this.scene.anims.create({
    key: `${animKey}_idle`,
    frames: this.scene.anims.generateFrameNumbers(textureKey, { frames }),
    frameRate: meta.idle_frame_rate ?? 4,
    repeat: -1,
  })
}
```

### Bước 3 — Sửa đoạn tạo sprite mới trong `renderPlacementsGroup`

Tìm đoạn `// Create new sprite` (khoảng dòng 318–324).

**Trước:**

```ts
// Create new sprite
let sprite: Phaser.GameObjects.Image
if (hasFrames) {
  sprite = this.scene.add.image(p.x, p.y, item.asset_key, frameIndex)
} else {
  sprite = this.scene.add.image(p.x, p.y, item.asset_key)
}
```

**Sau:**

```ts
// Create new sprite
let sprite: Phaser.GameObjects.Image
if (meta.is_animal) {
  const animKey = item.asset_key.replace(/\//g, '_').replace(/\.png$/i, '')
  this.ensureIdleAnim(animKey, item.asset_key, meta)
  const sp = this.scene.add.sprite(p.x, p.y, item.asset_key, 0)
  sp.play(`${animKey}_idle`)
  // Sprite có đủ mọi method của Image lúc runtime; cast để hết cảnh báo TS.
  sprite = sp as unknown as Phaser.GameObjects.Image
} else if (hasFrames) {
  sprite = this.scene.add.image(p.x, p.y, item.asset_key, frameIndex)
} else {
  sprite = this.scene.add.image(p.x, p.y, item.asset_key)
}
```

Không cần sửa gì thêm. Phần bên dưới (origin, depth, `placementId`, `setInteractive`, handler xoá, đồng bộ realtime) đều dùng chung biến `sprite`. Vì `collides:false`, đoạn tạo body va chạm tự bỏ qua.

---

## Ghi chú & tuỳ chọn

- **Preview lúc rê chuột**: hiện frame tĩnh (frame 0), không chạy anim — đúng ý, không phải sửa gì thêm.
- **Depth / che khuất**: con đặt để `collides:false` → depth base 2, luôn nằm dưới người chơi. Nếu muốn nó che/bị-che theo trục y như cây/nhà thì đổi `collides:true` (khi đó nó cũng sẽ **chặn** người chơi — cân nhắc theo ý đồ game).
- **Xoá con đã đặt**: hoạt động sẵn nhờ dùng chung pipeline — vào delete mode rồi click vào con vật.
- **Đồng bộ nhiều người**: con đặt lưu ở bảng placements nên tự đồng bộ và load lại đúng; đàn seed thì cố định theo `map_npc_spawns`.

## Checklist kiểm thử

- [ ] Đàn seed hiện lên và chạy anim idle (không trôi, không đứng "chạy tại chỗ").
- [ ] Con vật xuất hiện trong hotbar/inventory như item decoration.
- [ ] Đặt được con vật ra map, nó chạy anim idle.
- [ ] Người chơi đi xuyên qua con đặt (vì `collides:false`).
- [ ] Delete mode: click vào con đặt → xoá được.
- [ ] Reload lại map: con đặt vẫn còn đúng vị trí.
