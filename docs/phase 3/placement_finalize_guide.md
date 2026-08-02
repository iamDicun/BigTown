# BigTown — Placement: những gì cần sửa để coi là "xong" + dọn code rác

> Phạm vi: nhánh `dev`, từ `3c7db67` tới `246b9f9`.
> Trạng thái hiện tại: kiến trúc Tier 2 (actor‑per‑map + write‑behind) đã đúng về bản chất cho double‑spend / trùng ô / double‑refund / collision metadata‑driven. Còn **1 bug thật sự chặn "xong"**, **1 rủi ro shutdown**, **1 hợp đồng ví cần chốt**, và một mớ **code chết Tier 1** cần xoá.

Thứ tự ưu tiên:

| Mức | Việc | Vì sao |
|-----|------|--------|
| **P0 – bắt buộc** | Fix 1: Delete‑before‑flush | Xoá món vừa đặt trong ~1s trả `NotFound` sai. Đây là bug người chơi gặp thật. |
| **P1 – nên làm** | Fix 2: Shutdown race | `Shutdown()` có thể panic "send on closed channel" và mất ghi cuối. |
| **P2 – chốt trước khi thêm nguồn coin** | Fix 3: Hook `CreditCoins` cho bất biến ví | Cơ chế `CmdCredit` có sẵn nhưng chưa nối gì; chưa nguy hiểm ngay nhưng dễ vỡ ngầm sau này. |
| **Cleanup** | Xoá 8–10 method Tier 1 chết | Repo/port phình, gây hiểu nhầm còn 2 code path. |

---

## FIX 1 (P0) — Delete không được đọc placement từ DB nữa

### Vấn đề
`EditorUsecase.DeletePlacement` đọc `GetPlacementByID` từ **DB** để lấy `ItemID` + `MapID`. Nhưng nguồn chân lý là **actor RAM** (`byID`), còn write‑behind trễ tới 1s. Đặt xong xoá ngay → DB chưa có row → trả `NotFound`, dù actor đang giữ nó.

### Hướng sửa
Client luôn biết `mapCode` nó đang đứng. Cho delete gửi kèm `mapCode`; usecase resolve actor theo `mapCode` rồi để **actor tự tra `byID`** và tự tính giá hoàn từ một bảng giá nạp sẵn trong RAM. Không còn đọc placement/item/mapCode từ DB.

### 1.1 Actor giữ bảng giá trong RAM — `backend/internal/module/editor/room/map_actor.go`

Thêm field vào struct:

```go
type MapActor struct {
	mapID    string
	mapCode  string
	tileSize int
	mapW     int
	mapH     int

	occupied  map[[2]int]*entity.Placement
	byID      map[string]*entity.Placement
	wallets   map[string]int
	residents map[string]int
	prices    map[string]int // itemID -> price (nạp sẵn, dùng cho refund khi delete)

	cmds     chan Cmd
	outbound chan any
	dirty    chan persistOp
	done     chan struct{} // đóng khi run() kết thúc (cho Fix 2)

	charReader port.CharacterReader
	repo       port.EditorRepository
}
```

Khởi tạo trong `NewMapActor` (thêm 2 dòng vào literal khởi tạo):

```go
	m := &MapActor{
		// ... giữ nguyên các field cũ ...
		prices: make(map[string]int),
		done:   make(chan struct{}),
	}
```

Nạp giá trong `loadFromDB` (đặt ngay đầu hàm, trước khi load placements):

```go
func (m *MapActor) loadFromDB() error {
	items, err := m.repo.GetDecorationItems(context.Background())
	if err != nil {
		return err
	}
	for _, it := range items {
		m.prices[it.ID] = it.Price
	}

	placements, err := m.repo.GetPlacementsByMap(context.Background(), m.mapID)
	if err != nil {
		return err
	}
	for _, p := range placements {
		pCopy := p
		key := [2]int{p.X, p.Y}
		m.occupied[key] = &pCopy
		m.byID[p.ID] = &pCopy
	}
	return nil
}
```

Thay `handleDelete` — không nhận `c.Item` nữa, tự lấy giá từ `m.prices` (có lazy fallback cho item thêm sau khi actor đã chạy):

```go
func (m *MapActor) handleDelete(c Cmd) {
	p, ok := m.byID[c.TargetID]
	if !ok {
		c.Reply <- CmdResult{Err: ErrNotFound}
		return
	}
	if p.CharacterID != c.CharID {
		c.Reply <- CmdResult{Err: ErrNotOwner}
		return
	}

	price, ok := m.prices[p.ItemID]
	if !ok {
		// item được thêm sau khi actor khởi động: nạp 1 lần rồi cache (hiếm khi xảy ra)
		item, err := m.repo.GetItemByID(context.Background(), p.ItemID)
		if err != nil || item == nil {
			c.Reply <- CmdResult{Err: ErrNotFound}
			return
		}
		price = item.Price
		m.prices[p.ItemID] = price
	}

	coins, err := m.getOrLoadWallet(c.CharID)
	if err != nil {
		c.Reply <- CmdResult{Err: err}
		return
	}

	newCoins := coins + price
	m.wallets[c.CharID] = newCoins

	delete(m.occupied, [2]int{p.X, p.Y})
	delete(m.byID, p.ID)

	c.Reply <- CmdResult{NewCoins: newCoins}

	m.outbound <- map[string]any{
		"type":        "decoration_deleted",
		"placementId": p.ID,
	}

	m.dirty <- persistOp{
		Kind:      opDelete,
		P:         p,
		CharID:    c.CharID,
		NewCoins:  newCoins,
		CoinDelta: price,
		EventType: "decoration_refund",
	}
}
```

### 1.2 Usecase — bỏ hết DB read, nhận `mapCode` — `usecase/editor_usecase.go`

```go
func (u *EditorUsecase) DeletePlacement(ctx context.Context, userID, mapCode, placementID string) (int, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return 0, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	a, err := u.rooms.Actor(mapCode)
	if err != nil {
		return 0, apperror.Internal(err)
	}
	if a == nil {
		return 0, apperror.NotFound("Không tìm thấy bản đồ", nil)
	}

	reply := make(chan room.CmdResult, 1)
	cmd := room.Cmd{
		Kind:     room.CmdDelete,
		CharID:   charInfo.ID,
		TargetID: placementID,
		Reply:    reply,
	}
	if err := a.SendCmd(cmd); err != nil {
		return 0, mapErr(err)
	}

	res := <-reply
	if res.Err != nil {
		return 0, mapErr(res.Err)
	}
	return res.NewCoins, nil
}
```

> Sau thay đổi này, `u.repo.GetPlacementByID`, `u.repo.GetMapCodeByID` và lời gọi `GetItemByID` trong delete **không còn ai dùng** → xem mục Cleanup.

### 1.3 Handler — đọc `map_code` từ query — `delivery/…handler.go`

```go
func (h *EditorHandler) DeletePlacement(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	placementID := ctx.Param("id")
	if placementID == "" {
		ctx.Error(apperror.BadRequest("Thiếu ID vật phẩm cần xóa", nil))
		return
	}

	mapCode := ctx.Query("map_code")
	if mapCode == "" {
		ctx.Error(apperror.BadRequest("Thiếu map_code", nil))
		return
	}

	newCoins, err := h.usecase.DeletePlacement(ctx.Request.Context(), userID.(string), mapCode, placementID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[gin.H]{
		Success: true,
		Data:    gin.H{"new_coins": newCoins},
	})
}
```

### 1.4 Frontend — gửi kèm `map_code`

`frontend/src/features/game/services/editor.service.ts`:

```ts
export function deletePlacement(id: string, mapCode: string) {
  return http.delete<DeletePlacementResultDto>(
    `/editor/place/${id}?map_code=${encodeURIComponent(mapCode)}`
  )
}
```

`frontend/src/features/game/systems/editorSystem.ts` trong `confirmDelete`:

```ts
const res = await editorService.deletePlacement(placementId, this.mapCode)
```

### 1.5 Cập nhật test cho khớp chữ ký mới
Trong `usecase/editor_usecase_test.go`, `TestDeletePlacement_DoubleClick_Concurrently` gọi:

```go
coins, err := uc.DeletePlacement(ctx, userID, "village_adventure", placementID)
```

(Test này pre‑insert placement vào DB rồi tạo `RoomManager` mới; `loadFromDB` sẽ nạp cả `prices` lẫn placement vào `byID` nên vẫn chạy đúng.)

---

## FIX 2 (P1) — Shutdown không được đóng writer trước khi actor drain xong

### Vấn đề
`RoomManager.Shutdown()` đóng `actor.cmds` rồi **gọi ngay** `writer.Close()` (đóng `writer.in`). Nếu còn command trong buffer, goroutine `run()` vẫn đang `m.dirty <- …` **sau khi** `writer.in` đã đóng → panic `send on closed channel`, và mất các op cuối.

### Sửa — `map_actor.go`: báo hiệu `run()` đã xong
Ở cuối `run()` (ngay sau `close(m.outbound)`):

```go
	close(m.outbound) // stop broadcastLoop
	close(m.done)     // báo cho RoomManager: đã drain hết, không còn dirty send
}
```

(`m.done` đã được khởi tạo ở Fix 1.1.)

### Sửa — `room_manager.go`: đợi mọi actor xong rồi mới đóng writer

```go
func (rm *RoomManager) Shutdown() {
	rm.mu.Lock()
	actors := make([]*MapActor, 0, len(rm.actors))
	for _, a := range rm.actors {
		actors = append(actors, a)
	}
	rm.mu.Unlock()

	// 1) Ngừng nhận lệnh
	for _, a := range actors {
		close(a.cmds)
	}
	// 2) Đợi từng run() drain hết -> đảm bảo không còn ai ghi vào writer.in
	for _, a := range actors {
		<-a.done
	}
	// 3) Bây giờ mới an toàn đóng writer + flush batch cuối
	rm.writer.Close()
}
```

Sau fix này, bài "shutdown drain" trong `docs/test_phase_3.md` (mục 3) mới thật sự pass ổn định dưới `-race -count=20`.

---

## FIX 3 (P2) — Chốt hợp đồng ví trước khi có nguồn coin khác

### Bối cảnh
`CmdCredit` đã có + được test (`TestMapActor_CmdCredit`) nhưng **chưa nối với gì**. Hiện an toàn vì không module nào ghi `characters.coins` cho player đang resident (`character` chỉ ghi lúc tạo; `leaderboard` dùng `score`; chưa có combat). Nhưng writer flush bằng `coins = $1` **tuyệt đối** — ngày thêm feature thưởng coin mà ghi thẳng DB, giá trị sẽ bị đè âm thầm.

### Sửa — expose một cửa duy nhất trên `RoomManager`

`room_manager.go`:

```go
import "errors"

// CreditCoins cộng/trừ coin cho một character ĐANG resident trong map,
// đi qua actor để giữ bất biến ví. delta < 0 để trừ.
func (rm *RoomManager) CreditCoins(ctx context.Context, mapCode, characterID string, delta int) (int, error) {
	a, err := rm.Actor(mapCode)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, errors.New("map not found")
	}

	reply := make(chan CmdResult, 1)
	if err := a.SendCmd(Cmd{
		Kind:   CmdCredit,
		CharID: characterID,
		Coins:  delta,
		Reply:  reply,
	}); err != nil {
		return 0, err
	}
	res := <-reply
	return res.NewCoins, res.Err
}
```

### Quy tắc (ghi vào ARCHITECTURE_GUIDE.md để không ai quên)
> Với một character **có thể đang ở trong map** (resident), **mọi** thay đổi coin phải đi qua `RoomManager.CreditCoins(...)` (hoặc luồng place/delete). **Không** module nào được `UPDATE characters SET coins ...` trực tiếp cho player resident — nếu không, lần flush write‑behind kế tiếp sẽ ghi đè giá trị RAM lên DB và "nuốt" số coin vừa ghi thẳng.

> Lưu ý: `CmdJoin` set ví bằng giá trị DB tại thời điểm join. Nên credit khi player chưa resident thì flush xuống DB trước, lần join sau sẽ đọc lại đúng — vẫn nhất quán.

---

## CLEANUP — code chết Tier 1 cần xoá

Sau khi chuyển hẳn sang actor (Tier 2), các method sau **0 lần dùng** trong production (chỉ còn nằm ở interface + repo impl + mock). Xoá cả 3 nơi: **interface `port/repository.go`**, **impl `repository/editor_repository.go`**, **mock trong 2 file test**.

Chết hoàn toàn:

```
PlaceItemWithTx
PlaceItemWithIDAndTx
DeductCoinsWithTx
AddCoinsWithTx
DeductCoinsGuardedWithTx
AddCoinsGuardedWithTx
DeletePlacementWithTx
InsertRewardEventWithTx     ← writer.go tự chạy raw INSERT reward_events, không qua method này
```

Chết **sau khi làm Fix 1** (delete không đọc DB nữa):

```
GetPlacementByID
GetMapCodeByID
```

Còn giữ lại (vẫn dùng): `GetMapIDByCode` (GetEditorData), `GetItemByID` (PlaceItem), `GetMapInfoByCode` (Actor), `GetPlacementsByMap` (GetEditorData + actor loadFromDB), `GetDecorationItems` (GetEditorData + actor loadFromDB prices).

Ngoài ra:
- `port/repository.go`: xoá `var ErrInsufficientCoins = errors.New("insufficient coins")` (sentinel Tier 1, giờ dùng `room.ErrInsufficientCoins`). Kiểm tra không còn ai import.
- Trong `mockEditorRepo` (`room/map_actor_test.go`) và mock của `usecase/editor_usecase_test.go`: xoá đúng các method tương ứng để mock vẫn khớp interface đã rút gọn (nếu để thừa method thì không lỗi compile, nhưng nên xoá cho sạch).

Gợi ý cách rà nhanh trước khi xoá:

```bash
for m in PlaceItemWithTx PlaceItemWithIDAndTx DeductCoinsWithTx AddCoinsWithTx \
         DeductCoinsGuardedWithTx AddCoinsGuardedWithTx DeletePlacementWithTx \
         InsertRewardEventWithTx GetPlacementByID GetMapCodeByID; do
  echo "== $m =="
  grep -rn "\b$m\b" backend --include=*.go | grep -v _test.go
done
```

Nếu output chỉ còn dòng định nghĩa (`func (r *EditorRepository)`) và dòng interface → xoá an toàn.

---

## Checklist "coi như placement đã xong"

- [ ] **Fix 1** làm đủ 5 mảnh (actor prices + handleDelete + usecase + handler + frontend + test). Kiểm thủ công: đặt 1 món rồi xoá **ngay lập tức** → xoá thành công, coin hoàn đúng (trước đây fail `NotFound`).
- [ ] **Fix 2**: `go test ./internal/module/editor/... -race -count=20` không panic; bài shutdown drain: kill sau khi bơm op, `SELECT` đủ.
- [ ] **Fix 3**: có `CreditCoins`; quy tắc ví ghi vào `ARCHITECTURE_GUIDE.md`.
- [ ] **Cleanup**: `grep` xác nhận 8–10 method trên = 0 usage rồi xoá; build lại sạch.
- [ ] Chạy lại 4 bài cốt lõi trong `docs/test_phase_3.md`: overspend / trùng ô / double‑delete / crash‑durability.
- [ ] Smoke 2 tab: A đặt → B thấy ngay; A xoá → B mất ngay; F5 hai bên khớp DB.

Xong 4 mục đầu thì phần **thêm decoration mới = chỉ tạo row `items` + `metadata_json`** (dùng `docs/phase 3/spritesheet-collision-tool.html` xuất `collides`, `frameWidth/frameHeight/frame`, `anchorX/anchorY`, `collision_x/y/w/h`, và `extra_colliders` nếu cần) — runtime đã metadata‑driven, không phải đụng code nữa.

---

## Ghi chú durability (theo thiết kế, không phải bug)
SIGKILL cứng giữa 2 lần flush vẫn mất ≤1s placement cuối (vốn đã broadcast) → vật thể ma tạm thời khi reload. Đây là đánh đổi Tier 2 mà chính doc chấp nhận. Nếu sau này muốn siết: giảm `every` của writer, hoặc thêm WAL/append‑only log trước batch. Không nằm trong phạm vi "xong" của placement.
