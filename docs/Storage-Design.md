# BigTown Storage Design
**Thiết kế lưu trữ dữ liệu cho MVP game 2D multiplayer**

---

## 1. Mục tiêu

Tài liệu này mô tả cách tổ chức dữ liệu cho BigTown MVP ở 3 lớp:

- **PostgreSQL:** lưu dữ liệu bền vững, không mất khi server restart.
- **Go RAM:** lưu trạng thái runtime của phòng chơi/map, thay đổi liên tục theo thời gian thực.
- **Frontend:** lưu trạng thái phục vụ render, animation, nội suy và UI overlay.

Nguyên tắc chính: **database là nguồn sự thật cho tài sản và tiến trình dài hạn của người chơi; RAM là nguồn sự thật tạm thời cho phiên game đang chạy; frontend chỉ là lớp hiển thị và gửi input.**

---

## 2. Tổng quan dữ liệu

```text
PostgreSQL
├── users
├── characters
├── items
├── player_items
├── character_equipment
├── maps
├── npc_types
├── map_npc_spawns
└── reward_events

Go RAM
└── GameRoom
    ├── Players
    │   ├── position
    │   ├── direction
    │   ├── moving
    │   ├── equipped weapon snapshot
    │   ├── current HP
    │   └── cooldown
    └── NPCs
        ├── position
        ├── current HP
        ├── alive/dead
        ├── respawn time
        └── AI state

Frontend
├── local player sprite
├── remote player sprites
├── NPC sprites
├── movement interpolation
├── attack animation
├── HP bars
├── chat bubbles
└── UI overlays
```

---

## 3. PostgreSQL

PostgreSQL chỉ lưu các dữ liệu quan trọng cần khôi phục sau khi người chơi thoát game hoặc server restart.

Không ghi vào database mỗi lần người chơi di chuyển. Movement realtime nên giữ trong RAM và broadcast qua WebSocket.

### 3.1 `users`

Lưu tài khoản đăng nhập.

```text
id              uuid primary key
email           varchar unique not null
password_hash   text not null
created_at      timestamptz not null
updated_at      timestamptz not null
```

Ghi chú:

- `users` chỉ đại diện cho tài khoản.
- Thông tin gameplay nên nằm ở `characters`.

### 3.2 `characters`

Lưu nhân vật chính của người chơi.

```text
id              uuid primary key
user_id         uuid not null references users(id)
name            varchar not null
map_id          uuid references maps(id)
base_asset_key  varchar not null
coins           integer not null default 0
score           integer not null default 0
last_x          integer
last_y          integer
created_at      timestamptz not null
updated_at      timestamptz not null
```

Ghi chú:

- `coins` dùng để mua item.
- `score` dùng cho leaderboard.
- `last_x`, `last_y` không cần update mỗi 100ms. Chỉ update khi logout, rời room, đổi map hoặc autosave định kỳ.
- Nếu MVP chỉ cho mỗi user một nhân vật, đặt unique constraint trên `user_id`.

### 3.3 `items`

Lưu danh mục item/avatar asset có thể mua, sở hữu hoặc trang bị.

```text
id              uuid primary key
code            varchar unique not null
name            varchar not null
type            varchar not null
slot            varchar
asset_key       varchar not null
price           integer not null default 0
attack_bonus    integer not null default 0
hp_bonus        integer not null default 0
metadata_json   jsonb
created_at      timestamptz not null
updated_at      timestamptz not null
```

Ví dụ `type`:

- `body`
- `hair`
- `hat`
- `shirt`
- `weapon`
- `accessory`

Ví dụ `slot`:

- `base`
- `head`
- `body`
- `hand`
- `weapon`

Ghi chú:

- `asset_key` là key để frontend biết cần load sprite/texture nào.
- `metadata_json` dùng cho thông tin phụ như animation frame, rarity, description. Không nên lạm dụng để lưu dữ liệu cốt lõi.

### 3.4 `player_items`

Lưu inventory của nhân vật.

```text
id              uuid primary key
character_id    uuid not null references characters(id)
item_id         uuid not null references items(id)
quantity        integer not null default 1
acquired_at     timestamptz not null
```

Constraint đề xuất:

```text
unique(character_id, item_id)
check(quantity > 0)
```

Ghi chú:

- Với item không stack được, `quantity` luôn là `1`.
- Với coin/reward, không lưu coin như item nếu đã có `characters.coins`.

### 3.5 `character_equipment`

Lưu item đang được nhân vật trang bị.

```text
character_id    uuid not null references characters(id)
slot            varchar not null
item_id         uuid not null references items(id)
equipped_at     timestamptz not null
primary key(character_id, slot)
```

Ghi chú:

- Mỗi `slot` chỉ được equip một item tại một thời điểm.
- Khi equip, backend phải kiểm tra item đó có trong `player_items` hay không.
- RAM có thể giữ snapshot `equipped_weapon_id`, nhưng source of truth vẫn là bảng này.

### 3.6 `maps`

Lưu metadata map.

```text
id                  uuid primary key
code                varchar unique not null
name                varchar not null
tilemap_asset_key   varchar not null
tileset_asset_key   varchar not null
collision_asset_key varchar
spawn_x             integer not null
spawn_y             integer not null
width               integer not null
height              integer not null
created_at          timestamptz not null
updated_at          timestamptz not null
```

Ghi chú:

- `tilemap_asset_key` trỏ tới file map, ví dụ JSON export từ Tiled.
- `tileset_asset_key` trỏ tới ảnh tileset.
- Collision có thể nằm trong tilemap JSON; nếu tách riêng thì dùng `collision_asset_key`.

### 3.7 `npc_types`

Lưu định nghĩa loại NPC/enemy.

```text
id              uuid primary key
code            varchar unique not null
name            varchar not null
asset_key       varchar not null
max_hp          integer not null
attack          integer not null default 0
reward_score    integer not null default 0
reward_coin     integer not null default 0
respawn_ms      integer not null default 5000
metadata_json   jsonb
created_at      timestamptz not null
updated_at      timestamptz not null
```

Ghi chú:

- `npc_types` là template, không phải NPC runtime.
- HP hiện tại, alive/dead, AI state không lưu ở đây. Những giá trị đó nằm trong RAM.

### 3.8 `map_npc_spawns`

Lưu vị trí spawn NPC trên từng map.

```text
id              uuid primary key
map_id          uuid not null references maps(id)
npc_type_id     uuid not null references npc_types(id)
spawn_x         integer not null
spawn_y         integer not null
spawn_group     varchar
respawn_ms      integer
created_at      timestamptz not null
updated_at      timestamptz not null
```

Ghi chú:

- Bảng này trả lời câu hỏi: loại NPC nào sẽ xuất hiện ở map nào và tọa độ nào.
- Khi room khởi động, backend đọc bảng này để tạo danh sách NPC runtime trong RAM.

### 3.9 `reward_events`

Lưu lịch sử nhận điểm/tiền nếu cần audit hoặc debug.

```text
id              uuid primary key
character_id    uuid not null references characters(id)
event_type      varchar not null
npc_type_id     uuid references npc_types(id)
score_delta     integer not null default 0
coin_delta      integer not null default 0
metadata_json   jsonb
created_at      timestamptz not null
```

Ví dụ `event_type`:

- `kill_npc`
- `quest_reward`
- `admin_adjustment`
- `item_purchase`

Ghi chú:

- Leaderboard MVP có thể query trực tiếp từ `characters.score`.
- `reward_events` giúp biết vì sao điểm thay đổi.

---

## 4. Dữ liệu không nên lưu liên tục vào DB

Các dữ liệu sau thay đổi quá nhanh, nên giữ trong RAM và gửi qua WebSocket:

- Vị trí hiện tại của player trong từng frame/tick.
- Hướng di chuyển hiện tại.
- Trạng thái đang chạy/đứng yên.
- Cooldown đánh hiện tại.
- HP hiện tại của NPC runtime.
- Alive/dead của NPC runtime.
- AI state của NPC.
- Chat bubble đang hiển thị trên màn hình.

Có thể ghi snapshot xuống DB trong các trường hợp:

- Người chơi logout.
- Người chơi rời map.
- Server autosave mỗi 10-30 giây.
- Người chơi nhận điểm/coin.
- Người chơi mua/equip item.

---

## 5. Go RAM State

RAM state nằm trong server Golang, đại diện cho trạng thái realtime của một room/map đang chạy.

### 5.1 GameRoom

```go
type GameRoom struct {
    MapID   string
    Players map[string]*RoomPlayer
    NPCs    map[string]*RoomNPC
}
```

Ghi chú:

- Key của `Players` nên là `characterID`.
- Key của `NPCs` nên là `runtimeID`, không phải `npcTypeID`, vì cùng một loại NPC có thể spawn nhiều con.

### 5.2 RoomPlayer

```go
type RoomPlayer struct {
    CharacterID string
    ClientID    string

    X         int
    Y         int
    Direction Direction
    Moving    bool

    CurrentHP int
    WeaponID  *string

    AttackCooldownUntil time.Time
    LastSeenAt          time.Time
}
```

Ghi chú:

- `WeaponID` là snapshot từ `character_equipment` để tính combat nhanh.
- Khi người chơi đổi vũ khí, update DB rồi update snapshot trong RAM.
- `CurrentHP` của player có thể để RAM trước. Nếu sau này có hệ thống chết/hồi sinh nghiêm túc thì cân nhắc persistence.

### 5.3 RoomNPC

```go
type RoomNPC struct {
    RuntimeID string
    SpawnID   string
    NPCTypeID string

    X int
    Y int

    CurrentHP int
    Alive     bool
    AIState   NPCState
    RespawnAt *time.Time
}
```

Ghi chú:

- `SpawnID` map về `map_npc_spawns.id`.
- `NPCTypeID` map về `npc_types.id` để lấy max HP, reward, asset.
- `CurrentHP`, `Alive`, `AIState`, `RespawnAt` chỉ nằm trong RAM.

### 5.4 Cooldown

Cooldown nên nằm trong RAM vì đây là trạng thái realtime.

```go
type PlayerCooldowns struct {
    AttackUntil time.Time
    SkillUntil  time.Time
}
```

MVP chỉ cần `AttackCooldownUntil` là đủ.

---

## 6. Frontend State

Frontend không phải nguồn sự thật cho dữ liệu quan trọng. Frontend chỉ giữ dữ liệu để render nhanh và phản hồi mượt.

### 6.1 Local player

Frontend local player có thể render ngay khi người dùng bấm phím để cảm giác điều khiển nhanh.

```text
localPlayer
├── sprite
├── x, y
├── direction
├── moving
├── current animation
└── pending movement packet
```

Sau mỗi khoảng 100ms, client gửi movement packet lên server.

### 6.2 Remote players

Remote player chỉ nên cập nhật từ event server.

```text
remotePlayers
├── characterId
├── sprite
├── current position
├── target position
├── direction
├── moving
└── interpolation tween
```

Khi nhận `player_move`, frontend không cần nhảy sprite ngay tới vị trí mới. Thay vào đó, dùng interpolation/tween để kéo sprite từ vị trí hiện tại tới target trong khoảng 100ms.

### 6.3 NPC

NPC trên frontend là bản render của NPC runtime từ server.

```text
npcs
├── runtimeId
├── npcTypeId
├── sprite
├── x, y
├── currentHp
├── maxHp
├── alive
├── hpBar
└── animation state
```

Damage, chết và respawn nên theo event server:

- `npc_spawned`
- `npc_hit`
- `npc_killed`
- `npc_respawned`

### 6.4 UI overlay

Vue nên xử lý các UI overlay:

- Chat panel.
- Chat input.
- Leaderboard panel.
- Shop modal.
- Avatar builder.
- Inventory/equipment panel.

Phaser nên xử lý:

- Map rendering.
- Player/NPC sprite.
- Animation.
- Collision.
- Camera.
- HP bar nếu muốn gắn trực tiếp với sprite.
- Chat bubble trên đầu nhân vật.

---

## 7. Mapping giữa DB, RAM và Frontend

### 7.1 Character

```text
DB characters
  -> Go Character entity
  -> Go RoomPlayer khi vào room
  -> Frontend player sprite
```

Thông tin từ DB:

- `character_id`
- `name`
- `base_asset_key`
- `coins`
- `score`
- equipment hiện tại

Thông tin sinh ra trong RAM:

- `x`, `y` hiện tại
- `direction`
- `moving`
- cooldown

Thông tin render ở frontend:

- sprite texture key
- animation key
- chat bubble
- HP bar

### 7.2 Item

```text
DB items
  -> Go Item entity
  -> Inventory/equipment response
  -> Frontend asset key để render
```

Backend dùng item để:

- Kiểm tra người chơi có sở hữu item không.
- Kiểm tra item có đúng slot không.
- Tính chỉ số combat đơn giản như `attack_bonus`.

Frontend dùng item để:

- Hiển thị inventory.
- Hiển thị shop.
- Load đúng sprite/texture theo `asset_key`.

### 7.3 NPC

```text
DB npc_types + map_npc_spawns
  -> Go RoomNPC khi room khởi động
  -> Frontend NPC sprite khi nhận event spawn/sync
```

DB lưu:

- Loại NPC.
- Max HP.
- Reward.
- Asset.
- Điểm spawn.

RAM lưu:

- NPC runtime nào đang sống.
- HP hiện tại.
- AI state.
- Respawn timer.

Frontend render:

- Sprite NPC.
- HP bar.
- Hit/death animation.

---

## 8. Luồng dữ liệu chính

### 8.1 Vào game

```text
Client gọi API lấy profile
Backend đọc users, characters, inventory, equipment
Client mở WebSocket
Backend tạo RoomPlayer trong GameRoom
Backend gửi room snapshot về client
Backend broadcast player_joined cho người chơi khác
```

Room snapshot nên gồm:

- Thông tin player hiện tại.
- Danh sách remote players đang online.
- Danh sách NPC runtime.
- Map metadata.
- Equipment/avatar cần render.

### 8.2 Di chuyển

```text
Client local render movement ngay
Mỗi 100ms gửi player_move qua WebSocket
Server validate movement tối thiểu
Server update RoomPlayer trong RAM
Server broadcast player_move cho client khác
Remote clients dùng interpolation để render mượt
```

Không ghi DB cho từng packet movement.

### 8.3 Đánh NPC

```text
Client gửi enemy_hit {npcRuntimeId}
Server kiểm tra player tồn tại trong room
Server kiểm tra NPC còn sống
Server kiểm tra khoảng cách và cooldown
Server tính damage từ weapon snapshot/item config
Server trừ HP NPC trong RAM
Nếu NPC chết:
  - Cộng score/coin vào DB trong transaction
  - Ghi reward_events nếu cần
  - Update score/coin snapshot nếu có
  - Broadcast npc_killed và player_updated
Nếu NPC chưa chết:
  - Broadcast npc_hit
```

### 8.4 Mua item

```text
Client gọi REST API buy item
Backend kiểm tra coins trong characters
Backend kiểm tra item tồn tại
Backend transaction:
  - Trừ coins
  - Thêm player_items
  - Ghi reward_events/item_purchase nếu cần
Backend trả inventory mới
Frontend cập nhật shop/inventory UI
```

### 8.5 Equip item

```text
Client gọi REST API equip item
Backend kiểm tra item thuộc inventory
Backend kiểm tra slot hợp lệ
Backend upsert character_equipment
Nếu player đang online, update RoomPlayer snapshot
Backend broadcast player_updated để client khác đổi sprite/weapon render
```

### 8.6 Chat

```text
Client gửi player_chat qua WebSocket
Server validate độ dài/nội dung tối thiểu
Server broadcast player_chat tới room
Frontend hiển thị bubble trên đầu nhân vật và thêm vào chat panel
```

Chat MVP không cần lưu DB. Nếu sau này cần lịch sử chat, thêm bảng riêng.

---

## 9. Object gợi ý trong Go

### 9.1 Domain entity

```go
type Character struct {
    ID           string
    UserID       string
    Name         string
    MapID        string
    BaseAssetKey string
    Coins        int
    Score        int
    LastX        *int
    LastY        *int
}

type Item struct {
    ID          string
    Code        string
    Name        string
    Type        string
    Slot        string
    AssetKey    string
    Price       int
    AttackBonus int
    HPBonus     int
}

type NPCType struct {
    ID          string
    Code        string
    Name        string
    AssetKey    string
    MaxHP       int
    Attack      int
    RewardScore int
    RewardCoin  int
    RespawnMS   int
}

type MapNPCSpawn struct {
    ID        string
    MapID     string
    NPCTypeID string
    SpawnX    int
    SpawnY    int
    RespawnMS *int
}
```

### 9.2 Runtime model

```go
type Direction string

const (
    DirectionUp    Direction = "up"
    DirectionDown  Direction = "down"
    DirectionLeft  Direction = "left"
    DirectionRight Direction = "right"
)

type NPCState string

const (
    NPCStateIdle    NPCState = "idle"
    NPCStateChasing NPCState = "chasing"
    NPCStateDead    NPCState = "dead"
)

type RoomPlayer struct {
    CharacterID string
    ClientID    string
    X           int
    Y           int
    Direction   Direction
    Moving      bool
    CurrentHP   int
    WeaponID    *string
    AttackUntil time.Time
    LastSeenAt  time.Time
}

type RoomNPC struct {
    RuntimeID string
    SpawnID   string
    NPCTypeID string
    X         int
    Y         int
    CurrentHP int
    Alive     bool
    AIState   NPCState
    RespawnAt *time.Time
}
```

---

## 10. API và WebSocket event gợi ý

### REST API

```text
POST /api/auth/register
POST /api/auth/login
GET  /api/me

GET  /api/characters/me
GET  /api/items
GET  /api/inventory
POST /api/shop/buy
POST /api/equipment/equip

GET  /api/maps/:mapId
GET  /api/leaderboard
```

### WebSocket events

```text
client -> server
player_move
player_chat
enemy_hit

server -> client
room_snapshot
player_joined
player_left
player_move
player_chat
npc_hit
npc_killed
npc_respawned
player_updated
leaderboard_updated
error
```

---

## 11. Quy tắc consistency

Các thao tác thay đổi tài sản hoặc điểm phải đi qua backend và ghi DB trước khi thông báo cho client.

Nên dùng transaction cho các thao tác:

- Mua item: trừ coin và thêm inventory.
- Đánh chết NPC: cộng score/coin và ghi reward event.
- Equip item: kiểm tra ownership và cập nhật equipment.

Không tin dữ liệu quan trọng từ frontend:

- Không tin damage client gửi lên.
- Không tin reward client tự tính.
- Không tin item client nói đang sở hữu.
- Không tin score/coin client gửi lên.

Frontend chỉ gửi intention/input:

- Muốn di chuyển tới đâu.
- Muốn đánh NPC nào.
- Muốn gửi chat gì.
- Muốn mua/equip item nào.

Backend là nơi quyết định kết quả cuối cùng.

---

## 12. MVP Schema đề xuất cuối cùng

```text
users
characters
items
player_items
character_equipment
maps
npc_types
map_npc_spawns
reward_events
```

Chưa cần đưa vào MVP nếu chưa có gameplay cụ thể:

- `quests`
- `player_quests`
- `chat_messages`
- `guilds`
- `matches`
- `game_results`

Nếu sau này có quest thật, thêm `quests` và `player_quests`. Nếu sau này có trận đấu/session riêng, thêm `game_sessions` và `game_results`.
