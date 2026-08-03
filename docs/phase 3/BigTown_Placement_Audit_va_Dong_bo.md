# BigTown — Placement Feature: Audit, Fix & Đồng bộ hoá hệ thống

> Commit review: `feat: add feature place decoration and stream realtime with other player` (`3c7db67`)
> Phạm vi: `backend/internal/module/editor/**`, `frontend/src/features/game/**`, `schema.sql`, `seed.sql`

---

## 1. Tóm tắt vấn đề

| # | Vấn đề | Mức độ | Gốc rễ | File |
|---|--------|--------|--------|------|
| P1 | TOCTOU coin → tiêu âm / double-spend | 🔴 Nghiêm trọng | Check coin đọc DB cũ, trừ coin async, `DeductCoins` không guard `>= price` | `editor_usecase.go`, `editor_repository.go` |
| P2 | Broadcast + trả success TRƯỚC khi persist → "vật thể ma" | 🔴 Nghiêm trọng | DB là source-of-truth nhưng broadcast trước commit | `editor_usecase.go` |
| P3 | Delete refund optimistic → coin hiển thị sai khi double-click | 🟠 Cao | Refund trả về client trước khi DELETE thật chạy | `editor_usecase.go` |
| P4 | Worker không graceful shutdown, `writeChan` không drain | 🟠 Cao | `context.Background()`, channel không close khi tắt server | `editor_usecase.go` |
| P5 | Không validate server-side (bounds / occupancy / snap grid) | 🟠 Cao | `checkOccupied` thuần client-side, X/Y nhận thẳng từ client | `editor_usecase.go`, `EditorPanel.vue` |
| P6 | `reward_events.event_type` hardcode `'decoration_place'` cho cả delete | 🟡 Trung bình | Sai nhãn loại sự kiện (dấu `coin_delta` thì đúng) | `editor_repository.go` |
| P7 | Bridge collision hardcode pixel theo `item.code` | 🟡 Trung bình | Không metadata-driven, giòn theo kích thước sprite | `editorSystem.ts` |
| P8 | Collision offset suy từ anchor, chưa nhận `collision_x/y` | 🟢 Thấp | Không khớp mẫu JSON của tool (kém chính xác cho box lệch tâm) | `editorSystem.ts` |

**Điểm ĐÚNG (giữ nguyên):** kiến trúc hexagonal tách tầng tốt; lọc echo realtime chuẩn (`onRealtimePlaced` bỏ qua placement của chính mình + guard trùng `id`); có `UNIQUE INDEX (map_id, x, y)`; công thức offset collision hiện tại (với origin `(0.5,1.0)`) tính đúng về mặt toán học.

---

## 2. Phân tích & phương pháp fix từng vấn đề

### P1 — TOCTOU coin (nghiêm trọng nhất)

**Hiện tại:** `PlaceItem` đọc `charInfo.Coins`, so sánh `< price`, trả `newCoins` optimistic, rồi đẩy task trừ coin sang worker. Trừ coin thật xảy ra **sau đó** trong `executePlaceTask`. `DeductCoinsWithTx` dùng `SET coins = coins - $1` **không có guard**.

**Hệ quả:** N request song song (hoặc double-click) cùng đọc coin cũ → tất cả pass check → coin xuống âm. Đây là lỗ double-spend khai thác được từ client.

**Fix cốt lõi:** check + trừ coin phải **atomic trong 1 câu UPDATE có điều kiện**, và không được fire-and-forget phần tiền:

```sql
-- Guard ngay tại DB: chỉ trừ khi đủ coin, trả về số coin mới
UPDATE characters
SET    coins = coins - $1
WHERE  id = $2 AND coins >= $1
RETURNING coins;
```

`RowsAffected == 0` (hoặc `sql.ErrNoRows` khi dùng RETURNING) ⇒ không đủ coin ⇒ trả `BadRequest`.

---

### P2 — Broadcast trước persist → vật thể ma

**Hiện tại:** cả place lẫn delete broadcast Centrifuge + trả HTTP success **ngay**, ghi DB async. Khi 2 người đặt cùng ô: cả hai nhận success + cả hai broadcast đi; ở worker, 1 INSERT dính `UNIQUE (map_id,x,y)` → rollback (coin không trừ) nhưng client đã render + đã trừ coin, người khác đã thấy. Reload ⇒ biến mất.

**Nguyên tắc:** nếu DB là source-of-truth thì **persist xong mới broadcast**. Nếu muốn broadcast trước (độ trễ thấp) thì phải chuyển source-of-truth vào bộ nhớ (xem Tier 2, mục 3.2).

---

### P3 — Double-refund optimistic (delete)

**Hiện tại:** `DeletePlacement` đọc placement, trả `newCoins = coins + price` ngay cho **mỗi** request. Double-click ⇒ cả hai đọc thấy row còn ⇒ cả hai trả refund cho client (DB an toàn nhờ rollback khi `DeletePlacementWithTx` trả `rows=0`, nhưng UI sai tới khi refresh).

**Fix:** DELETE có guard + đọc coin thật sau commit (gộp trong Tier 1).

---

### P4 — Graceful shutdown

**Hiện tại:** worker dùng `context.Background()`; `writeChan` không close, không drain khi SIGTERM ⇒ task đang chờ mất ⇒ coin đã trừ (client) + đã broadcast nhưng không vào DB.

**Fix:** nếu chuyển sang ghi đồng bộ (Tier 1) thì bỏ hẳn queue → hết vấn đề. Nếu giữ write-behind (Tier 2) thì phải `Flush()` khi shutdown (mục 3.2).

---

### P5 — Validate server-side

**Hiện tại:** `game:checkOccupied` chỉ chạy ở client (`placements.value.some(...)`), client sửa được; X/Y nhận thẳng, không check bounds / snap grid / quyền map.

**Fix:** server phải là nơi quyết định cuối. Tối thiểu:
- `x % TILE == 0 && y % TILE == 0` (đúng lưới), hoặc snap ở server.
- `0 <= x < mapWidth`, `0 <= y < mapHeight`.
- Occupancy: dựa vào `UNIQUE (map_id,x,y)` (Tier 1) hoặc in-memory map (Tier 2).

---

### P6 — `reward_events` sai nhãn

`InsertRewardEventWithTx` hardcode `event_type='decoration_place'`. Dấu `coin_delta = -amount` thì đúng (place ra `-price`, delete ra `+price`), nhưng event refund bị gán nhãn `decoration_place`. Truyền `event_type` vào tham số:

```go
func (r *EditorRepository) InsertRewardEventWithTx(
    ctx context.Context, tx *sql.Tx, characterID, eventType string, coinDelta int,
) error {
    _, err := tx.ExecContext(ctx,
        `INSERT INTO reward_events (character_id, event_type, coin_delta) VALUES ($1,$2,$3)`,
        characterID, eventType, coinDelta)
    return err
}
// place:  InsertRewardEventWithTx(ctx, tx, charID, "decoration_place",  -price)
// delete: InsertRewardEventWithTx(ctx, tx, charID, "decoration_refund", +price)
```

---

### P7 — Bridge collision hardcode

`renderPlacementsGroup` tạo zone rail cầu bằng pixel cứng (`p.y - 28`, `48`, `8`...) theo prefix `item.code`. Giòn khi đổi sprite. **Đưa các rail vào metadata** dạng mảng box phụ để cùng tool xử lý:

```jsonc
// metadata_json của item cầu:
{ "collides": false, "frameWidth":48, "frameHeight":32, "frame":0,
  "extra_colliders": [
    { "dx": 0, "dy": -28, "w": 48, "h": 8 },
    { "dx": 0, "dy": -4,  "w": 48, "h": 8 }
  ]
}
```

Runtime lặp `meta.extra_colliders` tạo zone thay vì `if item.code.startsWith(...)`.

---

### P8 — Collision offset nhận `collision_x/y` trực tiếp

Mẫu JSON của tool dùng `collision_x`, `collision_y` (offset từ mép trái/trên của **frame**). Với Arcade `StaticBody` + `updateFromGameObject`, vị trí body = `(gameObject.x − displayOriginX + offsetX, …)`, và `displayOriginX = originX × frameW`. Do đó nếu đo offset từ mép frame thì:

> **offset của body = đúng bằng (collision_x, collision_y)** — không cần suy từ anchor nữa.

Xem code frontend cập nhật ở mục 4.1. Giữ fallback cho item cũ chưa có `collision_x/y`.

---

## 3. Hai hướng triển khai

### 3.1 Tier 1 — Sửa đúng tối thiểu (đồng bộ, 1 roundtrip DB)

Mục tiêu: **đúng đắn tuyệt đối** với thay đổi nhỏ nhất. Đánh đổi: mỗi thao tác tốn 1 roundtrip DB (chấp nhận được ở quy mô nhỏ/vừa).

**Repository — thêm guard + trả coin thật:**

```go
// UPDATE có điều kiện; trả coin mới, ErrNoRows nếu không đủ.
func (r *EditorRepository) DeductCoinsGuardedWithTx(
    ctx context.Context, tx *sql.Tx, characterID string, amount int,
) (int, error) {
    var newCoins int
    err := tx.QueryRowContext(ctx,
        `UPDATE characters SET coins = coins - $1
         WHERE id = $2 AND coins >= $1
         RETURNING coins`, amount, characterID).Scan(&newCoins)
    if errors.Is(err, sql.ErrNoRows) {
        return 0, ErrInsufficientCoins // sentinel error riêng
    }
    return newCoins, err
}

func (r *EditorRepository) AddCoinsGuardedWithTx(
    ctx context.Context, tx *sql.Tx, characterID string, amount int,
) (int, error) {
    var newCoins int
    err := tx.QueryRowContext(ctx,
        `UPDATE characters SET coins = coins + $1 WHERE id = $2 RETURNING coins`,
        amount, characterID).Scan(&newCoins)
    return newCoins, err
}
```

**Usecase — place đồng bộ, persist-then-broadcast:**

```go
func (u *EditorUsecase) PlaceItem(ctx context.Context, userID string, in PlaceItemInput) (*PlaceItemOutput, error) {
    char, err := u.charReader.GetByUserID(ctx, userID)
    if err != nil { return nil, apperror.NotFound("Không tìm thấy nhân vật", err) }

    mapID, err := u.repo.GetMapIDByCode(ctx, in.MapCode)
    if err != nil { return nil, apperror.NotFound("Không tìm thấy bản đồ", err) }

    item, err := u.repo.GetItemByID(ctx, in.ItemID)
    if err != nil { return nil, apperror.Internal(err) }
    if item == nil { return nil, apperror.BadRequest("Vật phẩm không tồn tại", nil) }

    // --- Validate server-side (P5) ---
    if err := validatePlacement(in.X, in.Y, mapWidth, mapHeight, tileSize); err != nil {
        return nil, apperror.BadRequest("Toạ độ không hợp lệ", err)
    }

    placementID := uuid.NewString()

    tx, err := u.db.BeginTx(ctx, nil)
    if err != nil { return nil, apperror.Internal(err) }
    defer tx.Rollback()

    // 1) Trừ coin ATOMIC (P1)
    newCoins, err := u.repo.DeductCoinsGuardedWithTx(ctx, tx, char.ID, item.Price)
    if errors.Is(err, repository.ErrInsufficientCoins) {
        return nil, apperror.BadRequest("Không đủ coins", nil)
    }
    if err != nil { return nil, apperror.Internal(err) }

    // 2) Insert placement; bắt lỗi trùng ô (P2)
    p := &entity.Placement{ID: placementID, MapID: mapID, CharacterID: char.ID,
        ItemID: in.ItemID, X: in.X, Y: in.Y}
    if err := u.repo.PlaceItemWithIDAndTx(ctx, tx, p); err != nil {
        if isUniqueViolation(err) { // pgerrcode 23505
            return nil, apperror.BadRequest("Ô này đã có vật thể", nil)
        }
        return nil, apperror.Internal(err)
    }

    // 3) Log event
    if err := u.repo.InsertRewardEventWithTx(ctx, tx, char.ID, "decoration_place", -item.Price); err != nil {
        return nil, apperror.Internal(err)
    }

    if err := tx.Commit(); err != nil { return nil, apperror.Internal(err) }

    p.CreatedAt = time.Now()
    out := &PlaceItemOutput{Placement: *p, NewCoins: newCoins}

    // 4) Broadcast SAU commit (P2)
    _ = u.publisher.PublishRoom(ctx, in.MapCode, map[string]any{
        "type": "decoration_placed", "placement": out.Placement,
    })
    return out, nil
}
```

**Delete tương tự:** `BeginTx` → `DeletePlacementWithTx` (guard `rows==1` else `NotFound`) → `AddCoinsGuardedWithTx` → `InsertRewardEventWithTx("decoration_refund", +price)` → `Commit` → broadcast. Vì DELETE guard, double-click lần 2 trả `NotFound` sạch sẽ (P3 hết).

**`isUniqueViolation`:**

```go
import "github.com/jackc/pgx/v5/pgconn"
func isUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
```

> Xoá luôn `writeChan`, `startBackgroundWorkers`, `editorDbTask`, `executePlaceTask`, `executeDeleteTask` → P4 tự hết.

---

### 3.2 Tier 2 — Kiến trúc tối ưu latency (in-memory authority + write-behind)

**Đây là câu trả lời cho câu hỏi tối ưu.** Ý tưởng học từ Minecraft & game xây dựng online: **DB không nằm trên hot path**. Source-of-truth là bộ nhớ; DB chỉ là bản lưu bền (durability backstop).

Mô hình *actor per-map*: mỗi map là 1 goroutine sở hữu toàn bộ state, xử lý command tuần tự qua channel (không mutex rải rác, không race, có thứ tự).

```go
type Cmd struct {
    Kind      string        // "place" | "delete"
    UserID    string
    CharID    string
    Item      *entity.DecorationItem
    X, Y      int
    ID        string
    Reply     chan CmdResult // trả kết quả đồng bộ cho HTTP handler
}

type CmdResult struct {
    Placement *entity.Placement
    NewCoins  int
    Err       error
}

type MapActor struct {
    mapID, mapCode string
    occupied  map[[2]int]*entity.Placement // O(1) occupancy
    wallets   map[string]int               // coin cache theo charID (nạp lazy)
    cmds      chan Cmd
    dirty     chan persistOp               // → write-behind flusher
    publisher port.RoomPublisher
}

func (m *MapActor) run() {
    for c := range m.cmds {           // TUẦN TỰ → không cần lock, có ordering
        switch c.Kind {
        case "place":  m.handlePlace(c)
        case "delete": m.handleDelete(c)
        }
    }
}

func (m *MapActor) handlePlace(c Cmd) {
    key := [2]int{c.X, c.Y}
    if _, taken := m.occupied[key]; taken {           // occupancy in-memory
        c.Reply <- CmdResult{Err: ErrOccupied}; return
    }
    coins := m.wallets[c.CharID]
    if coins < c.Item.Price {                          // coin guard in-memory
        c.Reply <- CmdResult{Err: ErrInsufficientCoins}; return
    }
    // Mutate memory = quyết định cuối cùng
    m.wallets[c.CharID] = coins - c.Item.Price
    p := &entity.Placement{ID: c.ID, MapID: m.mapID, CharacterID: c.CharID,
        ItemID: c.Item.ID, X: c.X, Y: c.Y, CreatedAt: time.Now()}
    m.occupied[key] = p

    // Broadcast NGAY từ bộ nhớ (không chờ DB) — độ trễ chỉ 1 hop WS
    _ = m.publisher.PublishRoom(context.Background(), m.mapCode, map[string]any{
        "type": "decoration_placed", "placement": p,
    })
    // Ghi bền BẤT ĐỒNG BỘ (write-behind, batch)
    m.dirty <- persistOp{Kind: "place", P: p, CharID: c.CharID, CoinDelta: -c.Item.Price}

    c.Reply <- CmdResult{Placement: p, NewCoins: m.wallets[c.CharID]}
}
```

**Write-behind flusher** — gom nhiều op, ghi theo lô, chỉ ghi cái "dirty":

```go
func (w *Writer) loop() {
    ticker := time.NewTicker(1 * time.Second) // flush mỗi 1s
    batch := make([]persistOp, 0, 512)
    flush := func() {
        if len(batch) == 0 { return }
        tx, _ := w.db.BeginTx(context.Background(), nil)
        // multi-row INSERT / COPY cho placements, UPDATE gộp cho coins
        w.persistBatch(tx, batch)
        _ = tx.Commit()
        batch = batch[:0]
    }
    for {
        select {
        case op := <-w.dirty:
            batch = append(batch, op)
            if len(batch) >= 512 { flush() } // flush theo ngưỡng size
        case <-ticker.C:
            flush()                          // flush theo thời gian
        case <-w.done:
            flush()                          // FLUSH khi shutdown (P4)
            return
        }
    }
}
```

**Đánh đổi (chấp nhận như Minecraft):** nếu server crash giữa 2 lần flush, mất tối đa ~1s thao tác cuối. Giảm rủi ro bằng: flush-on-shutdown, chu kỳ ngắn, hoặc WAL/append-only log trước khi flush batch.

**Vì sao mô hình này vừa nhanh vừa đúng:**
- Không đọc/ghi DB trên hot path ⇒ độ trễ ≈ RTT websocket, không phụ thuộc tải DB.
- Mutation tuần tự trong actor ⇒ không race, không TOCTOU, có thứ tự ⇒ P1/P2/P3 biến mất về bản chất.
- Broadcast từ memory (đã là chân lý) ⇒ nhất quán giữa các client.
- Write-behind + dirty flag ⇒ giảm số ghi (coalescing last-write-wins cho cùng 1 ô).

---

## 4. Sửa phía Frontend

### 4.1 Collision offset đọc `collision_x/y` (P8) — `editorSystem.ts`

```ts
if (meta.collides) {
  this.scene.physics.add.existing(sprite, true)
  const body = sprite.body as Phaser.Physics.Arcade.StaticBody

  const fW = hasFrames ? meta.frameWidth  : sprite.width
  const fH = hasFrames ? meta.frameHeight : sprite.height

  const cw = meta.collision_w ?? fW
  const ch = meta.collision_h ?? fH
  body.setSize(cw, ch)

  if (meta.collision_x !== undefined && meta.collision_y !== undefined) {
    // Offset đo từ mép frame → dùng thẳng, khớp export của tool
    body.setOffset(meta.collision_x, meta.collision_y)
  } else {
    // Fallback item cũ: căn giữa ngang + đáy theo anchor
    body.setOffset(-cw / 2 + fW * sprite.originX, -ch + fH * sprite.originY)
  }
  body.updateFromGameObject()
  this.collisionGroup.add(sprite)
}

// P7: colliders phụ từ metadata thay cho hardcode bridge
if (Array.isArray(meta.extra_colliders)) {
  for (const c of meta.extra_colliders) {
    const zone = this.scene.add.zone(p.x + c.dx, p.y + c.dy, c.w, c.h)
    this.scene.physics.add.existing(zone, true)
    this.collisionGroup.add(zone)
  }
}
```

### 4.2 Coins & occupancy là của server, không phải client

- `onPlacementDone`: `gameStore.coins = detail.newCoins` — giữ, nhưng `newCoins` giờ là **giá trị thật** trả sau commit (Tier 1) hoặc từ memory-authority (Tier 2), không còn optimistic.
- `game:checkOccupied` chỉ để **preview/UX** (hiện tô đỏ). Không dựa vào nó để quyết định — server mới là nơi chốt (đã có ở mục 3). Khi server trả lỗi "ô đã có vật thể" thì client hiển thị và huỷ preview.
- Xử lý lỗi trong `confirmPlacement`: hiện đã `catch` và dispatch `game:placementCancel` — tốt; bổ sung hiển thị message lỗi cụ thể từ server (không đủ coin / trùng ô / ngoài bản đồ).

---

## 5. Tool soạn metadata collision

Xem file `spritesheet-collision-tool.html` đi kèm. Import PNG → cắt theo lưới hoặc chọn vùng → vẽ hộp collision → xuất JSON đúng mẫu (`file_name`, `item_code`, `name`, `price`, `collides`, `collision_x/y/w/h`) **cộng thêm** `frameWidth/frameHeight/frame/anchorX/anchorY` để cắm thẳng vào `items.metadata_json`. Tool còn xuất sẵn câu `INSERT` cho `seed.sql`.

**Khuyến nghị migrate:** cập nhật runtime theo mục 4.1 để đọc `collision_x/y`, rồi dần thay metadata trong `seed.sql` sang có `collision_x/y` (chính xác hơn cho box lệch tâm như nhà, gốc cây).

---

## 6. Checklist triển khai

- [ ] **P1/P2/P3** Chọn Tier 1 (nhanh gọn) hoặc Tier 2 (tối ưu) — không trộn nửa vời.
- [ ] Thêm `DeductCoinsGuardedWithTx` / `AddCoinsGuardedWithTx` + sentinel `ErrInsufficientCoins`.
- [ ] `isUniqueViolation` bắt `23505`; place/delete persist-then-broadcast.
- [ ] **P4** Bỏ `writeChan` (Tier 1) hoặc thêm `Flush()` on shutdown (Tier 2).
- [ ] **P5** `validatePlacement` (grid / bounds) ở server.
- [ ] **P6** `InsertRewardEventWithTx` nhận `event_type`; delete dùng `decoration_refund`.
- [ ] **P7** Chuyển bridge sang `extra_colliders` trong metadata.
- [ ] **P8** Runtime đọc `collision_x/y`; giữ fallback.
- [ ] Frontend: coins/occupancy do server chốt; hiển thị lỗi cụ thể.
- [ ] Test: N request place song song không làm coin âm; đặt trùng ô bị từ chối sạch; double-click delete không double-refund; kill server giữa chừng không tạo vật thể ma.
