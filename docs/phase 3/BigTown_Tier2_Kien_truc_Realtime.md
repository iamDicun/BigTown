# BigTown — Tier 2: Kiến trúc realtime kiểu game xây dựng online

> Tiếp nối `BigTown_Placement_Audit_va_Dong_bo.md`. Tài liệu này **giả định Tier 1 đã hoàn tất**
> (place/delete đồng bộ, coin guard atomic, persist-then-broadcast, validate server-side). Tier 2
> là bước nâng cấp khi cần **giảm độ trễ** và **chịu tải nhiều người thao tác liên tục**, bằng cách
> đưa DB ra khỏi hot path — đúng cách Minecraft & các game xây dựng online làm.

---

## 0. Vấn đề Tier 1 chưa giải, Tier 2 giải

Tier 1 đúng đắn nhưng **mỗi thao tác = 1 transaction DB đồng bộ** (`BeginTx → UPDATE coins → INSERT placement → INSERT event → Commit`). Khi N người đặt/xoá liên tục:
- Mỗi request chờ commit → độ trễ = f(tải DB), tăng phi tuyến khi contention.
- Tranh chấp ghi trên `characters` (coins) và `map_placements` (unique index) tạo lock contention ở Postgres.
- DB là điểm nghẽn trung tâm; scale server app không giúp được nhiều.

**Tier 2:** state sống trong RAM là chân lý, validate/broadcast trong RAM (µs), DB chỉ nhận **ghi theo lô, bất đồng bộ** (write-behind). Độ trễ người chơi ≈ 1 hop websocket, độc lập với tải DB.

---

## 1. Ánh xạ khái niệm: game xây dựng online → BigTown

| Khái niệm (Minecraft / game xây dựng) | Cơ chế | Ánh xạ sang BigTown |
|---|---|---|
| **World state trong RAM là chân lý** | Server giữ block state trong bộ nhớ, đĩa chỉ là snapshot | `MapState.occupied map[[2]int]*Placement` per-map; DB = backup |
| **Chunk (16×16)** — phân vùng không gian | Chia thế giới thành chunk để load/lock/save độc lập | Tuỳ chọn: chia map thành ô lưới chunk; **giai đoạn đầu 1 map = 1 đơn vị** là đủ |
| **Tick loop 20 TPS, xử lý tuần tự** | 1 luồng sim quyết định mọi thay đổi, có thứ tự | **Actor per-map**: 1 goroutine sở hữu state, nhận command qua channel → không lock, không race, có ordering |
| **Broadcast tới client quanh đó, ngay** | Gửi delta cho người chơi trong tầm nhìn | `RoomPublisher.PublishRoom` (Centrifuge) ngay sau khi mutate RAM |
| **Autosave định kỳ, chỉ chunk "dirty"** | Ghi region file `.mca` mỗi vài phút / khi unload | **Write-behind**: gom op dirty, flush theo lô mỗi ~1s + on-shutdown |
| **Region file gom 32×32 chunk** | Ghi nhiều chunk trong 1 file | Multi-row `INSERT` / `COPY`, `UPDATE` coins gộp trong 1 tx |
| **Client hiển thị block ngay, server sửa nếu sai** | Client-side prediction + server correction | Client render optimistic + `requestId`; server gửi event `placement_rejected` để rollback |
| **Zone/region server sở hữu entity người chơi khi resident** | Currency/inventory do zone server giữ khi online | **Wallet residency**: khi player join map, actor nạp coin vào RAM & là **người ghi coin duy nhất** cho player đó tới khi rời |
| **Area of Interest (AoI)** | Chỉ gửi update cho người trong bán kính | Tuỳ chọn scale: chỉ broadcast placement cho người gần vùng đó |

**Nguyên tắc vàng:** *DB không bao giờ nằm trên đường đi của một thao tác người chơi.* Nó chỉ được chạm tới bởi luồng nền write-behind và lúc nạp state.

---

## 2. Bức tranh tổng thể

```
                 HTTP handler (place/delete)
                        │  gửi Cmd + chờ reply (không chạm DB)
                        ▼
   ┌────────────────────────────────────────────────────────┐
   │  RoomManager: map[mapCode] -> *MapActor (lazy create)   │
   └────────────────────────────────────────────────────────┘
                        │
                        ▼
   ┌──────────── MapActor (1 goroutine / map) ──────────────┐
   │  state:  occupied map[[2]int]*Placement                │
   │          wallets map[charID]int   (coin authority)     │
   │  vòng lặp: for cmd := range cmds {  xử lý TUẦN TỰ  }   │
   │    1. validate trong RAM (occupancy, coin)  ── µs       │
   │    2. mutate RAM  (quyết định cuối cùng)                │
   │    3. reply <- kết quả  (handler trả HTTP ngay)         │
   │    4. outbound <- event (broadcast, không chặn loop)    │
   │    5. dirty   <- op    (write-behind, không chặn loop)  │
   └────────────────────────────────────────────────────────┘
             │ outbound                    │ dirty
             ▼                             ▼
   ┌──────────────────┐        ┌──────────────────────────┐
   │ Broadcaster       │        │ PersistQueue (write-behind)│
   │ (1 goroutine/map, │        │ gom lô, flush mỗi ~1s,     │
   │  giữ thứ tự) →    │        │ multi-row INSERT/UPSERT,   │
   │  Centrifuge       │        │ flush-on-shutdown          │
   └──────────────────┘        └──────────────────────────┘
             │                             │
             ▼                             ▼
      client (WS)                     Postgres (backup)
```

Ba luồng tách biệt cho mỗi map: **mutate** (actor, nhanh, quyết định), **broadcast** (mạng, có thể chậm, không được chặn mutate), **persist** (DB, theo lô). Tách ra để network/DB chậm không làm nghẽn quyết định.

---

## 3. Các thành phần (skeleton)

### 3.1 Command & kết quả

```go
package room

type CmdKind int
const ( CmdPlace CmdKind = iota; CmdDelete; CmdJoin; CmdLeave )

type Cmd struct {
    Kind    CmdKind
    CharID  string
    // place:
    Item    *entity.DecorationItem
    X, Y    int
    PlaceID string        // UUID sinh sẵn ở handler → idempotent
    // delete:
    TargetID string
    // join: nạp ví
    Coins   int           // coin đọc từ DB lúc join (residency)
    Reply   chan CmdResult
}

type CmdResult struct {
    Placement *entity.Placement
    NewCoins  int
    Err       error
}

var (
    ErrOccupied          = errors.New("ô đã có vật thể")
    ErrInsufficientCoins = errors.New("không đủ coins")
    ErrNotOwner          = errors.New("không có quyền xoá")
    ErrNotFound          = errors.New("vật thể không tồn tại")
    ErrBusy              = errors.New("hệ thống đang bận")
)
```

### 3.2 MapActor — "tick loop" xử lý tuần tự

```go
type MapActor struct {
    mapID, mapCode string
    tileSize       int
    mapW, mapH     int

    occupied map[[2]int]*entity.Placement // chân lý về vị trí
    byID     map[string]*entity.Placement // tra cứu nhanh khi xoá
    wallets  map[string]int               // chân lý coin cho player resident
    residents map[string]int              // charID -> refcount (join/leave)

    cmds     chan Cmd        // lệnh vào (buffered lớn, vd 4096)
    outbound chan any        // event broadcast (giữ thứ tự)
    dirty    chan persistOp  // ghi nền
    items    port.ItemReader // để tra giá khi cần (thường đã có ở Cmd)
}

func (m *MapActor) run() {
    for c := range m.cmds {
        switch c.Kind {
        case CmdPlace:  m.handlePlace(c)
        case CmdDelete: m.handleDelete(c)
        case CmdJoin:   m.wallets[c.CharID] = c.Coins; m.residents[c.CharID]++
        case CmdLeave:
            if m.residents[c.CharID]--; m.residents[c.CharID] <= 0 {
                m.dirty <- persistOp{Kind: opFlushWallet, CharID: c.CharID, Coins: m.wallets[c.CharID]}
                delete(m.residents, c.CharID); delete(m.wallets, c.CharID)
            }
        }
    }
}

func (m *MapActor) handlePlace(c Cmd) {
    // validate hình học (Tier 1 đã có, nay làm trong RAM)
    if c.X%m.tileSize != 0 || c.Y%m.tileSize != 0 ||
        c.X < 0 || c.Y < 0 || c.X >= m.mapW || c.Y >= m.mapH {
        c.Reply <- CmdResult{Err: errors.New("toạ độ không hợp lệ")}; return
    }
    key := [2]int{c.X, c.Y}
    if _, taken := m.occupied[key]; taken {
        c.Reply <- CmdResult{Err: ErrOccupied}; return
    }
    coins, ok := m.wallets[c.CharID]
    if !ok { coins = m.lazyLoadWallet(c.CharID) } // fallback nếu chưa resident
    if coins < c.Item.Price {
        c.Reply <- CmdResult{Err: ErrInsufficientCoins}; return
    }

    // ---- MUTATE RAM = quyết định cuối cùng ----
    m.wallets[c.CharID] = coins - c.Item.Price
    p := &entity.Placement{
        ID: c.PlaceID, MapID: m.mapID, CharacterID: c.CharID,
        ItemID: c.Item.ID, X: c.X, Y: c.Y, CreatedAt: time.Now(),
    }
    m.occupied[key] = p
    m.byID[p.ID] = p

    // trả HTTP NGAY (không chờ broadcast / DB)
    c.Reply <- CmdResult{Placement: p, NewCoins: m.wallets[c.CharID]}

    // broadcast + persist: offload, không chặn loop
    m.outbound <- map[string]any{"type": "decoration_placed", "placement": p}
    m.dirty <- persistOp{
        Kind: opPlace, P: p, CharID: c.CharID,
        CoinDelta: -c.Item.Price, NewCoins: m.wallets[c.CharID], EventType: "decoration_place",
    }
}

func (m *MapActor) handleDelete(c Cmd) {
    p, ok := m.byID[c.TargetID]
    if !ok { c.Reply <- CmdResult{Err: ErrNotFound}; return }
    if p.CharacterID != c.CharID { c.Reply <- CmdResult{Err: ErrNotOwner}; return }

    price := c.Item.Price // handler đã tra item theo p.ItemID
    coins := m.wallets[c.CharID]
    if _, resident := m.wallets[c.CharID]; !resident { coins = m.lazyLoadWallet(c.CharID) }

    // MUTATE RAM
    m.wallets[c.CharID] = coins + price
    delete(m.occupied, [2]int{p.X, p.Y})
    delete(m.byID, p.ID)

    c.Reply <- CmdResult{NewCoins: m.wallets[c.CharID]} // trả HTTP ngay

    m.outbound <- map[string]any{"type": "decoration_deleted", "placementId": p.ID}
    m.dirty <- persistOp{
        Kind: opDelete, P: p, CharID: c.CharID,
        CoinDelta: +price, NewCoins: m.wallets[c.CharID], EventType: "decoration_refund",
    }
}
```

> Vì actor xử lý **tuần tự**, hai lệnh đặt cùng ô sẽ vào loop lần lượt: lệnh đầu chiếm `occupied[key]`,
> lệnh sau thấy `taken` → `ErrOccupied`. **Không cần** unique-index-nổ-async như Tier 1, không TOCTOU coin,
> không double-refund — tất cả tan biến vì đã có 1 điểm quyết định tuần tự.

### 3.3 Broadcaster — giữ thứ tự, không chặn actor

```go
func (m *MapActor) broadcastLoop(pub port.RoomPublisher) {
    for ev := range m.outbound {
        // publish có thể chậm (mạng) nhưng không làm nghẽn vòng mutate
        if err := pub.PublishRoom(context.Background(), m.mapCode, ev); err != nil {
            log.Printf("[room %s] publish err: %v", m.mapCode, err)
        }
    }
}
```

Một goroutine đọc `outbound` tuần tự ⇒ thứ tự event trên channel Centrifuge được giữ nguyên (khớp thứ tự mutate).

### 3.4 Wallet residency — coin authority

**Bất biến (invariant):** *khi một character đang resident trong 1 MapActor, actor đó là NGƯỜI GHI COIN DUY NHẤT cho character.*

- **Join** (móc vào realtime `player_joined`): đọc coin từ DB 1 lần → gửi `CmdJoin{Coins}` → actor cache.
- **Trong lúc chơi:** mọi thay đổi coin (đặt/xoá, và cả **phần thưởng combat / leaderboard**) phải đi qua actor bằng command (vd `CmdCredit{amount}`), **không module nào ghi thẳng `characters.coins`**.
- **Leave** (`player_left`): flush coin cuối cùng ra DB rồi evict.

> Đây chính là mô hình MMO: zone/region server sở hữu entity người chơi (kể cả ví/inventory) khi họ ở trong zone. Nếu chưa muốn refactor toàn bộ hệ thưởng, dùng `lazyLoadWallet` (đọc DB lần đầu, cache sau) và đặt lịch chuyển dần các nguồn coin khác sang command.

```go
func (m *MapActor) lazyLoadWallet(charID string) int {
    coins, _ := m.charReader.GetCoins(context.Background(), charID) // 1 lần, ngoài hot path sau đó
    m.wallets[charID] = coins
    return coins
}
```

### 3.5 PersistQueue — write-behind, idempotent

```go
type opKind int
const ( opPlace opKind = iota; opDelete; opFlushWallet )

type persistOp struct {
    Kind      opKind
    P         *entity.Placement
    CharID    string
    NewCoins  int    // coin tuyệt đối sau mutate (last-write-wins, idempotent)
    CoinDelta int    // để ghi reward_events (audit)
    EventType string
}

type Writer struct {
    db    *sql.DB
    in    chan persistOp
    done  chan struct{}
    every time.Duration // vd 1s
    max   int           // vd 512
}

func (w *Writer) loop() {
    t := time.NewTicker(w.every); defer t.Stop()
    batch := make([]persistOp, 0, w.max)
    for {
        select {
        case op := <-w.in:
            batch = append(batch, op)
            if len(batch) >= w.max { w.flush(batch); batch = batch[:0] }
        case <-t.C:
            if len(batch) > 0 { w.flush(batch); batch = batch[:0] }
        case <-w.done:
            for { // drain hết trước khi thoát (P4: flush-on-shutdown)
                select {
                case op := <-w.in: batch = append(batch, op)
                default: w.flush(batch); return
                }
            }
        }
    }
}

func (w *Writer) flush(batch []persistOp) {
    if len(batch) == 0 { return }
    tx, err := w.db.BeginTx(context.Background(), nil)
    if err != nil { log.Printf("[writer] begin: %v", err); return }
    defer tx.Rollback()

    // 1) coalesce coin theo char (last-write-wins) → 1 UPDATE/char
    latestCoins := map[string]int{}
    for _, op := range batch {
        if op.CharID != "" { latestCoins[op.CharID] = op.NewCoins }
    }
    for charID, coins := range latestCoins {
        if _, err := tx.Exec(`UPDATE characters SET coins=$1 WHERE id=$2`, coins, charID); err != nil {
            log.Printf("[writer] coins: %v", err); return
        }
    }

    // 2) placements: INSERT idempotent + DELETE gộp
    for _, op := range batch {
        switch op.Kind {
        case opPlace:
            _, err = tx.Exec(
                `INSERT INTO map_placements (id,map_id,character_id,item_id,x,y)
                 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING`,
                op.P.ID, op.P.MapID, op.P.CharacterID, op.P.ItemID, op.P.X, op.P.Y)
        case opDelete:
            _, err = tx.Exec(`DELETE FROM map_placements WHERE id=$1`, op.P.ID)
        }
        if err != nil { log.Printf("[writer] placement: %v", err); return }
    }

    // 3) reward_events: append log (audit) — INSERT nhiều dòng
    for _, op := range batch {
        if op.EventType == "" { continue }
        if _, err = tx.Exec(
            `INSERT INTO reward_events (character_id,event_type,coin_delta) VALUES ($1,$2,$3)`,
            op.CharID, op.EventType, op.CoinDelta); err != nil {
            log.Printf("[writer] event: %v", err); return
        }
    }

    if err := tx.Commit(); err != nil { log.Printf("[writer] commit: %v", err) }
}
```

Điểm mấu chốt để **idempotent** (an toàn khi replay / batch trùng):
- Placement dùng UUID sinh sẵn + `ON CONFLICT (id) DO NOTHING`.
- Coin ghi **giá trị tuyệt đối** (không cộng dồn delta) ⇒ ghi lại nhiều lần vẫn đúng, và coalesce được.
- `reward_events` là log append (chấp nhận mất vài dòng khi crash như autosave; nếu cần chính xác tuyệt đối → xem mục 7 WAL).

### 3.6 RoomManager + lifecycle

```go
type RoomManager struct {
    mu     sync.RWMutex
    actors map[string]*MapActor
    deps   Deps // db, publisher, repo, charReader, writer...
}

func (rm *RoomManager) actor(mapCode string) (*MapActor, error) {
    rm.mu.RLock(); a := rm.actors[mapCode]; rm.mu.RUnlock()
    if a != nil { return a, nil }

    rm.mu.Lock(); defer rm.mu.Unlock()
    if a = rm.actors[mapCode]; a != nil { return a, nil }

    a = newMapActor(mapCode, rm.deps)
    if err := a.loadFromDB(); err != nil { return nil, err } // nạp placements hiện có
    go a.run()
    go a.broadcastLoop(rm.deps.Publisher)
    rm.actors[mapCode] = a
    return a, nil
}

// gọi khi shutdown: đóng cmds của mọi actor, rồi Writer.Close() sẽ drain & flush
func (rm *RoomManager) Shutdown() {
    rm.mu.Lock(); defer rm.mu.Unlock()
    for _, a := range rm.actors { close(a.cmds) }
    rm.deps.Writer.Close() // đóng done → flush nốt batch
}
```

Usecase mới chỉ còn là lớp mỏng gửi command:

```go
func (u *EditorUsecase) PlaceItem(ctx context.Context, userID string, in PlaceItemInput) (*PlaceItemOutput, error) {
    char, err := u.charReader.GetByUserID(ctx, userID); if err != nil { /*...*/ }
    item, err := u.repo.GetItemByID(ctx, in.ItemID);    if err != nil || item == nil { /*...*/ }
    mapID, _ := u.repo.GetMapIDByCode(ctx, in.MapCode)

    a, err := u.rooms.actor(in.MapCode); if err != nil { return nil, apperror.Internal(err) }

    reply := make(chan CmdResult, 1)
    select {
    case a.cmds <- Cmd{Kind: CmdPlace, CharID: char.ID, Item: item,
        X: in.X, Y: in.Y, PlaceID: uuid.NewString(), Reply: reply}:
    default:
        return nil, apperror.Internal(ErrBusy) // backpressure
    }

    res := <-reply // KHÔNG chạm DB; chỉ chờ actor xử lý trong RAM (µs)
    if res.Err != nil { return nil, mapErr(res.Err) }
    return &PlaceItemOutput{Placement: *res.Placement, NewCoins: res.NewCoins}, nil
}
```

Móc join/leave vào realtime để nạp/evict ví — tận dụng đúng các event bạn đã có (`player_joined` / `player_left` trong module realtime): khi player_joined ở `mapCode`, đọc coin & gửi `CmdJoin`; khi player_left, gửi `CmdLeave`.

---

## 4. Luồng place đầy đủ (sequence)

```
Client                Handler/Usecase       MapActor            Broadcaster     Writer         DB
  │  POST /editor/place   │                     │                   │            │             │
  ├──────────────────────>│  Cmd{place}         │                   │            │             │
  │                       ├────────────────────>│ validate RAM      │            │             │
  │                       │                     │ mutate RAM        │            │             │
  │                       │<── CmdResult ───────┤ (occupied, wallet)│            │             │
  │<─ 200 {placement,coins}│ (KHÔNG chờ DB)     │                   │            │             │
  │                       │                     ├── outbound ──────>│            │             │
  │                       │                     ├── dirty ─────────────────────> │ (gom lô)    │
  │<═ WS decoration_placed (tới mọi client) ════════════════════════┤            │             │
  │                       │                     │                   │      ~1s   ├── batch tx ─>│
```

Client nhận response + thấy vật thể realtime **trước khi** DB được ghi. Đúng đắn được đảm bảo bởi RAM (đã validate & mutate tuần tự), không phải bởi DB.

---

## 5. Client prediction + reconciliation (frontend)

Tận dụng phần lọc echo bạn đã có. Thêm optimistic + rollback:

```ts
// confirmPlacement(): render ngay, gắn requestId tạm
const tempId = `tmp_${crypto.randomUUID()}`
addLocalSprite(tempId, x, y, item)          // hiện ngay, không chờ server

try {
  const res = await editorService.placeItem({ item_id, map_code, x, y })
  reconcile(tempId, res.placement)           // đổi tempId -> id thật từ server
  gameStore.coins = res.new_coins            // coin thật từ actor
} catch (e) {
  removeLocalSprite(tempId)                   // server từ chối -> rollback
  showError(e.message)                        // "ô đã có vật thể" / "không đủ coins"
}
```

Với người chơi khác, giữ nguyên `onDecorationPlaced` hiện tại. Vì actor mutate tuần tự và broadcast qua 1 goroutine, thứ tự event nhất quán → không cần sort lại phía client.

> Optimistic là tuỳ chọn: nếu roundtrip HTTP tới actor đã ~vài ms (không còn DB), có thể **không cần** predict và chỉ render khi có response — đơn giản hơn, ít rollback. Bật predict khi RTT mạng người dùng cao.

---

## 6. Khi 1 map quá nóng: chunk sharding + Area of Interest

Actor per-map xử lý tuần tự ⇒ nếu **một** map có cực nhiều thao tác/giây, channel `cmds` là nút cổ chai. Lúc đó mới cần:

**Chunk sharding (như Minecraft chunk).** Chia map thành lưới chunk (vd 16×16 tile). Mỗi chunk một actor con (hoặc một shard lock). Lệnh route theo `chunk = (x/chunkPx, y/chunkPx)`. Occupancy/coin check chỉ trong chunk đó ⇒ song song hoá theo chunk. Coin (per-player, không thuộc chunk) tách sang **WalletShard** riêng (sharded theo charID) để không phải khoá chunk.

**Area of Interest (AoI).** Thay vì broadcast toàn room, chỉ publish placement cho người chơi trong bán kính quanh vị trí đặt (kênh Centrifuge theo chunk, client subscribe các chunk quanh mình). Giảm fan-out khi map đông.

> Với quy mô BigTown hiện tại (map làng, vài chục người/room), **1 actor/map là đủ** — đừng làm chunk sharding sớm (premature). Ghi lại đây làm đường nâng cấp khi metric `cmds` queue depth chạm ngưỡng.

---

## 7. Độ bền dữ liệu (durability) — các nút vặn & đánh đổi

Write-behind ⇒ crash giữa 2 lần flush có thể mất tối đa `every` (vd 1s) thao tác cuối. Các mức:

1. **Mặc định (như Minecraft autosave):** flush mỗi 1s + flush-on-shutdown. Mất tối đa ~1s khi crash cứng. Đủ tốt cho decoration.
2. **Chặt hơn:** giảm `every` (250ms) hoặc flush theo cả size lẫn thời gian (đã có). Đổi lại nhiều tx hơn.
3. **Zero-loss (nếu coin là "tiền thật"):** ghi **append-only journal (WAL)** *trước* khi actor reply — mỗi op nối vào file/stream tuần tự (rất nhanh), rồi mới apply RAM + write-behind vào Postgres. Khi khởi động lại, replay journal chưa flush. Đây là cách các hệ tài chính/DB làm; chỉ dùng nếu mất 1s coin là không chấp nhận được.

Chọn mức theo giá trị coin. Decoration BigTown: mức 1 là hợp lý.

---

## 8. Vì sao P1..P5 (Tier 1) biến mất ở Tier 2

| Lỗi | Cách Tier 2 loại bỏ |
|---|---|
| P1 TOCTOU coin | Coin check + trừ nằm trong **cùng một lần xử lý tuần tự** của actor; không có khoảng cách check-rồi-mới-trừ; wallet trong RAM là chân lý |
| P2 vật thể ma | Broadcast phát ra **sau khi** RAM đã là chân lý; DB chỉ theo sau. Không còn cảnh "DB fail nhưng đã broadcast" vì DB không quyết định gì |
| P3 double-refund | Delete kiểm `byID` tuần tự; lệnh xoá thứ 2 thấy đã mất → `ErrNotFound`, không refund lần hai |
| P4 shutdown mất task | `Writer.Close()` drain toàn bộ + flush; actor đóng `cmds` có trật tự |
| P5 validate client-side | Validate hình học/occupancy/coin đều ở actor (server), client chỉ preview |

Nói cách khác: Tier 1 sửa từng lỗi bằng transaction; Tier 2 **loại bỏ nguyên nhân** bằng cách hợp nhất mọi quyết định vào một điểm tuần tự trong RAM.

---

## 9. Kế hoạch rollout an toàn

1. **Feature flag per-map:** cờ `useRoomActor[mapCode]`. Bắt đầu bật cho 1 map ít người.
2. **Shadow validate:** giai đoạn đầu, sau mỗi flush so khớp `occupied` (RAM) với `SELECT` từ DB định kỳ; log lệch (phải luôn khớp).
3. **Metrics cần có:** độ sâu queue `cmds` & `dirty`, thời gian flush, số op/flush, tỉ lệ `ErrBusy`, tỉ lệ `ErrOccupied`, độ trễ actor xử lý (p50/p99). Cảnh báo khi queue depth tăng → tín hiệu cần chunk sharding (mục 6).
4. **Load test:** N client bắn place/delete song song lên cùng map & cùng ô; kiểm coin không âm, không vật thể ma, reload khớp DB.
5. **Rollback plan:** tắt cờ → quay lại đường Tier 1 đồng bộ (giữ cả hai path trong giai đoạn chuyển tiếp).

---

## 10. Checklist & khác biệt so với Tier 1

**Thêm mới (Tier 2):**
- [ ] Package `room`: `Cmd`, `CmdResult`, `MapActor`, `RoomManager`, `Writer` (persistOp).
- [ ] `MapActor.loadFromDB()` nạp placements khi tạo actor.
- [ ] Wallet residency: móc `CmdJoin/CmdLeave` vào realtime `player_joined/left`; `lazyLoadWallet` fallback.
- [ ] **Chuyển mọi nguồn ghi coin khác** (combat/leaderboard) sang command của actor để giữ bất biến "một người ghi".
- [ ] `Writer` batch: coalesce coin (absolute), placement `ON CONFLICT DO NOTHING`, delete gộp, reward_events append; `Close()` flush-on-shutdown.
- [ ] Broadcaster goroutine riêng (outbound channel) — không publish trong vòng mutate.
- [ ] Metrics + feature flag + shadow validate.

**Bỏ đi so với Tier 1:**
- [ ] Transaction đồng bộ trong `PlaceItem/DeletePlacement` (thay bằng gửi command).
- [ ] Coin guard DB trên hot path (nay guard trong RAM; DB chỉ nhận giá trị tuyệt đối theo lô).

**Giữ nguyên từ Tier 1:**
- [ ] `validatePlacement` (dùng lại logic, chạy trong actor).
- [ ] `event_type` đúng (`decoration_place` / `decoration_refund`).
- [ ] Frontend đọc coins/occupancy từ server; hiển thị lỗi cụ thể.
- [ ] Collision metadata `collision_x/y` + `extra_colliders` (không liên quan tầng realtime).

---

### Tóm tắt một câu

> Tier 1 = *"làm DB đúng"* (transaction, guard, persist-then-broadcast).
> Tier 2 = *"đừng để DB trên đường đi"* — RAM là chân lý, actor tuần tự là điểm quyết định duy nhất,
> DB nhận ghi theo lô ở luồng nền. Đây chính là mô hình world-in-memory + autosave của game xây dựng online.
