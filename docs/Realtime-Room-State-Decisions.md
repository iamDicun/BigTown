# BigTown Realtime Room State Decisions
**Quyết định thiết kế RAM room state, movement validation và chống trùng vị trí nhân vật**

---

## 1. Bối cảnh hiện tại

Hiện tại realtime đã chạy bằng Centrifuge:

```text
Frontend
  -> WS /connection/websocket
  -> Centrifuge channel room:starter-town
  -> publish/subscribe player_chat
```

Nhưng backend hiện mới làm được:

- Auth connection bằng BigTown access token.
- Cho client subscribe/publish vào channel prefix `room:`.
- Broadcast event qua Centrifuge.

Backend **chưa có authoritative room state trong RAM**.

Nghĩa là hiện chưa có nơi lưu:

- Player nào đang online trong room.
- Player đang ở tọa độ nào.
- Player đang di chuyển hướng nào.
- NPC runtime nào đang sống/chết.
- Vị trí nào đang bị chiếm.

Do đó hiện tại nếu 2 client cùng gửi vị trí trùng nhau thì backend chưa chặn được.

---

## 2. Quyết định chính

BigTown sẽ thêm backend runtime state theo mô hình:

```text
Realtime module
  -> RoomStore interface
  -> MemoryRoomStore implementation cho MVP
  -> GameRoom trong RAM
```

Mục tiêu:

- Backend biết trạng thái hiện tại của từng room.
- Backend validate movement trước khi broadcast.
- Backend có thể chặn player overlap/trùng vị trí.
- Sau này có thể thay MemoryRoomStore bằng implementation khác mà không phá usecase.

---

## 3. Room state trong RAM

MVP cần state tối thiểu:

```text
GameRoom
├── RoomID
├── Players
│   ├── CharacterID
│   ├── ClientID
│   ├── X, Y
│   ├── Direction
│   ├── Moving
│   ├── LastSentAt / LastSeenAt
│   └── optional CurrentHP / WeaponID / Cooldown
└── NPCs
    ├── RuntimeID
    ├── NPCTypeID
    ├── X, Y
    ├── CurrentHP
    ├── Alive
    └── AIState
```

Go model dự kiến:

```go
type GameRoom struct {
    ID      string
    Players map[string]*RoomPlayer
    NPCs    map[string]*RoomNPC
}

type RoomPlayer struct {
    CharacterID string
    ClientID    string
    X           int
    Y           int
    Direction   Direction
    Moving      bool
    LastSeenAt  time.Time
}

type RoomNPC struct {
    RuntimeID string
    NPCTypeID string
    X         int
    Y         int
    CurrentHP int
    Alive     bool
    AIState   NPCState
}
```

---

## 4. RoomStore interface

Backend không nên để Hub/Centrifuge handler truy cập trực tiếp `map[string]*GameRoom`.

Cần bọc sau interface:

```go
type RoomStore interface {
    JoinRoom(ctx context.Context, roomID string, player RoomPlayer) (*RoomSnapshot, error)
    LeaveRoom(ctx context.Context, roomID string, characterID string) error
    GetSnapshot(ctx context.Context, roomID string) (*RoomSnapshot, error)
    MovePlayer(ctx context.Context, roomID string, characterID string, movement PlayerMovement) (*RoomPlayer, error)
    GetPlayer(ctx context.Context, roomID string, characterID string) (*RoomPlayer, error)
}
```

MVP implementation:

```text
MemoryRoomStore
├── mutex
└── rooms map[string]*GameRoom
```

Lý do dùng interface:

- Tránh phụ thuộc cứng vào RAM.
- Dễ test usecase.
- Sau này có thể chuyển sang room ownership/distributed store.

---

## 5. Chống trùng vị trí nhân vật

Có 2 kiểu movement có thể chọn.

### 5.1 Grid-based movement

Nếu player di chuyển theo ô tile:

```text
tileX = x / tileSize
tileY = y / tileSize
```

Rule chống trùng:

```text
Không cho 2 player cùng đứng trong cùng tile.
```

Validation:

```go
if targetTile occupied by another player {
    reject movement
}
```

Ưu điểm:

- Dễ validate.
- Dễ tránh overlap.
- Phù hợp gameplay tile/grid.

Nhược điểm:

- Movement ít tự nhiên hơn nếu muốn chạy tự do.

### 5.2 Pixel/free movement

Nếu player di chuyển tự do theo pixel:

Rule chống trùng:

```text
Không cho khoảng cách giữa 2 player nhỏ hơn minDistance.
```

Ví dụ:

```text
minDistance = 24px
```

Validation:

```go
distance(playerA, playerB) < minDistance => overlap
```

Ưu điểm:

- Movement tự nhiên hơn.
- Hợp với Phaser Arcade Physics.

Nhược điểm:

- Validate phức tạp hơn grid.
- Dễ cần xử lý correction/snap-back.

### Quyết định MVP

MVP nên chọn một trong hai hướng trước khi implement movement thật.

Khuyến nghị hiện tại:

```text
Map dùng tile 32x32.
Movement có thể render free/pixel trên FE.
Backend chống overlap bằng minDistance đơn giản.
```

Rule MVP:

```text
Nếu target position cách player khác < 24px thì reject movement hoặc trả correction event.
```

---

## 6. Server authoritative movement

Frontend vẫn render local movement ngay để cảm giác điều khiển tốt.

Nhưng backend phải là nơi quyết định movement có hợp lệ hay không.

Frontend gửi proposed movement:

```json
{
  "type": "player_move",
  "characterId": "...",
  "x": 120,
  "y": 240,
  "direction": "right",
  "moving": true
}
```

Backend kiểm tra:

- User có đúng là character đó không.
- Player có đang trong room đó không.
- Movement có quá nhanh không.
- Target position có ra ngoài map không.
- Target position có trùng/overlap player khác không.
- Sau này: target position có xuyên wall/collision không.

Nếu hợp lệ:

```text
Update RoomStore
Broadcast accepted player_move
```

Nếu không hợp lệ:

```text
Không update RoomStore
Gửi correction event cho client đó
```

Correction event ví dụ:

```json
{
  "type": "player_position_correction",
  "characterId": "...",
  "x": 96,
  "y": 224,
  "reason": "occupied"
}
```

---

## 7. Centrifuge publish hiện tại và vấn đề

Hiện tại backend đang dùng `client.OnPublish` theo kiểu:

```go
client.OnPublish(func(event centrifuge.PublishEvent, cb centrifuge.PublishCallback) {
    if !isRoomChannel(event.Channel) {
        cb(centrifuge.PublishReply{}, centrifuge.ErrorPermissionDenied)
        return
    }
    cb(centrifuge.PublishReply{}, nil)
})
```

Với cách này, client publish vào room channel và Centrifuge tự broadcast sau khi callback success.

Vấn đề nếu dùng cách này cho gameplay hoặc chat có persistence:

- Backend chưa validate nội dung event.
- Backend chưa update RoomStore trước broadcast.
- Client có thể publish vị trí bất kỳ.
- Chat chưa được lưu DB trước khi broadcast.
- Client mới vào room không lấy được lịch sử chat nếu chat chỉ là event ephemeral.

---

## 8. Quyết định xử lý Centrifuge cho gameplay

Để gameplay authoritative hơn, không nên để client publish trực tiếp event gameplay rồi Centrifuge auto-broadcast ngay.

Có 2 hướng.

### Hướng A: Validate trong `OnPublish`

Luồng:

```text
Client publish player_move
Backend OnPublish parse event
Backend validate/update RoomStore
Nếu hợp lệ thì cho publish
Nếu không hợp lệ thì reject
```

Ưu điểm:

- Ít thay đổi frontend.
- Vẫn dùng subscription publish API hiện tại.

Nhược điểm:

- Cần kiểm soát kỹ để không broadcast event chưa được chuẩn hóa.
- Correction/private response khó hơn.

### Hướng B: Client gửi command, server tự publish accepted event

Luồng:

```text
Client gửi command player_move tới server
Backend validate/update RoomStore
Backend node.Publish accepted event vào room channel
Nếu reject thì gửi correction/private event cho client
```

Ưu điểm:

- Sạch hơn cho authoritative gameplay.
- Chỉ server publish state đã được accepted.
- Dễ chuẩn hóa event output.
- Dễ gửi correction.

Nhược điểm:

- Cần đổi frontend từ `subscription.publish` sang command/RPC/message cho gameplay events.

### Quyết định khuyến nghị cho gameplay

Gameplay events như `player_move`, `enemy_hit` nên chuyển sang hướng B:

```text
Client sends command
Server validates
Server publishes accepted event
```

Nếu muốn làm nhanh trước, có thể tạm dùng hướng A cho `player_move`, nhưng cần ghi rõ đây là bước tạm.

---

## 9. Quyết định xử lý chat

Chat được chốt theo hướng **HTTP authoritative + DB history + Centrifuge broadcast**.

Không dùng client `subscription.publish` trực tiếp cho chat chính thức.

Luồng chat chính thức:

```text
Client gửi HTTP POST chat message
Backend validate nội dung và quyền gửi
Backend lưu DB vào chat_messages
Backend publish player_chat qua Centrifuge vào room channel
Tất cả clients đang subscribe room nhận event
Frontend hiển thị trong chat panel
Phaser hiển thị bubble trên đầu nhân vật
```

Lý do:

- Chat tần suất thấp, dùng HTTP là đủ.
- Cần lưu lịch sử chat để user mới vào room có thể thấy các tin gần nhất.
- Backend kiểm soát nội dung trước khi broadcast.
- Backend chuẩn hóa payload trước khi publish.
- Dễ thêm moderation/report/rate limit sau này.

### 9.1 Phạm vi chat MVP

MVP dùng **room chat**, không dùng global toàn server.

```text
room:starter-town -> chỉ user trong starter-town thấy
room:dungeon-1    -> chỉ user trong dungeon-1 thấy
```

Với MVP hiện tại, frontend mặc định dùng:

```text
room:starter-town
```

Tất cả người chơi trong cùng room sẽ thấy chat của nhau trong:

- Chat panel.
- Bubble trên đầu player.

### 9.2 Bảng `chat_messages`

Schema đề xuất:

```text
chat_messages
├── id
├── room_id
├── character_id
├── message
├── message_type
└── created_at
```

SQL gợi ý:

```sql
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id VARCHAR(120) NOT NULL,
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    message_type VARCHAR(30) NOT NULL DEFAULT 'text',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_chat_messages_room_created_at
    ON chat_messages(room_id, created_at DESC);
```

### 9.3 REST API chat

Lấy lịch sử chat gần nhất:

```text
GET /api/rooms/:roomId/chat/messages?limit=50
```

Gửi chat:

```text
POST /api/rooms/:roomId/chat/messages
```

Payload:

```json
{
  "message": "hello"
}
```

Backend xử lý:

```text
1. Auth user bằng BigTown JWT.
2. Resolve character_id từ user.
3. Validate room_id.
4. Trim message.
5. Check message không rỗng và không vượt max length.
6. Insert chat_messages.
7. Publish player_chat vào Centrifuge channel room:{roomId}.
8. Return saved message.
```

Event publish:

```json
{
  "type": "player_chat",
  "roomId": "starter-town",
  "characterId": "...",
  "displayName": "Alice",
  "message": "hello",
  "sentAt": "2026-07-17T...Z"
}
```

### 9.4 So sánh với movement

Chat và movement giống nhau ở nguyên tắc:

```text
Client không tự broadcast state cuối cùng.
Backend validate trước.
Backend publish accepted event.
```

Nhưng khác transport:

```text
Chat:
  HTTP POST -> validate -> save DB -> Centrifuge publish

Movement:
  Realtime command -> validate RoomStore RAM -> Centrifuge publish accepted movement

Position persistence:
  Debounced HTTP save -> DB
```

Không dùng HTTP cho từng movement tick vì movement tần suất cao và không ghi DB mỗi tick.

---

## 10. FE movement vẫn giữ throttled publishing

Frontend vẫn áp dụng solution đã thống nhất:

```text
Local movement event
-> update latestMovement
-> network threshold loop kiểm tra now - lastSentAt >= 100ms
-> gửi latestMovement nếu có
-> remote clients interpolate accepted event
```

### 10.1 Latest event / last sent handling

Movement sync ở frontend dùng 2 khái niệm riêng:

```text
latestMovement = movement event mới nhất đang chờ gửi
lastSentAt     = thời điểm lần cuối client gửi movement lên backend
```

Khi người chơi di chuyển, Phaser có thể tạo nhiều movement update nhỏ hơn threshold 100ms. Frontend không gửi tất cả các update đó. Frontend chỉ ghi đè `latestMovement` bằng event mới nhất.

Ví dụ threshold = 100ms:

```text
T=0ms    đã gửi event trước đó -> lastSentAt = 0
T=16ms   local move x=105 -> latestMovement = x=105, chưa gửi
T=32ms   local move x=110 -> latestMovement = x=110, chưa gửi
T=80ms   local move x=125 -> latestMovement = x=125, chưa gửi
T=100ms  now - lastSentAt = 100ms -> gửi latestMovement x=125
          lastSentAt = 100
          latestMovement = nil
```

Điểm quan trọng:

- Threshold so với `lastSentAt`, không so với thời điểm event mới nhất xảy ra.
- Event nhỏ hơn threshold không gửi ngay.
- Các event trung gian bị coalesced, chỉ giữ latest event.
- Nếu không có `latestMovement`, network loop không gửi gì dù đã qua 100ms.
- Khi người chơi dừng, gửi ngay final movement event `moving=false` để remote clients dừng animation.

Pseudo code:

```ts
let latestMovement: PlayerMoveEvent | null = null
let lastSentAt = 0
const movementThresholdMs = 100

function recordMovement(event: PlayerMoveEvent) {
  latestMovement = event
}

function movementNetworkTick(now: number) {
  if (!latestMovement) return

  if (now - lastSentAt >= movementThresholdMs) {
    sendMovementCommand(latestMovement)
    latestMovement = null
    lastSentAt = now
  }
}
```

Đây là **throttled latest-event publishing**, không phải debounce. Debounce chỉ dùng cho persistence vị trí cuối vào DB sau khi đứng yên.

Điểm thay đổi khi server authoritative:

- FE gửi proposed movement.
- Remote players chỉ render accepted movement từ server.
- Local player có thể render optimistic trước.
- Nếu backend gửi correction, local player snap/tween về vị trí accepted.

---

## 11. Persistence vị trí cuối

Không ghi DB mỗi movement tick.

DB chỉ lưu vị trí cuối khi:

- Player đứng yên sau debounce 2-3 giây.
- Player rời room.
- Player logout.
- Autosave định kỳ nếu cần.

Luồng:

```text
Realtime state: RoomStore RAM
Persistent state: characters.last_x, characters.last_y trong PostgreSQL
```

---

## 12. Thứ tự triển khai đề xuất

Khi bắt đầu implement movement thật, làm theo thứ tự:

1. Tạo `chat_messages` và module/chat API nếu muốn xử lý chat trước.
2. Đổi `ChatPanel` từ `subscription.publish` sang `POST /api/rooms/:roomId/chat/messages`.
3. Backend chat usecase lưu DB rồi publish `player_chat` qua Centrifuge.
4. Tạo `realtime/room/state.go`.
5. Tạo `realtime/room/store.go` interface.
6. Tạo `realtime/room/memory_store.go`.
7. Khi client connect/join room, tạo `RoomPlayer` trong `MemoryRoomStore`.
8. Khi disconnect/leave room, remove player khỏi room.
9. Parse `player_move` event/command.
10. Validate basic speed và overlap bằng `minDistance`.
11. Update RoomStore nếu hợp lệ.
12. Broadcast accepted `player_move`.
13. Gửi correction nếu không hợp lệ.
14. FE remote player interpolate accepted movement.
15. FE local player xử lý correction nếu có.
16. Thêm debounced position persistence vào DB.

---

## 13. Quyết định cần chốt trước khi code

Trước khi triển khai, cần chốt:

- Movement là grid-based hay pixel/free movement.
- Nếu pixel/free movement, `minDistance` là bao nhiêu px.
- Chat module/API làm trước hay movement RoomStore làm trước.
- Gameplay event dùng `OnPublish` tạm hay chuyển sang command/RPC ngay.
- Correction event gửi riêng cho client bằng cách nào.
- Room join lấy character position từ DB hay spawn point mặc định nếu chưa có position.

Khuyến nghị hiện tại:

```text
Movement: pixel/free movement.
minDistance: 24px.
Chat chính thức: HTTP POST -> save DB -> server publish Centrifuge.
Gameplay: chuyển dần sang server-authoritative command -> server publish accepted event.
RoomStore: MemoryRoomStore cho MVP.
Persistence: debounce save last position.
```
