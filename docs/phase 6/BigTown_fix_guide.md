# BigTown — Hướng dẫn xử lý sau review nhánh `dev`

Phạm vi: các commit từ `8359ec9` → `c5abd73`. Ưu tiên từ trên xuống: **(1) bắt buộc**, (2)(3) nên làm, (4)(5) tối ưu thêm.

| # | Vấn đề | Mức | File |
|---|--------|-----|------|
| 1 | `dev` không compile (biến `placements` ngoài scope) | 🔴 Bắt buộc | `editor/usecase/editor_usecase.go` |
| 2 | `GetItemByID` query DB ngay trong actor loop (`handleDelete`) | 🟠 Nên làm | `editor/room/map_actor.go` |
| 3 | Draw order không xác định khi 2 item chồng cùng ô | 🟠 Nên làm | `game/systems/editorSystem.ts` + metadata |
| 4 | `DELETE` / `reward_events` ghi từng dòng | 🟢 Tối ưu | `editor/room/writer.go` |
| 5 | Blocking send vào `dirty` có thể đông cứng room | 🟢 Phòng thủ | `editor/room/map_actor.go` + `writer.go` |

---

## 1. 🔴 Sửa lỗi compile (`editor_usecase.go`)

Lỗi build thật đã xác nhận:

```
editor_usecase.go:69:3:  undefined: placements
editor_usecase.go:80:17: undefined: placements
```

Nguyên nhân: `placements` được khai báo bằng `:=` **bên trong** block `if`, nên không tồn tại ở nhánh `else` và ở `return`.

**Trước (không build được):**

```go
livePlacements := u.rooms.GetPlacements(mapCode)
if livePlacements == nil {
    placements, err := u.repo.GetPlacementsByMap(ctx, mapID) // scope kẹt trong if
    if err != nil {
        return nil, apperror.Internal(err)
    }
    if placements == nil {
        placements = make([]entity.Placement, 0)
    }
} else {
    placements = livePlacements   // undefined
}
// ...
Placements: placements,           // undefined
```

**Sau (khai báo `placements` ở scope hàm):**

```go
// RAM của actor là nguồn chuẩn (write-behind khiến DB trễ hơn RAM).
// Chỉ chạm DB khi actor thực sự không tồn tại — trường hợp gần như không xảy ra
// vì GetPlacements() tự lazy-create actor + loadFromDB().
var placements []entity.Placement
if live := u.rooms.GetPlacements(mapCode); live != nil {
    placements = live
} else {
    dbP, err := u.repo.GetPlacementsByMap(ctx, mapID)
    if err != nil {
        return nil, apperror.Internal(err)
    }
    placements = dbP
}
if placements == nil {
    placements = make([]entity.Placement, 0)
}
```

> ✅ Sau khi sửa, chốt lại bằng `go build ./...` và `go vet ./...` trong CI để lần sau lỗi scope kiểu này bị chặn trước khi merge.

---

## 2. 🟠 Bỏ query DB khỏi actor loop — cache `collides`

`handleDelete` gọi `parseItemCollides()` → `repo.GetItemByID()` (một round-trip DB, hiện tại tới Aiven Nhật Bản ~SG↔JP) **ngay trong goroutine actor**. Actor xử lý tuần tự mọi lệnh place/delete/claim của cả room, nên mỗi lần chạm DB ở đây làm *đứng cả room*. Bạn đã preload `prices` từ `GetDecorationItems` — hãy preload `collides` theo đúng cách đó.

**2.1. Thêm field vào `MapActor`:**

```go
type MapActor struct {
    // ...
    prices   map[string]int  // itemID -> price (đã có)
    collides map[string]bool // itemID -> có va chạm không (THÊM MỚI)
    // ...
}
```

**2.2. Khởi tạo trong `NewMapActor`:**

```go
prices:   make(map[string]int),
collides: make(map[string]bool), // THÊM MỚI
```

**2.3. Nạp cache trong `loadFromDB` (dùng lại vòng lặp items sẵn có):**

```go
for _, it := range items {
    m.prices[it.ID] = it.Price
    m.collides[it.ID] = parseMetadataCollides(it.MetadataJSON) // THÊM MỚI — không tốn query mới
}
```

**2.4. Thay `parseItemCollides` (query mỗi lần) bằng bản đọc cache, fallback DB đúng 1 lần cho item thêm lúc runtime:**

```go
// itemCollides: ưu tiên cache RAM; chỉ chạm DB đúng một lần cho item mới,
// sau đó cache lại. Chạy trong actor nên không cần lock (giống prices).
func (m *MapActor) itemCollides(itemID string) bool {
    if c, ok := m.collides[itemID]; ok {
        return c
    }
    item, err := m.repo.GetItemByID(context.Background(), itemID)
    if err != nil || item == nil {
        return false
    }
    c := parseMetadataCollides(item.MetadataJSON)
    m.collides[itemID] = c
    return c
}
```

**2.5. Đổi 2 chỗ gọi cũ:**

```go
// Trong loadFromDB:
if m.itemCollides(p.ItemID) {          // thay cho: if collides, _ := m.parseItemCollides(...)
    m.hasCollision[key] = true
}

// Trong handleDelete (vòng recompute hasCollision):
hasCol := false
for _, pp := range filtered {
    if m.itemCollides(pp.ItemID) {     // thay cho: parseItemCollides(...)
        hasCol = true
        break
    }
}
```

Xong thì xoá hàm `parseItemCollides` cũ (không còn ai gọi). Kết quả: đường xóa item không còn round-trip DB trong actor cho các item đã biết (99% trường hợp).

---

## 3. 🟠 Draw order xác định khi 2 item chồng cùng ô

Depth hiện tại: `itemDepth + p.y / 10000.0`. Hai item **cùng ô** có cùng `(x, y)`, nên nếu `meta.depth` giống nhau → **cùng depth → thứ tự vẽ không xác định** (z-fighting giữa nền và vật đặt lên).

**Cách chính (khuyến nghị) — đặt `depth` theo metadata từng loại item.** Vật "nền/sàn" phải có `depth` nhỏ hơn vật đặt lên:

```jsonc
// item nền (thảm, bệ, sàn trang trí)
{ "collides": false, "depth": 1 }
// item đặt lên trên nền
{ "collides": false, "depth": 3 }
```

Code depth hiện tại đã tôn trọng `meta.depth` khi nó là number nên **không cần đổi code** — chỉ cần set data cho item nền thấp hơn.

**Tuỳ chọn phòng thủ — thêm tiebreak ổn định trong frontend** để không bao giờ hoà, kể cả khi quên set metadata. Dùng thứ tự trong ô (item vào sau nằm trên):

```ts
// p.stackIndex: vị trí trong danh sách item của ô (0 = dưới cùng), do BE trả về
const itemDepth = (typeof meta.depth === 'number'
  ? meta.depth
  : meta.is_animal ? PLAYER_DEPTH + 0.2 : meta.collides ? PLAYER_DEPTH : 2)
  + p.y / 10000.0
  + (p.stackIndex ?? 0) * 0.001   // tiebreak nhỏ, đảm bảo không hoà
sprite.setDepth(itemDepth)
```

Nếu chọn tuỳ chọn này, BE cần đính kèm chỉ số thứ tự khi trả `placements` (thứ tự append trong `occupied[key]`).

---

## 4. 🟢 Gộp batch cho DELETE và reward_events (`writer.go`)

Hiện `flush()` chèn placement theo multi-row (tốt) nhưng `DELETE` và `reward_events` vẫn chạy từng dòng. Gộp lại để cắt round-trip. Driver là **pgx v5 (không có `pq.Array`)**, nên dùng IN-list sinh placeholder — giống hệt style `batchInsertPlacements` của bạn.

**Batch delete:**

```go
if len(deleteOps) > 0 {
    placeholders := make([]string, len(deleteOps))
    args := make([]interface{}, len(deleteOps))
    for i, op := range deleteOps {
        placeholders[i] = fmt.Sprintf("$%d", i+1)
        args[i] = op.P.ID
    }
    query := "DELETE FROM map_placements WHERE id IN (" + strings.Join(placeholders, ",") + ")"
    if _, err := tx.Exec(query, args...); err != nil {
        log.Printf("[writer] failed to batch delete %d placements: %v", len(deleteOps), err)
        return
    }
}
```

**Batch reward_events (multi-row INSERT):**

```go
var evOps []persistOp
for _, op := range batch {
    if op.EventType != "" {
        evOps = append(evOps, op)
    }
}
if len(evOps) > 0 {
    rows := make([]string, 0, len(evOps))
    args := make([]interface{}, 0, len(evOps)*3)
    for i, op := range evOps {
        b := i * 3
        rows = append(rows, fmt.Sprintf("($%d,$%d,$%d)", b+1, b+2, b+3))
        args = append(args, op.CharID, op.EventType, op.CoinDelta)
    }
    query := "INSERT INTO reward_events (character_id, event_type, coin_delta) VALUES " +
        strings.Join(rows, ",")
    if _, err := tx.Exec(query, args...); err != nil {
        log.Printf("[writer] failed to batch insert %d reward events: %v", len(evOps), err)
        return
    }
}
```

> Giữ nguyên thứ tự trong transaction: **INSERT placements → DELETE placements → INSERT reward_events**. Thứ tự INSERT-trước-DELETE là thứ đảm bảo "đặt rồi xoá trong cùng batch" cho kết quả đúng, đừng đảo.

---

## 5. 🟢 Phòng thủ backpressure — đừng để writer chậm làm đông cứng room

`handlePlace`/`handleDelete`/`CmdCredit`/`CmdClaimCoin` gửi `m.dirty <- persistOp{...}` bằng **blocking send**. Nếu DB chậm kéo dài và buffer `in` (10000) đầy, actor kẹt ở lệnh send → `cmds` dồn → `SendCmd` trả `ErrBusy` → lỗi lan cả room.

Ở tải hiện tại điều này **chưa xảy ra** (report cho thấy req/s phẳng, không nghẽn) — nên đây là **phòng thủ**, không phải chữa cháy. Việc cần nhất là *nhìn thấy* backpressure trước khi nó cắn:

**5.1. Expose độ sâu hàng đợi để theo dõi (thêm vào `Writer`):**

```go
func (w *Writer) QueueLen() int { return len(w.in) }
```

Bắn `w.QueueLen()` và `len(actor.cmds)` ra Prometheus cạnh các metric goroutines/heap bạn đã có. Khi `QueueLen()` chạm ~80% của 10000 kéo dài → đó là tín hiệu DB không theo kịp, cần nâng DB hoặc rút ngắn `every`.

**5.2. (Tuỳ chọn) Shed load ở tầng HTTP thay vì đông cứng room.** Trước khi `SendCmd(CmdPlace)`, nếu hàng đợi ghi đã gần đầy thì từ chối sớm với thông báo "hệ thống đang bận" — người chơi retry, còn hơn để cả room treo. Đây là đánh đổi availability-vs-durability, cân nhắc theo nhu cầu.

> ⚠️ **Không** đổi blocking send thành "non-blocking + drop op". Drop một `persistOp` = mất vĩnh viễn placement/coin đó khỏi DB (dù còn trong RAM tới lúc restart thì mất). Thà chặn còn hơn mất dữ liệu.

---

## Checklist

- [ ] (1) Sửa scope `placements`, `go build ./...` + `go vet ./...` xanh, thêm 2 lệnh này vào CI
- [ ] (2) Thêm `collides` cache, thay `parseItemCollides` → `itemCollides`, xoá hàm cũ
- [ ] (3) Set `meta.depth` cho item nền < item đặt lên (và/hoặc thêm tiebreak `stackIndex`)
- [ ] (4) Gộp batch DELETE + reward_events
- [ ] (5) Expose `QueueLen()` + `len(cmds)` ra metrics
- [ ] Chạy lại placement load test ở mức tải **dưới điểm bão hoà** (p95 < ~1s) để đo sạch trước/sau
