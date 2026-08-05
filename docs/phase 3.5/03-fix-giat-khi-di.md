# Mục 3 — Fix "đi lòng vòng bị giật / giật ngược lại"

Đây là **3 vấn đề chồng lên nhau**, xử lý riêng từng cái. Làm A trước (dễ, tác động rõ nhất), rồi B, rồi C.

---

## A. Micro-stutter do âm thanh bước chân (nhiều khả năng là cái bạn thấy rõ nhất)

### Nguyên nhân
`src/shared/audio/audio.service.ts` — `playSfx()` tạo `new Audio(src)` **mỗi lần** phát. Bước chân bắn mỗi ~460 ms khi đang đi (`localPlayerController.ts`, `FOOTSTEP_INTERVAL_MS = 460`). Mỗi `new Audio()` là một `HTMLMediaElement` mới + decode MP3 trên main thread → khựng nhẹ đúng theo nhịp bước ("cứ đi chút là giật lại").

### Cách sửa: preload 1 lần rồi tái sử dụng qua sound manager (WebAudio) của Phaser

**Bước 1 —** nạp sẵn âm thanh trong `src/features/game/phaser/PreloadScene.ts`, thêm vào cuối `preload()`:
```ts
// Footsteps + click: nạp sẵn để phát qua WebAudio, không tạo HTMLAudio mỗi lần
for (let i = 1; i <= 7; i++) {
  this.load.audio(`footstep_${i}`, `/assets/sounds/f${i}.mp3`)
}
this.load.audio('sfx_click', '/assets/sounds/click.mp3')
```

**Bước 2 —** đổi cách phát bước chân trong `src/features/game/systems/localPlayerController.ts`.

Thêm import ở đầu file:
```ts
import { audioState } from '@/shared/audio/audio.service'
```

Sửa hằng số danh sách âm thanh thành danh sách **key** đã preload (bỏ mảng đường dẫn `FOOTSTEP_SOUNDS` cũ):
```ts
const FOOTSTEP_KEYS = ['footstep_1','footstep_2','footstep_3','footstep_4','footstep_5','footstep_6','footstep_7']
```

Thay hàm `playFootstepIfNeeded`:
```ts
private playFootstepIfNeeded(time: number, moving: boolean, speedMultiplier: number): void {
  if (!moving) return
  const interval = FOOTSTEP_INTERVAL_MS / speedMultiplier
  if (time - this.lastFootstepAt < interval) return

  this.lastFootstepAt = time
  const key = FOOTSTEP_KEYS[Math.floor(Math.random() * FOOTSTEP_KEYS.length)]
  // scene.sound dùng WebAudio: không cấp phát HTMLMediaElement, không decode lại.
  this.scene.sound.play(key, { volume: FOOTSTEP_VOLUME * audioState.sfxVolume.value })
}
```

> `scene.sound.play` tự quản lý pool nguồn phát, cho phép nhiều bước chân chồng nhau mà không tạo object mới mỗi lần.

**Bước 3 (tuỳ chọn) —** click SFX toàn cục trong `audio.service.ts` cũng đang `new Audio()` mỗi click. Nếu muốn triệt để, giữ một AudioBuffer decode sẵn và phát qua một `AudioContext` dùng chung. Nhưng click thưa hơn nhiều bước chân nên ưu tiên thấp — làm A/bước 1–2 trước.

---

## B. Giật đều do GC churn mỗi frame

### Nguyên nhân
`GameScene.update()` (mỗi frame) gọi:
- `editorSystem.update()` — duyệt **toàn bộ** placements **2 lần** (fade cây + glow đèn), mỗi child gọi `getBounds()` (cấp phát `Rectangle` mới).
- `editorSystem.isPlayerOnBridge()` — duyệt **toàn bộ** placements **lần nữa** + `getBounds()`.
- `player.getBounds()` cấp phát mỗi frame.
- `localPlayerController.onPostUpdate()` — `staminaGraphics.clear()` + vẽ lại mỗi frame kể cả khi stamina đầy.

Trên map nhiều decoration (winter/dark_village): mỗi frame cấp phát hàng loạt `Rectangle` → GC dồn → pause định kỳ = giật đều.

### Cách sửa: gộp thành 1 vòng duyệt, cache bounds, early-out

File `src/features/game/systems/editorSystem.ts`.

**B1 —** thêm field cache cạnh `isBehindDecoration`:
```ts
private isBehindDecoration = false
private isOnBridgeCached = false   // THÊM
```

**B2 —** gộp `update()` thành **một** vòng duyệt duy nhất, tính luôn cả bridge trong đó, và tính `player.getBounds()` một lần:
```ts
public update() {
  // ... giữ nguyên đoạn previewSprite ở đầu ...

  const player = this.playerSprite
  const playerBounds = player.getBounds()   // tính MỘT lần/frame
  let localBehindDecoration = false
  let localOnBridge = false

  const darkness = getDarkness()
  const time = this.scene.time.now
  const flicker = 0.92 + Math.sin(time * 0.008) * 0.05 + Math.sin(time * 0.021) * 0.03

  const children = this.placementsGroup?.active ? this.placementsGroup.getChildren() : []
  for (const child of children) {
    const sprite = child as Phaser.GameObjects.Image
    const itemCode = (sprite.getData('itemCode') as string) ?? ''

    // (1) Bridge — trước đây nằm ở isPlayerOnBridge(), giờ gộp vào đây
    if (itemCode.startsWith('deco_bridge_')) {
      if (Phaser.Geom.Intersects.RectangleToRectangle(playerBounds, sprite.getBounds())) {
        localOnBridge = true
      }
    }

    // (2) Fade cây khi player đứng sau
    if (itemCode.toLowerCase().includes('tree')) {
      const behind = player.y < sprite.y - 16 && player.y > sprite.y - sprite.height
      const isBehind = behind && Phaser.Geom.Intersects.RectangleToRectangle(playerBounds, sprite.getBounds())
      if (isBehind) localBehindDecoration = true

      const targetAlpha = isBehind ? 0.35 : 1.0
      if (sprite.getData('targetAlpha') !== targetAlpha) {
        sprite.setData('targetAlpha', targetAlpha)
        this.scene.tweens.killTweensOf(sprite)
        this.scene.tweens.add({ targets: sprite, alpha: targetAlpha, duration: 150, ease: 'Power1' })
      }
    }

    // (3) Glow đèn theo chu kỳ đêm
    const glow = sprite.getData('glow') as Phaser.GameObjects.Image
    if (glow) {
      glow.setAlpha(darkness * 0.8 * flicker)
      glow.setVisible(darkness > 0.05)
    }
  }

  this.isBehindDecoration = localBehindDecoration
  this.isOnBridgeCached = localOnBridge
}
```

**B3 —** `isPlayerOnBridge()` chỉ trả cache, KHÔNG duyệt lại:
```ts
public isPlayerOnBridge(): boolean {
  return this.isOnBridgeCached
}
```
Như vậy từ **3 vòng duyệt/frame** còn **1**, và `player.getBounds()` từ nhiều lần còn 1 lần.

**B4 —** trong `localPlayerController.ts`, `onPostUpdate()` chỉ vẽ khi stamina chưa đầy, và tránh clear vô ích:
```ts
private onPostUpdate(): void {
  if (!this.sprite || !this.sprite.active) return
  if (this.nameTag) updateNameTagPosition(this.nameTag, this.sprite)

  const needDraw = this.stamina < this.maxStamina
  if (!needDraw) {
    // chỉ clear MỘT lần khi vừa đầy lại, rồi thôi
    if (this.staminaGraphics && this.staminaGraphics.commandBuffer.length) {
      this.staminaGraphics.clear()
    }
    return
  }
  if (!this.staminaGraphics) this.staminaGraphics = this.scene.add.graphics()
  this.staminaGraphics.clear()
  // ... giữ nguyên đoạn vẽ vòng stamina ...
}
```

---

## C. Rubber-band thật (giật ngược vị trí)

### Nguyên nhân
Khi server từ chối một lệnh move, nó gửi `player_position_correction`; client `applyCorrection()` **snap cứng** `setPosition(x,y)`, không tween, không replay input. Server từ chối trong 2 tình huống dễ nổ oan (`backend/.../room_usecase.go` → `MovePlayer`):
- **`too_fast`**: dưới jitter mạng, các RPC bị throttle dồn lại thành burst → `elapsed` server tính ra rất nhỏ (bị clamp 0.01s) → `distance > 400*elapsed` → reject.
- **`occupied`**: đi sát người khác. Server dùng `minDistancePx = 24`, client chặn bằng `REMOTE_BLOCK_RADIUS = 26` (`remotePlayerManager.ts`) — lệch nhau nên client cho đi tới chỗ server coi là "đè".

### Cách sửa

**C1 — Correction mượt thay vì snap, và chỉ chỉnh khi lệch đáng kể.**
File `src/features/game/systems/localPlayerController.ts`:
```ts
applyCorrection(x: number, y: number): void {
  const dx = x - this.sprite.x
  const dy = y - this.sprite.y
  const dist = Math.hypot(dx, dy)

  // Lệch nhỏ (jitter bình thường): bỏ qua để không giật.
  if (dist < 6) {
    this.movementThrottle.latestMovement = null
    return
  }
  // Lệch lớn: snap để không bị teleport lố. Lệch vừa: tween ngắn cho mượt.
  if (dist > 64) {
    this.sprite.setPosition(x, y)
  } else {
    this.scene.tweens.add({
      targets: this.sprite, x, y, duration: 80, ease: 'Quad.easeOut',
    })
  }
  this.movementThrottle.latestMovement = null
}
```

**C2 — Đồng bộ bán kính va chạm client với server.**
`src/features/game/systems/remotePlayerManager.ts`: hạ `REMOTE_BLOCK_RADIUS` xuống khớp `minDistancePx = 24` phía server (hoặc thấp hơn chút để client luôn "chặn trước" server, tránh gửi lệnh mà server sẽ reject):
```ts
const REMOTE_BLOCK_RADIUS = 22  // <= server minDistancePx (24), client tự chặn trước
```

**C3 — Làm ngưỡng `too_fast` chịu jitter tốt hơn (server).**
`backend/internal/module/realtime/usecase/room_usecase.go`, hàm `MovePlayer`. Ngưỡng đang là `maxSpeedPxPerSec = 400` với `elapsed` clamp tối thiểu 0.01s. Khi burst, `elapsed` quá nhỏ làm mẫu số bé → dễ vượt. Nới sàn `elapsed` để không phạt oan burst hợp lệ:
```go
elapsed := time.Since(current.LastSeenAt).Seconds()
if elapsed < 0.08 {   // trước là 0.01 — nới sàn để chịu burst do throttle 100ms client
    elapsed = 0.08
}
```
Client di chuyển tối đa 150 px/s (sprint). `400 * 0.08 = 32 px` mỗi lần kiểm tra — vẫn chặn được cheat tốc độ thật (nhảy hàng trăm px) nhưng không phạt burst bình thường.

> Triệt để nhất là client-side prediction + reconciliation replay (client giữ hàng đợi input chưa được server xác nhận, khi nhận vị trí authoritative thì tua lại các input đó). Nhưng đó là refactor lớn; C1–C3 xử lý được phần lớn cảm giác giật ngược mà chi phí thấp.

---

## D. Cách kiểm chứng từng phần

**Đo giật (A + B):**
1. DevTools → tab **Performance** → record ~10s trong lúc đi vòng → tìm "long task" (thanh đỏ) và vạch GC (vàng). Sau khi sửa, các đỉnh scripting theo nhịp bước phải giảm hẳn.
2. Tab **Memory → Allocation instrumentation on timeline** → record khi đi → sẽ thấy trước khi sửa có sóng cấp phát `Rectangle` / `HTMLMediaElement` đều đặn; sau khi sửa gần như phẳng.
3. Thêm overlay FPS để thấy trực tiếp:
   ```ts
   // trong GameScene.create(), sau khi tạo camera:
   const fpsText = this.add.text(8, 8, '', { color: '#0f0', fontFamily: 'monospace' })
     .setScrollFactor(0).setDepth(99999)
   this.events.on('update', () => fpsText.setText(`FPS ${Math.round(this.game.loop.actualFps)}`))
   ```

**Đo rubber-band (C):**
1. Đếm correction phía client — trong callback `onCorrection` ở `GameScene.create()`:
   ```ts
   onCorrection: (event) => {
     console.count('correction')
     this.localPlayer.applyCorrection(event.x, event.y)
   },
   ```
   Đi thẳng một mình mà số `correction` vẫn tăng → là `too_fast` (xử lý bằng C3). Chỉ tăng khi gần người khác → là `occupied` (xử lý bằng C2).
2. Phía server, log lý do reject trong `MovePlayer` để phân loại chính xác:
   ```go
   // trước mỗi return &MovementRejection{...}
   log.Printf("move reject: reason=%s char=%s", "too_fast", character)
   ```
