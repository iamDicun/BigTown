# Realtime Room — Chuyển `MemoryRoomStore` sang mô hình Actor per-room

> Mục tiêu: bỏ **global `sync.Mutex`** đang serialize toàn bộ room, để nhiều room chạy song song thật trên nhiều CPU core, đúng triết lý *"share memory by communicating"* của Go.
> Ràng buộc: **không sửa `RoomUsecase`, không sửa transport**. Actor store chỉ cần implement đúng interface `room.RoomStore` hiện có (drop-in).
>
> Liên quan: `docs/Realtime-Room-State-Decisions.md`, `docs/Realtime-Performance-Techniques.md`.

---

## 1. Vấn đề hiện tại

`MemoryRoomStore` dùng **một `sync.Mutex` duy nhất** bảo vệ toàn bộ `rooms map[string]*GameRoom`:

```go
type MemoryRoomStore struct {
    mu    sync.Mutex   // 1 lock cho TẤT CẢ room
    rooms map[string]*GameRoom
}
```

Hệ quả khi tải tăng:

- Mỗi `player_move` RPC (~10 lần/giây/player đang di chuyển) đều `Lock()` cái mutex này. Player ở **room A** đang chặn player ở **room B** — hai room hoàn toàn độc lập về logic nhưng bị serialize về mặt thực thi.
- `RoomUsecase.MovePlayer` gọi **hai** lệnh store (`GetPlayerByUserID` rồi `GetSnapshot`), mỗi lệnh `Lock/Unlock` một lần → một movement chạm lock 3 lần (2 đọc + 1 `MovePlayer`), và giữa 2 lần lock đó state có thể đã đổi (snapshot đọc ra không nhất quán với thời điểm ghi).
- CPU 8 core cũng gần như vô dụng: mọi goroutine websocket xếp hàng sau đúng 1 lock. Đây là bottleneck chính, không phải số goroutine hay pool DB.

## 2. Ý tưởng: mỗi room là một goroutine sở hữu state của nó

Thay vì nhiều goroutine cùng chạm một `GameRoom` qua lock, ta cho **duy nhất một goroutine** (actor) sở hữu mỗi `GameRoom`. Mọi thao tác (join/leave/move/snapshot) được gửi tới actor đó dưới dạng **command qua channel**; actor xử lý tuần tự.

Lợi ích:

- **Không còn lock trên state room.** Chỉ một goroutine chạm `GameRoom` → không thể có data race, không cần `sync.Mutex` bên trong actor.
- **Song song thật giữa các room.** Room A và room B là 2 goroutine độc lập, chạy trên 2 core khác nhau cùng lúc.
- **Thao tác ghép được thành atomic.** Vì actor chạy tuần tự, "đọc snapshot + validate + ghi vị trí" nằm gọn trong một command → không còn khe hở race giữa các bước như hiện tại.
- **Sẵn sàng cho tick-based broadcast.** Actor có thể tự chạy `time.Ticker` 100ms, gom nhiều movement trong một tick rồi publish 1 lần thay vì publish mỗi RPC — giảm tải Centrifuge đáng kể (nâng cấp ở mục 8).

Lock duy nhất còn lại chỉ để **tra/ tạo actor theo `roomID`** (map `roomID → actor`), là thao tác cực ngắn và hiếm (chỉ lúc join/leave/tạo room), **không** nằm trên hot path movement.

## 3. Kiến trúc tổng thể

```
                 ┌────────────────────────────────────────────┐
   RoomUsecase   │           ActorRoomStore                    │
  (không đổi)    │  actors map[string]*roomActor  (RWMutex)    │
      │          │        │            │            │          │
      │  store.  │   ┌────▼───┐   ┌────▼───┐   ┌────▼───┐      │
      └─────────▶│   │ room A │   │ room B │   │ room C │ ...  │
   MovePlayer()  │   │ actor  │   │ actor  │   │ actor  │      │
                 │   │ (1 gor)│   │ (1 gor)│   │ (1 gor)│      │
                 │   └────────┘   └────────┘   └────────┘      │
                 └────────────────────────────────────────────┘
   RWMutex chỉ bảo vệ map "actors" (tra/tạo actor) — KHÔNG bảo vệ state bên trong room.
   State mỗi GameRoom do đúng 1 goroutine actor sở hữu → không cần lock.
```

- `ActorRoomStore` implement `room.RoomStore` → thay thẳng `NewMemoryRoomStore()` bằng `NewActorRoomStore()` trong `realtime/provider.go`, không đụng gì khác.
- Mỗi `roomActor` có một channel `cmds` nhận **closure** `func(*GameRoom)`; closure được actor gọi trong goroutine của nó, nên bên trong closure thao tác `GameRoom` **không cần lock**.
- Caller (store method) tạo một **reply channel có buffer 1**, gửi command, rồi chờ kết quả (có tôn trọng `ctx`).

## 4. Cơ chế command + reply (mẫu nền)

Dùng một kiểu command chung: closure chạy trong actor + tín hiệu `done`. Reply đi kèm qua channel do chính store method cấp phát (typed theo nhu cầu từng lệnh).

```go
// command là đơn vị công việc actor chạy tuần tự trên GameRoom của nó.
// fn được gọi BÊN TRONG goroutine actor => thao tác room không cần lock.
type command struct {
    fn func(room *GameRoom)
}
```

Mẫu một store method điển hình (ví dụ `MovePlayer`):

```go
func (s *ActorRoomStore) MovePlayer(
    ctx context.Context, roomID, characterID string, m room.PlayerMovement,
) (*room.RoomPlayer, error) {

    a := s.actorFor(roomID, false) // false = không tạo mới nếu chưa có
    if a == nil {
        return nil, room.ErrPlayerNotFound
    }

    // reply có buffer 1 => actor không bao giờ bị block khi gửi kết quả,
    // kể cả khi caller đã bỏ đi vì ctx hết hạn.
    type result struct {
        player *room.RoomPlayer
        err    error
    }
    reply := make(chan result, 1)

    cmd := command{fn: func(gr *GameRoom) {
        p, ok := gr.Players[characterID]
        if !ok {
            reply <- result{nil, room.ErrPlayerNotFound}
            return
        }
        p.X, p.Y = m.X, m.Y
        p.Direction, p.Moving = m.Direction, m.Moving
        p.LastSeenAt = time.Now()

        cp := *p // copy ra ngoài — không để caller giữ con trỏ vào state actor
        reply <- result{&cp, nil}
    }}

    // Gửi command; nếu ctx hết hạn/hủy trước khi gửi được thì thoát sớm.
    select {
    case a.cmds <- cmd:
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-a.quit:
        return nil, room.ErrPlayerNotFound
    }

    // Chờ kết quả, vẫn tôn trọng ctx.
    select {
    case r := <-reply:
        return r.player, r.err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

> **Quy tắc vàng:** con trỏ vào `GameRoom`/`RoomPlayer` **không bao giờ được rời khỏi goroutine actor**. Luôn `cp := *p` copy giá trị ra rồi trả copy — giống pattern `playerCopy := *player` mà `MemoryRoomStore` đang làm. Nếu để lọt con trỏ ra ngoài, ta lại có data race và mất toàn bộ lợi ích.

## 5. Skeleton đầy đủ — `actor_room_store.go`

File mới: `internal/module/realtime/room/actor_room_store.go`. Giữ nguyên `state.go`, `store.go`. Có thể để `memory_store.go` lại làm bản đối chiếu/fallback.

```go
package room

import (
    "context"
    "sync"
    "time"
)

var _ RoomStore = (*ActorRoomStore)(nil)

// ─────────────────────────────────────────────────────────────
// command: closure chạy tuần tự trong goroutine actor.
// ─────────────────────────────────────────────────────────────
type command struct {
    fn func(room *GameRoom)
}

// ─────────────────────────────────────────────────────────────
// roomActor: sở hữu độc quyền 1 GameRoom. Chỉ goroutine loop()
// được chạm state => không cần mutex bên trong.
// ─────────────────────────────────────────────────────────────
type roomActor struct {
    id    string
    state *GameRoom
    cmds  chan command
    quit  chan struct{}   // đóng khi actor dừng
    store *ActorRoomStore // để tự gỡ mình khỏi map khi rỗng (mục 7)
}

func newRoomActor(id string, store *ActorRoomStore) *roomActor {
    a := &roomActor{
        id: id,
        state: &GameRoom{
            ID:            id,
            Players:       make(map[string]*RoomPlayer),
            Clients:       make(map[string]map[string]struct{}),
            PlayersByUser: make(map[string]string),
        },
        cmds:  make(chan command, 64), // buffer nhỏ để hấp thụ burst RPC
        quit:  make(chan struct{}),
        store: store,
    }
    go a.loop()
    return a
}

// loop là TRÁI TIM của mô hình: xử lý command tuần tự, một tại một thời điểm.
func (a *roomActor) loop() {
    for {
        select {
        case cmd := <-a.cmds:
            cmd.fn(a.state) // an toàn: chỉ goroutine này chạm a.state
        case <-a.quit:
            return
        }
    }
}

// ─────────────────────────────────────────────────────────────
// ActorRoomStore: registry các actor. RWMutex CHỈ bảo vệ map,
// không bảo vệ state trong room.
// ─────────────────────────────────────────────────────────────
type ActorRoomStore struct {
    mu     sync.RWMutex
    actors map[string]*roomActor
}

func NewActorRoomStore() *ActorRoomStore {
    return &ActorRoomStore{actors: make(map[string]*roomActor)}
}

// actorFor tra actor theo roomID. create=true thì tạo mới nếu chưa có.
func (s *ActorRoomStore) actorFor(roomID string, create bool) *roomActor {
    // Fast path: đọc bằng RLock (nhiều reader song song).
    s.mu.RLock()
    a, ok := s.actors[roomID]
    s.mu.RUnlock()
    if ok {
        return a
    }
    if !create {
        return nil
    }

    // Slow path: cần Lock để tạo. Double-check vì có thể goroutine khác
    // vừa tạo xong giữa lúc ta nhả RLock và lấy Lock.
    s.mu.Lock()
    defer s.mu.Unlock()
    if a, ok := s.actors[roomID]; ok {
        return a
    }
    a = newRoomActor(roomID, s)
    s.actors[roomID] = a
    return a
}

// dispatch gửi 1 command tới actor, tôn trọng ctx và vòng đời actor.
func (a *roomActor) dispatch(ctx context.Context, fn func(*GameRoom)) error {
    select {
    case a.cmds <- command{fn: fn}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    case <-a.quit:
        return ErrPlayerNotFound
    }
}

// ─────────────────────────────────────────────────────────────
// Implement RoomStore. Mỗi method: lấy actor -> gửi closure -> chờ reply.
// ─────────────────────────────────────────────────────────────

func (s *ActorRoomStore) JoinRoom(
    ctx context.Context, roomID string, player RoomPlayer,
) (*RoomSnapshot, *RoomPlayer, bool, error) {

    a := s.actorFor(roomID, true) // join luôn tạo room nếu chưa có

    type result struct {
        snap    *RoomSnapshot
        joined  *RoomPlayer
        isFirst bool
    }
    reply := make(chan result, 1)

    if err := a.dispatch(ctx, func(gr *GameRoom) {
        // ── toàn bộ logic check-và-insert atomic của MemoryRoomStore.JoinRoom
        //    bê nguyên vào đây, KHÔNG cần lock ──
        if existing, ok := gr.Players[player.CharacterID]; ok {
            existing.LastSeenAt = time.Now()
            addClientLocked(gr, player.CharacterID, player.ClientID)
            setUserIndexLocked(gr, existing.UserID, player.CharacterID)
            cp := *existing
            reply <- result{snapshotLocked(gr), &cp, false}
            return
        }
        p := player
        p.LastSeenAt = time.Now()
        gr.Players[p.CharacterID] = &p
        addClientLocked(gr, p.CharacterID, p.ClientID)
        setUserIndexLocked(gr, p.UserID, p.CharacterID)
        cp := p
        reply <- result{snapshotLocked(gr), &cp, true}
    }); err != nil {
        return nil, nil, false, err
    }

    select {
    case r := <-reply:
        return r.snap, r.joined, r.isFirst, nil
    case <-ctx.Done():
        return nil, nil, false, ctx.Err()
    }
}

func (s *ActorRoomStore) LeaveRoom(
    ctx context.Context, roomID, characterID, clientID string,
) (*RoomPlayer, bool, error) {

    a := s.actorFor(roomID, false)
    if a == nil {
        return nil, false, nil
    }

    type result struct {
        player  *RoomPlayer
        removed bool
    }
    reply := make(chan result, 1)

    if err := a.dispatch(ctx, func(gr *GameRoom) {
        player, ok := gr.Players[characterID]
        if !ok {
            reply <- result{nil, false}
            return
        }
        if clients, ok := gr.Clients[characterID]; ok {
            delete(clients, clientID)
            if len(clients) > 0 { // còn connection khác giữ chỗ
                cp := *player
                reply <- result{&cp, false}
                return
            }
        }
        cp := *player
        delete(gr.Players, characterID)
        delete(gr.Clients, characterID)
        delete(gr.PlayersByUser, player.UserID)
        reply <- result{&cp, true}

        a.stopIfEmpty(gr) // GC actor khi room rỗng — xem mục 7
    }); err != nil {
        return nil, false, err
    }

    select {
    case r := <-reply:
        return r.player, r.removed, nil
    case <-ctx.Done():
        return nil, false, ctx.Err()
    }
}

func (s *ActorRoomStore) GetSnapshot(ctx context.Context, roomID string) (*RoomSnapshot, error) {
    a := s.actorFor(roomID, false)
    if a == nil {
        return &RoomSnapshot{RoomID: roomID}, nil
    }
    reply := make(chan *RoomSnapshot, 1)
    if err := a.dispatch(ctx, func(gr *GameRoom) {
        reply <- snapshotLocked(gr)
    }); err != nil {
        return nil, err
    }
    select {
    case r := <-reply:
        return r, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (s *ActorRoomStore) GetPlayer(ctx context.Context, roomID, characterID string) (*RoomPlayer, error) {
    return s.getPlayerBy(ctx, roomID, func(gr *GameRoom) (*RoomPlayer, bool) {
        p, ok := gr.Players[characterID]
        return p, ok
    })
}

func (s *ActorRoomStore) GetPlayerByUserID(ctx context.Context, roomID, userID string) (*RoomPlayer, error) {
    return s.getPlayerBy(ctx, roomID, func(gr *GameRoom) (*RoomPlayer, bool) {
        cid, ok := gr.PlayersByUser[userID]
        if !ok {
            return nil, false
        }
        p, ok := gr.Players[cid]
        return p, ok
    })
}

// getPlayerBy gom logic chung cho GetPlayer / GetPlayerByUserID.
func (s *ActorRoomStore) getPlayerBy(
    ctx context.Context, roomID string, pick func(*GameRoom) (*RoomPlayer, bool),
) (*RoomPlayer, error) {
    a := s.actorFor(roomID, false)
    if a == nil {
        return nil, ErrPlayerNotFound
    }
    reply := make(chan *RoomPlayer, 1)
    if err := a.dispatch(ctx, func(gr *GameRoom) {
        if p, ok := pick(gr); ok {
            cp := *p
            reply <- &cp
        } else {
            reply <- nil
        }
    }); err != nil {
        return nil, err
    }
    select {
    case p := <-reply:
        if p == nil {
            return nil, ErrPlayerNotFound
        }
        return p, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (s *ActorRoomStore) MovePlayer(
    ctx context.Context, roomID, characterID string, m PlayerMovement,
) (*RoomPlayer, error) {
    a := s.actorFor(roomID, false)
    if a == nil {
        return nil, ErrPlayerNotFound
    }
    reply := make(chan *RoomPlayer, 1)
    if err := a.dispatch(ctx, func(gr *GameRoom) {
        p, ok := gr.Players[characterID]
        if !ok {
            reply <- nil
            return
        }
        p.X, p.Y = m.X, m.Y
        p.Direction, p.Moving = m.Direction, m.Moving
        p.LastSeenAt = time.Now()
        cp := *p
        reply <- &cp
    }); err != nil {
        return nil, err
    }
    select {
    case p := <-reply:
        if p == nil {
            return nil, ErrPlayerNotFound
        }
        return p, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

> Các hàm helper `addClientLocked`, `setUserIndexLocked`, `snapshotLocked` trong `memory_store.go` **không thực sự dùng lock** (tên "Locked" chỉ là quy ước "phải gọi khi đang giữ lock"). Chúng thao tác thuần trên `*GameRoom` nên **tái sử dụng nguyên vẹn** trong actor. Cân nhắc đổi tên thành `addClientTo`, `setUserIndex`, `buildSnapshot` cho khỏi gây hiểu nhầm, nhưng không bắt buộc.

## 6. Điểm cắm vào code hiện có

Chỉ sửa **một dòng** ở `internal/module/realtime/provider.go` (nơi đang khởi tạo store):

```go
// store := room.NewMemoryRoomStore()
store := room.NewActorRoomStore()
```

`RoomUsecase`, `CentrifugeTransport`, routes, DTO — **không đổi**, vì tất cả chỉ phụ thuộc interface `room.RoomStore`. Đây là lý do lớp `port`/interface bạn đã dựng sẵn có giá trị.

## 7. Vòng đời actor & tránh goroutine leak (phần tinh tế nhất)

Mỗi actor là một goroutine sống mãi trong `loop()`. Nếu không bao giờ dừng, room rỗng vẫn giữ goroutine → rò rỉ dần. Có 2 lựa chọn:

**Lựa chọn A — Không GC (đơn giản, khuyến nghị cho MVP).**
Số map/room hữu hạn và nhỏ (theo `docs/Architecture.md` mục 9.1, MVP gần như 1 room/map). Một goroutine idle block trên `select` tốn vài KB, không đáng lo. Bỏ luôn `stopIfEmpty`. Đây là lựa chọn an toàn nhất để bắt đầu.

**Lựa chọn B — GC khi rỗng (cần cho hệ nhiều room động, ví dụ instance/dungeon).**
Khi `LeaveRoom` làm room hết player, actor tự gỡ mình khỏi registry rồi dừng:

```go
func (a *roomActor) stopIfEmpty(gr *GameRoom) {
    if len(gr.Players) > 0 {
        return
    }
    // Gỡ khỏi map (cần Lock của store) rồi đóng quit để loop() thoát.
    a.store.mu.Lock()
    // Double-check: chỉ xóa nếu map vẫn trỏ đúng actor này (tránh xóa nhầm
    // actor mới do một JoinRoom xen vào vừa tạo lại cùng roomID).
    if cur, ok := a.store.actors[a.id]; ok && cur == a {
        delete(a.store.actors, a.id)
    }
    a.store.mu.Unlock()
    close(a.quit)
}
```

Cạm bẫy cần biết ở lựa chọn B: có **race** giữa "actor đang tự dừng" và "một `JoinRoom` mới cùng `roomID` đang chờ trên `a.cmds`". Vì `stopIfEmpty` được gọi *bên trong* `loop()` (single-thread với các command khác của chính actor này), và việc xóa khỏi map dùng `store.mu`, một `actorFor(create:true)` đến sau sẽ hoặc thấy actor cũ (chưa kịp xóa) hoặc tạo actor mới — cả hai đều hợp lệ. Rủi ro thật sự là command đã nằm trong buffer `a.cmds` của actor sắp chết sẽ không được xử lý; xử lý bằng cách trong `dispatch` đã `select` trên `a.quit` để trả `ErrPlayerNotFound` và caller (usecase) retry/xử lý như "chưa join". **Khuyến nghị: chỉ bật B khi thực sự có room động; MVP dùng A.**

## 8. Nâng cấp về sau: tick-based broadcast (khi cần)

Hiện tại mỗi `player_move` publish ngay một event ra room channel. Với N player di chuyển liên tục, đó là N×10 publish/giây. Actor mở đường cho việc gom lại:

- Cho actor giữ thêm một `time.Ticker` (ví dụ 100ms) trong `loop()`:

```go
func (a *roomActor) loop() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case cmd := <-a.cmds:
            cmd.fn(a.state)
        case <-ticker.C:
            a.flushDirty() // gom mọi thay đổi từ tick trước, publish 1 lần
        case <-a.quit:
            return
        }
    }
}
```

- `MovePlayer` chỉ đánh dấu player "dirty" thay vì publish ngay; `flushDirty` gộp thành một `room_state` event và publish một lần mỗi tick.
- Việc này cần transport phơi ra một hàm publish để actor gọi (đảo chiều phụ thuộc — cân nhắc một `port.RoomPublisher`). **Không làm trong đợt migration này**; ghi lại đây như bước kế tiếp sau khi actor đã chạy ổn.

## 9. Kiểm thử & xác nhận

1. **Race detector** là tối quan trọng cho thay đổi concurrency:
   ```
   go test -race ./internal/module/realtime/...
   go run -race ./cmd/server   # chạy thử local với -race
   ```
   Viết một test cho `ActorRoomStore`: spawn ~100 goroutine cùng `JoinRoom`/`MovePlayer`/`LeaveRoom` trên vài `roomID`, assert không có race và số player cuối cùng đúng.

2. **Test tương đương hành vi:** vì `ActorRoomStore` và `MemoryRoomStore` cùng implement `RoomStore`, viết một bảng test chạy **cùng bộ ca** trên cả hai implementation (table-driven, tham số hóa constructor) để đảm bảo actor không đổi hành vi quan sát được.

3. **Đo trước/sau:** dưới tải giả lập nhiều room, so p99 latency của `player_move` và mức sử dụng CPU đa core. Kỳ vọng: throughput tăng gần tuyến tính theo số room (tới khi chạm trần core), không còn hiện tượng một room làm nghẽn room khác.

## 10. Tóm tắt checklist triển khai

- [ ] Thêm `internal/module/realtime/room/actor_room_store.go` (mục 5).
- [ ] (Tuỳ chọn) Đổi tên `*Locked` helper cho khỏi hiểu nhầm.
- [ ] Đổi `NewMemoryRoomStore()` → `NewActorRoomStore()` trong `realtime/provider.go` (mục 6).
- [ ] Chọn vòng đời actor: **A (không GC)** cho MVP, hoặc **B** nếu có room động (mục 7).
- [ ] Đảm bảo mọi con trỏ ra ngoài đều là **copy** (`cp := *p`).
- [ ] `go test -race` + test song song đa room (mục 9).
- [ ] (Sau) Cân nhắc tick-based broadcast (mục 8).
