# BigTown Phaser Frontend Guide
**Hướng dẫn triển khai map, nhân vật và realtime UI bằng Phaser**

---

## 1. Mục tiêu

Tài liệu này hướng dẫn cách triển khai phần frontend game của BigTown MVP bằng Phaser trong app Vue hiện tại.

Phạm vi gồm:

- Tổ chức code frontend giữa Vue và Phaser.
- Thiết kế map 2D pixel art.
- Load tileset/tilemap vào Phaser.
- Tạo nhân vật bằng sprite/spritesheet.
- Render NPC/enemy.
- Gửi movement bằng throttled publishing: tối đa mỗi 100ms nếu có movement mới.
- Nhận movement của người chơi khác và dùng interpolation để hiển thị mượt.
- Gắn chat bubble trên đầu nhân vật.
- Chuẩn bị luồng combat đơn giản.

Nguyên tắc quan trọng: **Phaser chịu trách nhiệm render/game loop; Vue chịu trách nhiệm app shell và UI overlay.**

---

## 2. Ranh giới Vue và Phaser

### Vue nên làm

- Routing.
- Login/register/auth state.
- Layout app.
- Chat panel dev hoặc chat input overlay.
- Leaderboard panel.
- Shop modal.
- Avatar builder.
- Inventory/equipment UI.
- Gọi REST API.
- Khởi tạo/hủy Phaser game instance.

### Phaser nên làm

- Render map.
- Render player/NPC sprites.
- Camera follow player.
- Animation idle/walk/attack/death.
- Collision với map.
- Game loop.
- Local player movement.
- Remote player interpolation.
- NPC HP bar nếu muốn gắn trực tiếp vào sprite.
- Chat bubble trên đầu nhân vật.

### Không nên làm

- Không để Vue component xử lý từng frame movement.
- Không để Phaser gọi REST API rải rác.
- Không để Phaser biết chi tiết auth/login.
- Không dùng chung một object cho DB entity, WebSocket event và Phaser sprite state.

---

## 3. Cấu trúc thư mục đề xuất

Frontend hiện đã có `features/game`. Tiếp tục phát triển theo cấu trúc này:

```text
frontend/src/features/game/
  views/
    GameView.vue
  components/
    GameCanvas.vue
    ChatPanel.vue
    LeaderboardPanel.vue
  phaser/
    createGame.ts
    BootScene.ts
    PreloadScene.ts
    GameScene.ts
  systems/
    movementSystem.ts
    interpolationSystem.ts
    chatBubbleSystem.ts
    enemySystem.ts
  network/
    gameSocket.ts
    gameEvents.ts
  stores/
    game.store.ts
  types/
    game.ts
```

Ý nghĩa:

- `GameCanvas.vue`: nơi mount Phaser vào DOM.
- `createGame.ts`: tạo `new Phaser.Game(...)`.
- `BootScene`: cấu hình ban đầu nếu cần.
- `PreloadScene`: load assets.
- `GameScene`: scene chính của map/gameplay.
- `systems`: tách các logic nhỏ khỏi scene để scene không phình quá nhanh.
- `network`: Centrifuge connection và event types.
- `stores`: Pinia state cần chia sẻ với Vue overlay.

---

## 4. Luồng tổng thể khi vào game

```text
User login local hoặc Teams SSO
Frontend có BigTown access_token
GameView render GameCanvas + overlay panels
GameCanvas tạo Phaser game instance
PreloadScene load tileset, tilemap, spritesheet, audio
GameScene tạo map, local player, camera, collision
GameScene mở realtime socket hoặc nhận socket từ Vue layer
Client subscribe room:starter-town
Server/room gửi snapshot ban đầu sau này
GameScene update local movement mỗi frame
MovementSyncSystem giữ latestMovement và publish tối đa mỗi 100ms nếu có thay đổi
Remote clients nhận player_move và interpolate sprite
```

Ghi chú:

- **Không gửi tọa độ mỗi frame.**
- Phaser vẫn chạy 60 FPS để render mượt.
- Network không gửi cứng mỗi 100ms. Nó chỉ publish nếu có latest movement event mới và đã qua threshold khoảng **100ms** từ lần gửi trước.
- Các movement event nhỏ hơn threshold bị gom lại, chỉ giữ event mới nhất.
- Remote player không nhảy vị trí tức thì, mà dùng interpolation/tween.

---

## 5. Thiết kế map

### Tool nên dùng

Nên dùng **Tiled Map Editor** để vẽ map.

Tiled cho phép:

- Import tileset pixel art.
- Vẽ tilemap theo grid.
- Tạo nhiều layer.
- Đánh dấu collision.
- Đặt object spawn point.
- Export JSON để Phaser load.

### Kích thước tile

Bộ asset hiện tại là pixel art 16x16. MVP nên thống nhất:

```text
Asset gốc: 16x16
Tile hiển thị trong game: 32x32
Scale: 2x
```

Lý do:

- Pixel art nhìn rõ hơn.
- Grid 32x32 dễ tính collision và movement.
- Nhân vật/NPC dễ canh theo tile.

### Layer map đề xuất

Trong Tiled, map nên có các layer:

```text
Ground
DecorationBelow
Collision
DecorationAbove
SpawnPoints
NPCSpawns
```

Ý nghĩa:

- `Ground`: cỏ, đường, đất, nước nền.
- `DecorationBelow`: vật trang trí nằm dưới player như bụi cỏ nhỏ.
- `Collision`: tile hoặc object không đi xuyên qua được.
- `DecorationAbove`: tán cây, mái nhà, vật cần render trên đầu player.
- `SpawnPoints`: điểm spawn player.
- `NPCSpawns`: điểm spawn NPC nếu muốn đọc từ map thay vì DB.

### Collision

Có 2 cách:

1. Collision bằng tile property.
2. Collision bằng object layer.

MVP nên dùng **tile property** nếu map đơn giản:

```text
Tile tree/water/rock có property collides = true
Phaser setCollisionByProperty({ collides: true })
```

Nếu sau này cần vùng collision phức tạp hơn, dùng object layer.

---

## 6. Đưa map vào Phaser

Luồng load map:

```text
PreloadScene
  load tilemap JSON
  load tileset image

GameScene
  create tilemap
  add tileset image
  create layers
  setup collision
  setup camera bounds
```

Ví dụ skeleton:

```ts
// PreloadScene.ts
import Phaser from 'phaser'

export class PreloadScene extends Phaser.Scene {
  constructor() {
    super('preload')
  }

  preload() {
    this.load.tilemapTiledJSON('starter-town-map', '/assets/maps/starter-town.json')
    this.load.image('cute-fantasy-tiles', '/assets/tiles/cute-fantasy-tiles.png')
    this.load.spritesheet('player', '/assets/player/player.png', {
      frameWidth: 16,
      frameHeight: 16,
    })
  }

  create() {
    this.scene.start('game')
  }
}
```

```ts
// GameScene.ts
import Phaser from 'phaser'

export class GameScene extends Phaser.Scene {
  private player!: Phaser.Physics.Arcade.Sprite
  private cursors!: Phaser.Types.Input.Keyboard.CursorKeys

  constructor() {
    super('game')
  }

  create() {
    const map = this.make.tilemap({ key: 'starter-town-map' })
    const tileset = map.addTilesetImage('CuteFantasyTiles', 'cute-fantasy-tiles')

    if (!tileset) throw new Error('Tileset not found')

    map.createLayer('Ground', tileset, 0, 0)
    const collisionLayer = map.createLayer('Collision', tileset, 0, 0)
    map.createLayer('DecorationAbove', tileset, 0, 0)

    collisionLayer?.setCollisionByProperty({ collides: true })

    this.player = this.physics.add.sprite(160, 160, 'player')
    this.player.setScale(2)

    if (collisionLayer) {
      this.physics.add.collider(this.player, collisionLayer)
    }

    this.cameras.main.startFollow(this.player)
    this.cameras.main.setBounds(0, 0, map.widthInPixels, map.heightInPixels)

    this.cursors = this.input.keyboard!.createCursorKeys()
  }
}
```

---

## 7. Thiết kế nhân vật

### Cách đơn giản cho MVP

Dùng spritesheet nhân vật hoàn chỉnh.

Ví dụ:

```text
player.png
  row 0: walk down
  row 1: walk left
  row 2: walk right
  row 3: walk up
```

MVP nên bắt đầu bằng:

- 1 spritesheet player.
- 1 spritesheet slime.
- 1 spritesheet skeleton.

Sau khi movement/realtime ổn mới làm avatar nhiều layer.

### Avatar nhiều phụ kiện

Có 2 hướng:

1. **Pre-composed spritesheet**
   - Mỗi avatar là một spritesheet hoàn chỉnh.
   - Dễ nhất.
   - Ít linh hoạt.

2. **Layered avatar**
   - Body, hair, shirt, weapon, hat là nhiều sprite chồng lên nhau.
   - Tất cả layer dùng cùng frame animation.
   - Linh hoạt nhưng phức tạp hơn.

MVP nên đi theo hướng 1 trước.

---

## 8. Animation nhân vật

Tạo animation trong `GameScene.create()` hoặc file helper riêng.

```ts
this.anims.create({
  key: 'player-walk-down',
  frames: this.anims.generateFrameNumbers('player', { start: 0, end: 3 }),
  frameRate: 8,
  repeat: -1,
})

this.anims.create({
  key: 'player-idle-down',
  frames: [{ key: 'player', frame: 0 }],
})
```

Khi di chuyển:

```ts
this.player.anims.play('player-walk-down', true)
```

Khi đứng yên:

```ts
this.player.anims.play('player-idle-down', true)
```

---

## 9. Local movement

Local player nên phản hồi ngay khi người dùng bấm phím. Đây là **client-side immediate rendering**.

```text
User bấm phím
Phaser update player velocity ngay
Camera follow ngay
Animation chạy ngay
MovementSyncSystem ghi nhận latestMovement
Network threshold loop publish latestMovement nếu now - lastSentAt >= 100ms
```

Ví dụ:

```ts
update(_time: number, delta: number) {
  const speed = 120
  this.player.setVelocity(0)

  if (this.cursors.left?.isDown) {
    this.player.setVelocityX(-speed)
    this.player.anims.play('player-walk-left', true)
  } else if (this.cursors.right?.isDown) {
    this.player.setVelocityX(speed)
    this.player.anims.play('player-walk-right', true)
  } else if (this.cursors.up?.isDown) {
    this.player.setVelocityY(-speed)
    this.player.anims.play('player-walk-up', true)
  } else if (this.cursors.down?.isDown) {
    this.player.setVelocityY(speed)
    this.player.anims.play('player-walk-down', true)
  } else {
    this.player.anims.play('player-idle-down', true)
  }
}
```

---

## 10. Throttled movement publishing

Architecture đã thống nhất: **client không bắn movement liên tục mỗi frame**.

Phaser render có thể chạy 60 FPS, nhưng network chỉ gửi khi đủ điều kiện:

```text
60 FPS render: khoảng 16.6ms/frame
Movement threshold: 100ms
Điều kiện gửi: latestMovement exists && now - lastSentAt >= threshold
```

Đây không phải debounce chuẩn. Đây là **throttle/network tick + latest-event coalescing**:

- Movement event local xảy ra liên tục khi người chơi giữ phím.
- FE chỉ lưu `latestMovement` mới nhất.
- Nếu event mới xảy ra nhỏ hơn threshold, nó ghi đè event cũ nhưng chưa gửi.
- Khi đủ threshold từ lần gửi trước, FE publish event mới nhất.
- Nếu không có movement mới, không gửi gì.

Ví dụ:

```ts
private lastMoveSentAt = 0
private latestMovement: PlayerMoveEvent | null = null
private readonly movementThresholdMs = 100

recordMovement(event: PlayerMoveEvent) {
  this.latestMovement = event
}

update(time: number) {
  this.updateLocalMovement()

  if (this.latestMovement && time - this.lastMoveSentAt >= this.movementThresholdMs) {

    this.gameSocket.send(this.latestMovement)
    this.latestMovement = null
    this.lastMoveSentAt = time
  }
}
```

Packet gửi đi:

```ts
{
  type: 'player_move',
  characterId: currentUserId,
  x: Math.round(this.player.x),
  y: Math.round(this.player.y),
  direction: 'down',
  moving: true,
}
```

Ghi chú:

- Chỉ gửi khi có movement event mới.
- Nếu nhiều movement event xảy ra trong 100ms, chỉ gửi event mới nhất.
- Khi thả phím, gửi ngay một packet cuối `moving: false` để remote dừng animation.
- Lưu vị trí cuối vào DB là luồng riêng: debounce 2-3 giây sau khi đứng yên, không ghi DB mỗi movement tick.
- Server/backend sau này phải validate position, speed, collision/cooldown ở mức cần thiết.

---

## 11. Remote player interpolation

Remote player là người chơi khác. Không điều khiển bằng keyboard local.

Luồng:

```text
Nhận player_move từ Centrifuge
Tìm remote sprite theo characterId
Nếu chưa có thì tạo sprite
Set target position = event.x/event.y
Tween/interpolate sprite từ vị trí hiện tại tới target trong 100ms
Play animation theo direction/moving
```

Ví dụ bằng Phaser tween:

```ts
function applyRemoteMove(sprite: Phaser.GameObjects.Sprite, event: PlayerMoveEvent) {
  this.tweens.killTweensOf(sprite)

  this.tweens.add({
    targets: sprite,
    x: event.x,
    y: event.y,
    duration: 100,
    ease: 'Linear',
  })

  if (event.moving) {
    sprite.anims.play(`player-walk-${event.direction}`, true)
  } else {
    sprite.anims.play(`player-idle-${event.direction}`, true)
  }
}
```

Lý do dùng interpolation:

- Giảm network traffic.
- Tránh remote player bị giật vì chỉ nhận packet khi FE publish theo threshold.
- Giữ cảm giác mượt mà dù backend không gửi mỗi frame.

---

## 12. Realtime event qua Centrifuge

Hiện frontend đã có:

```text
features/game/network/gameSocket.ts
features/game/network/gameEvents.ts
```

Channel mặc định:

```text
room:starter-town
```

Endpoint:

```text
ws://localhost:8080/connection/websocket
```

Token:

```text
BigTown access_token
```

Các event MVP:

```ts
type PlayerMoveEvent = {
  type: 'player_move'
  characterId: string
  x: number
  y: number
  direction: 'up' | 'down' | 'left' | 'right'
  moving: boolean
}

type PlayerChatEvent = {
  type: 'player_chat'
  characterId: string
  message: string
  sentAt: string
}

type EnemyHitEvent = {
  type: 'enemy_hit'
  npcRuntimeId: string
}
```

Hiện `ChatPanel` đã dùng realtime để test `player_chat`. Sau này `GameScene` sẽ dùng cùng socket/event channel để gửi `player_move`, `enemy_hit`.

---

## 13. Chat bubble trên đầu nhân vật

Chat panel là Vue overlay. Chat bubble trên đầu sprite nên nằm trong Phaser.

Luồng:

```text
Nhận player_chat event
Tìm player sprite theo characterId
Tạo Phaser Text hoặc Container phía trên sprite
Gắn bubble theo sprite trong update loop hoặc dùng container chung
Sau 2-3 giây destroy bubble
```

Ví dụ ý tưởng:

```ts
function showChatBubble(characterId: string, message: string) {
  const sprite = this.remotePlayers.get(characterId) ?? this.player
  if (!sprite) return

  const text = this.add.text(sprite.x, sprite.y - 32, message, {
    fontSize: '12px',
    color: '#ffffff',
    backgroundColor: '#000000aa',
    padding: { x: 6, y: 4 },
  })
  text.setOrigin(0.5, 1)

  this.time.delayedCall(2500, () => text.destroy())
}
```

Sau này nên có `chatBubbleSystem.ts` để quản lý bubble tốt hơn.

---

## 14. NPC/enemy MVP

MVP có thể spawn NPC cố định trong `GameScene.create()` trước, chưa cần server authoritative ngay.

Ví dụ:

```ts
const slime = this.physics.add.sprite(320, 220, 'slime')
slime.setScale(2)
```

Sau đó nâng cấp:

```text
Backend đọc npc_types + map_npc_spawns
Backend gửi room_snapshot có NPC runtime
Frontend render NPC theo snapshot
Client gửi enemy_hit
Backend validate khoảng cách/cooldown
Backend broadcast npc_hit/npc_killed
Frontend chạy animation hit/death
```

Không để frontend tự cộng điểm khi đánh NPC. Frontend chỉ gửi ý định `enemy_hit`.

---

## 15. Asset organization

Nên copy asset cần dùng vào frontend public folder để Phaser load bằng URL ổn định.

Đề xuất:

```text
frontend/public/assets/
  maps/
    starter-town.json
  tiles/
    cute-fantasy-tiles.png
  player/
    player.png
  enemies/
    slime.png
    skeleton.png
  audio/
    town-theme.mp3
```

Phaser load:

```ts
this.load.image('cute-fantasy-tiles', '/assets/tiles/cute-fantasy-tiles.png')
```

Không nên import image trực tiếp vào Phaser scene bằng Vite import ở giai đoạn đầu, vì public URL đơn giản hơn khi load Tiled JSON.

---

## 16. Audio

Preload:

```ts
this.load.audio('town-theme', '/assets/audio/town-theme.mp3')
```

Play:

```ts
const music = this.sound.add('town-theme', { loop: true, volume: 0.35 })
music.play()
```

Lưu ý browser thường chặn autoplay audio. Nên chỉ play nhạc sau khi user click vào game lần đầu.

---

## 17. Thứ tự triển khai đề xuất

Làm theo thứ tự này để ít rủi ro:

1. Tạo `GameCanvas.vue` mount Phaser game thật.
2. Tạo `createGame.ts`, `PreloadScene`, `GameScene`.
3. Load một map tĩnh hoặc tilemap JSON đơn giản.
4. Render local player sprite.
5. Điều khiển player bằng keyboard.
6. Thêm collision với map.
7. Tạo `MovementSyncSystem` để coalesce latest movement và publish tối đa mỗi 100ms khi có thay đổi.
8. Mở 2 tab, render remote player bằng event nhận được.
9. Thêm interpolation cho remote player.
10. Thêm chat bubble trong Phaser.
11. Render NPC tĩnh.
12. Thêm `enemy_hit` event.
13. Chuyển NPC spawn/damage/reward sang backend authoritative.
14. Thêm shop/avatar builder.
15. Sau khi browser ổn mới gắn Teams SDK auto-login.

---

## 18. Checklist kỹ thuật

Trước khi coi phần Phaser MVP là ổn, cần kiểm tra:

- Local player di chuyển mượt.
- Không gửi movement mỗi frame.
- Không gửi cứng mỗi 100ms khi không có movement mới.
- Movement event nhỏ hơn threshold được coalesced, chỉ gửi latest event.
- Movement packet gửi tối đa khoảng 100ms/lần khi đang có thay đổi.
- Remote player nhận event và interpolate mượt.
- Khi local player dừng, remote player cũng dừng animation.
- Reload tab không làm app crash.
- Mở 2 tab thấy nhau.
- Chat panel nhận/gửi được.
- Chat bubble hiện đúng trên đầu nhân vật.
- Collision không cho đi xuyên tường/nước/cây.
- Backend chưa tin reward/damage từ client.

---

## 19. Quy tắc giữ kiến trúc sạch

- `GameScene` không gọi DB/API trực tiếp nếu không cần.
- `GameScene` giao tiếp realtime qua `gameSocket` hoặc một adapter.
- `ChatPanel` có thể dùng realtime cho dev, nhưng chat bubble gameplay nên nằm trong Phaser.
- REST API cho inventory/shop/leaderboard nên nằm ở service riêng.
- Runtime state như position/cooldown/remote players không lưu Pinia trừ khi Vue UI cần đọc.
- Persistent state như inventory/equipment/score đọc từ API hoặc event server.
- Không để một file `GameScene.ts` chứa tất cả logic quá lâu; khi dài ra thì tách sang `systems`.
