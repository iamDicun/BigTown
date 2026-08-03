# BigTown — Bug nhặt coin & sai dữ liệu ví khi đổi map (chẩn đoán + cách sửa)

> Phạm vi rà soát: nhánh `dev`, commit mới nhất `7aa95e6 feat: select coin in map and finalize placement`.
> Đọc chéo: `docs/phase 3/coin_pickup_system.md` và `docs/phase 3/placement_finalize_guide.md`.
> Kết luận ngắn: phần **placement/finalize đã đúng** như doc mô tả (Fix 1/2/3 đã áp dụng). Nhưng tính năng **nhặt coin mới** làm lộ ra một **chuỗi 3 lỗi ví** chỉ kích hoạt khi **đổi map**, cộng thêm **2 lỗ hổng thiết kế** của bản thân coin pickup (client-authoritative + không đồng bộ multiplayer). Số coin "sai khi đổi qua lại giữa các map" là hệ quả trực tiếp của chuỗi lỗi ví này.

---

## 0. Bảng ưu tiên

| Mức | Lỗi | Triệu chứng người chơi thấy | File chính |
|-----|-----|------------------------------|-----------|
| **P0** | Actor nạp ví từ **DB thẳng** (write‑behind trễ ~1s), bỏ qua ví "live" | Nhặt coin ở map A, sang map B thấy coin **tụt về mốc cũ** / mất số vừa nhặt | `room/map_actor.go` (`getOrLoadWallet`, `CmdJoin`) |
| **P0** | `onlineCoins` (cache ví xuyên map) bị **xoá mỗi lần flush** | Sai xảy ra ngẫu nhiên, đúng lúc trùng nhịp flush 1s | `room/writer.go` (`flush` gọi `EvictOnlineCoins`) |
| **P1** | `OnPlayerLeave` **fire 2 lần** khi warp → double `CmdLeave` | Ví bị flush 2 lần, `residents` âm; race thêm với join đích | `realtime/usecase/room_usecase.go` (`WarpPlayer`) + `transport/centrifuge.go` (`OnDisconnect`) |
| **P1** | Coin **client‑authoritative**: client tự chọn `coin_type` (giá tới 100), không `coin_id`, không chống trùng | Ai cũng spam được `gold` → **coin vô hạn** | `editor/delivery/handler.go`, `usecase.ClaimCoinPickup`, `systems/coinPickupSystem.ts` |
| **P2** | Coin spawn **thuần client** → mỗi client thấy coin khác nhau; thiếu broadcast `coin_picked` (doc §1 có mô tả nhưng chưa code) | Multiplayer không thấy chung coin | `systems/coinPickupSystem.ts` |

Ba dòng đầu là nguyên nhân trực tiếp của "sai coin khi đổi map". Hai dòng cuối là nợ kỹ thuật của coin pickup nên xử lý sớm vì đang mở cửa exploit.

---

## 1. Vì sao sai — kiến trúc ví hiện tại

Số coin "live" của một character khi đang online nằm ở **3 nơi** và chúng có thể lệch nhau:

1. `RoomManager.onlineCoins[charID]` — cache ví **xuyên map** (nguồn duy nhất mang giá trị qua ranh giới warp).
2. `MapActor.wallets[charID]` — ví trong RAM **của riêng từng map**; bị `delete` khi rời map.
3. `characters.coins` (DB) — được ghi **write‑behind, trễ tới ~1s** (`writer.every = 1s`).

Khi đổi map, ví map cũ bị xoá, ví map mới phải được **dựng lại**. Điểm chết nằm ở chỗ *dựng lại từ nguồn nào*.

### 1.1 Lỗi P0‑A: `getOrLoadWallet` nạp thẳng từ DB, bỏ qua ví live

`room/map_actor.go`:

```go
func (m *MapActor) getOrLoadWallet(charID string) (int, error) {
	coins, ok := m.wallets[charID]
	if ok {
		return coins, nil
	}
	// lazy load từ DB — ĐÂY LÀ VẤN ĐỀ: DB trễ tới 1s
	dbCoins, err := m.charReader.GetCoins(context.Background(), charID)
	...
}
```

Có **2 đường async độc lập** cùng chạm actor đích sau khi warp:

- **Đường WS**: subscribe room mới → `JoinRoom` → `OnPlayerJoin` → `CmdJoin` (đặt ví từ `onlineCoins`).
- **Đường HTTP**: người chơi vừa spawn đã overlap coin → `POST /editor/coin-pickup` → `CreditCoins` → `CmdCredit`.

**Không có gì đảm bảo `CmdJoin` chạy trước `CmdCredit`.** Nếu `CmdCredit` chạy trước:

```
onlineCoins[char] = 200 (đúng, mang từ map cũ)
DB.coins          = 150 (write-behind của map cũ chưa đáp)

CmdCredit tới trước CmdJoin:
  getOrLoadWallet: wallets[char] trống → đọc DB = 150 (CŨ!)
  wallets[char] = 150 + delta  → SAI, mất 50 coin của map trước
```

Sau đó `CmdJoin` set `wallets = c.Coins` (đè tiếp) càng làm rối. Đây chính là "sai khi đổi map".

### 1.2 Lỗi P0‑B: `onlineCoins` bị xoá mỗi lần flush

`room/writer.go`, cuối `flush()`:

```go
// Evict online cache after database changes are fully committed
for charID := range latestCoins {
	w.rm.EvictOnlineCoins(charID)
}
```

Eviction chạy cho **mọi** character có op trong batch — kể cả người **vẫn đang chơi** và đang nhặt coin. Cửa sổ hỏng:

```
op1 pickup: SetOnlineCoins(175), enqueue flush1(175)
op2 pickup: SetOnlineCoins(200), enqueue flush2(200)   // sang batch sau
writer commit flush1 → DB=175, rồi EvictOnlineCoins(char)  // xoá luôn 200 đang chờ!
→ onlineCoins TRỐNG, nhưng giá trị đúng là 200 (flush2 chưa đáp)
→ nếu warp+join lúc này: OnPlayerJoin thấy onlineCoins trống → seed = DB = 175  → MẤT 25
```

Bản chất: eviction post‑commit chỉ an toàn nếu không có op mới hơn đang bay. Với người chơi đang active, luôn có op mới hơn → invariant vỡ. `onlineCoins` không nên bị xoá theo nhịp flush; nó chỉ nên xoá khi người chơi **thật sự offline hẳn**.

### 1.3 Lỗi P1‑C: warp fire `OnPlayerLeave` hai lần

`realtime/usecase/room_usecase.go` — `WarpPlayer` gọi `LeaveRoom` với `clientID = ""`:

```go
if _, _, err := u.store.LeaveRoom(ctx, roomID, character.ID, ""); err != nil { ... }
for _, l := range u.listeners {
	_ = l.OnPlayerLeave(ctx, roomID, character.ID)   // luôn fire
}
```

Trong `actor_room_store.go`, `LeaveRoom` xoá theo `clientID`:

```go
if clients, ok := gr.Clients[characterID]; ok {
	delete(clients, clientID)          // clientID="" → không khớp client thật
	if len(clients) > 0 {              // client thật vẫn còn → removed=false
		reply <- result{&cp, false}
		return
	}
}
```

Nên: `WarpPlayer` **không thật sự rời** store (removed=false) nhưng **vẫn fire** `OnPlayerLeave` → actor cũ `CmdLeave` lần 1 (flush + `delete wallets`). Sau đó socket đóng → `OnDisconnect` → `handleLeaveRoom(clientID thật)` → removed=true → `OnPlayerLeave` **lần 2** → `CmdLeave` lần 2 → `residents[char]` xuống **âm**, flush lần nữa. Thứ tự lần‑2 này với `CmdJoin` ở map đích không được đảm bảo → thêm một nguồn race ví.

---

## 2. Cách sửa (P0 trước)

Ý tưởng chốt: **`onlineCoins` là ví live duy nhất khi online; DB chỉ là bản lưu nguội.** Mọi lần dựng lại ví (join / lazy‑load) phải đọc `onlineCoins` trước, DB sau. Và `onlineCoins` chỉ xoá khi player rời **toàn bộ** kết nối.

### Fix P0‑A + P0‑B — Actor đọc ví "live", CmdJoin seed non‑destructive, bỏ evict theo flush

**Bước 1 — Cho actor một "coin resolver" trỏ về ví live.** `room/map_actor.go`:

```go
// CoinResolver trả về số coin LIVE của character (ưu tiên onlineCoins, fallback DB).
type CoinResolver func(ctx context.Context, charID string) (int, error)

type MapActor struct {
	// ... giữ nguyên ...
	charReader port.CharacterReader
	repo       port.EditorRepository
	coins      CoinResolver // <-- thêm
}
```

Sửa `NewMapActor` nhận thêm `coins CoinResolver` và gán vào struct:

```go
func NewMapActor(
	mapID, mapCode string, mapW, mapH, tileSize int,
	charReader port.CharacterReader, repo port.EditorRepository,
	dirty chan persistOp, publisher port.RoomPublisher,
	coins CoinResolver, // <-- thêm tham số cuối
) *MapActor {
	m := &MapActor{
		// ... giữ nguyên các field cũ ...
		coins: coins,
	}
	go m.run()
	go m.broadcastLoop(publisher)
	return m
}
```

**Bước 2 — `getOrLoadWallet` đọc ví live thay vì DB thẳng:**

```go
func (m *MapActor) getOrLoadWallet(charID string) (int, error) {
	if coins, ok := m.wallets[charID]; ok {
		return coins, nil // ví của actor này đã seed → là nguồn chuẩn tại chỗ
	}
	// Chưa seed: hỏi ví LIVE (onlineCoins trước, DB sau) — không bao giờ đọc DB "trần"
	live, err := m.coins(context.Background(), charID)
	if err != nil {
		return 0, err
	}
	m.wallets[charID] = live
	return live, nil
}
```

**Bước 3 — `CmdJoin` seed *chỉ khi chưa có* (non‑destructive):** tránh đè giá trị mới hơn nếu một `CmdCredit` đã lỡ chạy trước.

```go
case CmdJoin:
	if _, ok := m.wallets[c.CharID]; !ok {
		m.wallets[c.CharID] = c.Coins
	}
	m.residents[c.CharID]++
```

> Sau bước 1–3: dù `CmdCredit` chạy trước `CmdJoin`, `getOrLoadWallet` đã lấy đúng `onlineCoins`(=200) chứ không phải DB(=150). Race thứ tự không còn làm sai số.

**Bước 4 — Truyền resolver khi tạo actor.** `room/room_manager.go`, trong `Actor(...)`:

```go
	a = NewMapActor(
		mapInfo.ID, mapCode,
		mapInfo.Width*mapInfo.TileSize, mapInfo.Height*mapInfo.TileSize, mapInfo.TileSize,
		rm.charReader, rm.repo, rm.writer.in, rm.publisher,
		rm.GetCoins, // <-- ví live: GetCoins đã ưu tiên onlineCoins rồi tới DB
	)
```

`rm.GetCoins` sẵn có đúng chữ ký `(ctx, charID) (int, error)` và đã ưu tiên `onlineCoins`:

```go
func (rm *RoomManager) GetCoins(ctx context.Context, characterID string) (int, error) {
	rm.mu.RLock()
	coins, ok := rm.onlineCoins[characterID]
	rm.mu.RUnlock()
	if ok {
		return coins, nil
	}
	return rm.charReader.GetCoins(ctx, characterID)
}
```

**Bước 5 — Bỏ eviction theo flush.** `room/writer.go`, xoá đoạn cuối `flush()`:

```go
	// XOÁ HẲN đoạn này:
	// for charID := range latestCoins {
	// 	w.rm.EvictOnlineCoins(charID)
	// }
```

**Bước 6 — Xoá `onlineCoins` đúng thời điểm: khi player rời *toàn bộ* kết nối.** Thêm đếm connection ở `RoomManager` và evict có debounce để **không** vỡ khi warp (warp là leave‑rồi‑join, `liveConns` chạm 0 tạm thời).

`room/room_manager.go`:

```go
type RoomManager struct {
	// ... giữ nguyên ...
	onlineCoins map[string]int
	liveConns   map[string]int // <-- thêm: số kết nối đang online của mỗi char
}

func NewRoomManager(...) *RoomManager {
	rm := &RoomManager{
		// ... giữ nguyên ...
		onlineCoins: make(map[string]int),
		liveConns:   make(map[string]int),
	}
	rm.writer = NewWriter(db, rm)
	return rm
}

func (rm *RoomManager) OnPlayerJoin(ctx context.Context, roomID, characterID string, coins int) error {
	rm.mu.Lock()
	if _, exists := rm.onlineCoins[characterID]; !exists {
		rm.onlineCoins[characterID] = coins
	}
	rm.liveConns[characterID]++          // <-- đếm lên
	currentCoins := rm.onlineCoins[characterID]
	rm.mu.Unlock()

	a, err := rm.Actor(roomID)
	if err != nil || a == nil {
		return err
	}
	return a.SendCmd(Cmd{Kind: CmdJoin, CharID: characterID, Coins: currentCoins})
}

func (rm *RoomManager) OnPlayerLeave(ctx context.Context, roomID, characterID string) error {
	rm.mu.Lock()
	if rm.liveConns[characterID] > 0 {
		rm.liveConns[characterID]--
	}
	gone := rm.liveConns[characterID] <= 0
	if gone {
		delete(rm.liveConns, characterID)
	}
	rm.mu.Unlock()

	if gone {
		// Debounce: warp = leave→join, liveConns chạm 0 vài trăm ms rồi lại lên.
		// Chỉ evict nếu sau grace vẫn không có kết nối nào (offline thật).
		go rm.scheduleEvict(characterID)
	}

	a, err := rm.Actor(roomID)
	if err != nil || a == nil {
		return err
	}
	return a.SendCmd(Cmd{Kind: CmdLeave, CharID: characterID})
}

func (rm *RoomManager) scheduleEvict(characterID string) {
	time.Sleep(3 * time.Second) // grace cho warp/reconnect
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.liveConns[characterID] == 0 { // vẫn offline sau grace
		delete(rm.onlineCoins, characterID)
	}
}
```

> Lưu ý an toàn: kể cả nếu **không bao giờ** evict, tính đúng đắn vẫn được giữ (onlineCoins chỉ là cache, DB luôn hội tụ nhờ mỗi mutation đều enqueue flush). Eviction ở đây **chỉ để chống rò rỉ RAM**, nên chọn được grace thoải mái mà không sợ vỡ ví. Nhớ `import "time"` trong `room_manager.go`.

**Bước 7 — cập nhật test cho chữ ký `NewMapActor` mới** (`room/map_actor_test.go`, 3 chỗ): thêm resolver DB đơn giản ở cuối:

```go
resolver := func(ctx context.Context, id string) (int, error) {
	return charReader.GetCoins(ctx, id)
}
actor := NewMapActor("map-1", "village", 1000, 1000, 16, charReader, repo, dirty, publisher, resolver)
```

### Fix P1‑C — Warp không được fire `OnPlayerLeave` trùng với `OnDisconnect`

Cách sạch nhất: để **duy nhất `OnDisconnect`/`OnUnsubscribe`** chịu trách nhiệm phát `LeaveRoom` + `OnPlayerLeave`; `WarpPlayer` **không tự fire leave** nữa (client sẽ đóng socket ngay sau warp, disconnect tự dọn map cũ). Sửa `WarpPlayer` trong `realtime/usecase/room_usecase.go`:

```go
func (u *RoomUsecase) WarpPlayer(ctx context.Context, roomID, userID, destMap string, destX, destY int) (*WarpDestination, error) {
	if _, err := u.maps.GetMapByCode(ctx, destMap); err != nil {
		return nil, err
	}
	// KHÔNG LeaveRoom / KHÔNG fire OnPlayerLeave ở đây nữa.
	// Việc rời map cũ (store + wallet flush) do OnDisconnect của socket cũ đảm nhiệm,
	// tránh double-leave và residents âm.
	return &WarpDestination{MapCode: destMap, X: destX, Y: destY}, nil
}
```

Nếu muốn giữ dọn dẹp chủ động trong warp (để flush ví sớm, không chờ socket đóng) thì phải **truyền đúng `clientID`** và **bỏ** nhánh fire ở `OnDisconnect` cho chính client đó — nhưng cách "để một cửa duy nhất" ở trên đơn giản và ít race hơn. Chốt một trong hai, đừng để cả hai cùng fire.

> Phòng thủ thêm (rẻ, nên có): trong `map_actor.go` chặn underflow `residents`:
> ```go
> case CmdLeave:
> 	if m.residents[c.CharID] > 0 {
> 		m.residents[c.CharID]--
> 	}
> 	if m.residents[c.CharID] <= 0 {
> 		if coins, ok := m.wallets[c.CharID]; ok {
> 			m.dirty <- persistOp{Kind: opFlushWallet, CharID: c.CharID, NewCoins: coins}
> 			delete(m.wallets, c.CharID)
> 		}
> 		delete(m.residents, c.CharID)
> 	}
> ```

---

## 3. Nợ kỹ thuật của coin pickup (P1/P2) — nên xử lý sớm

### 3.1 Coin đang client‑authoritative → exploit coin vô hạn

Hiện `POST /editor/coin-pickup` chỉ nhận `{map_code, coin_type}` và cộng tiền theo `coin_type` do **client tự khai**:

```go
switch coinType {
case "gri":  delta = 5
case "gold": delta = 100 // client chỉ cần spam "gold"
...
}
newCoins, _ := u.rooms.CreditCoins(ctx, mapCode, charInfo.ID, delta)
```

Không có `coin_id`, không có registry coin nào đã spawn, không dedup, không rate‑limit → bất kỳ ai cũng gọi lặp `{coin_type:"gold"}` để +100 vô hạn. Điều này **mâu thuẫn** với chính `coin_pickup_system.md` §4 ("server kiểm soát `CmdCredit` và lưu trữ đồng xu nào đã nhặt").

**Hướng sửa tối thiểu (server‑authoritative spawn registry per map):**

- Cho `MapActor` giữ `coinsOnMap map[string]SpawnedCoin` (id → {type, x, y}) và spawn server‑side theo tick, broadcast `coin_spawned`.
- Client chỉ **render** coin theo broadcast, khi overlap thì gửi **`coin_id`** (không gửi type/giá).
- Handler đổi payload sang `{map_code, coin_id}`; actor tra `coinsOnMap[coin_id]`: nếu tồn tại → xoá khỏi registry (dedup), tính `delta` **từ type do server lưu**, `CmdCredit`, rồi broadcast `coin_picked{coin_id}`.

Payload/handler mới (phác thảo):

```go
// handler.go
var input struct {
	MapCode string `json:"map_code" binding:"required"`
	CoinID  string `json:"coin_id"  binding:"required"` // đổi từ coin_type
}
newCoins, err := h.usecase.ClaimCoinPickup(ctx.Request.Context(), userID.(string), input.MapCode, input.CoinID)
```

```go
// map_actor.go — nhánh CmdClaimCoin (mới)
case CmdClaimCoin:
	sc, ok := m.coinsOnMap[c.CoinID]
	if !ok { // đã bị người khác nhặt hoặc không tồn tại → không cộng, chống trùng
		c.Reply <- CmdResult{Err: ErrNotFound}
		break
	}
	delete(m.coinsOnMap, c.CoinID)
	delta := coinValue(sc.Type) // giá do SERVER quyết, không phải client
	coins, err := m.getOrLoadWallet(c.CharID)
	if err != nil { c.Reply <- CmdResult{Err: err}; break }
	newCoins := coins + delta
	m.wallets[c.CharID] = newCoins
	c.Reply <- CmdResult{NewCoins: newCoins}
	m.outbound <- map[string]any{"type": "coin_picked", "coinId": c.CoinID}
	m.dirty <- persistOp{Kind: opFlushWallet, CharID: c.CharID, NewCoins: newCoins,
		CoinDelta: delta, EventType: "coin_pickup"}
```

> Nếu chưa muốn làm registry ngay, tối thiểu hãy **rate‑limit** endpoint (vd token bucket theo charID) và log `reward_events` để phát hiện bất thường — nhưng đây chỉ là vá tạm, không đóng được exploit.

### 3.2 Coin spawn thuần client → multiplayer không đồng bộ

`coinPickupSystem.ts` spawn coin bằng `Phaser.Math.Between` ở **mỗi client** một cách độc lập → 2 người trong cùng map **thấy coin ở chỗ khác nhau**, và không có broadcast `coin_picked` (doc §1 có vẽ sequence nhưng code chưa hiện thực). Khi chuyển sang server‑authoritative (3.1), client bỏ `spawnRandomCoin`/`spawnInitialCoins`, thay bằng lắng nghe `coin_spawned`/`coin_picked` từ Centrifuge (giống cách đã làm cho `decoration_placed`/`decoration_deleted` trong `GameScene.ts`).

Xử lý overlap phía client khi đã server‑authoritative:

```ts
private async handlePickup(coin: Phaser.GameObjects.Sprite) {
  const body = coin.body as Phaser.Physics.Arcade.StaticBody
  if (body) body.enable = false
  const coinId = coin.getData('id') as string
  // client prediction: ẩn ngay
  this.playPickupFx(coin)
  try {
    const res = await editorService.claimCoinPickup(this.mapCode, coinId) // gửi coin_id
    window.dispatchEvent(new CustomEvent('game:placementDone', { detail: { newCoins: res.new_coins } }))
  } catch {
    // server từ chối (coin đã bị người khác nhặt) → có thể hoàn lại sprite nếu muốn
  }
}
```

---

## 4. Ghi chú nhỏ (không chặn "xong", nên dọn khi tiện)

- **Số frame animation coin**: `coinPickupSystem.ts` dùng `generateFrameNumbers(..., { start: 0, end: 3 })` (4 frame — khớp `spr_coin_strip4` và width 64/16=4). Doc `coin_pickup_system.md` §2.1 vẫn ghi `end: 4` (5 frame). Sửa doc cho khớp code, kẻo người sau nạp nhầm 5 frame.
- **`spawnRandomCoin` dùng CustomEvent đồng bộ** để hỏi `game:checkOccupied` và kỳ vọng `callback` được gọi ngay trong lúc dispatch. Chỉ đúng nếu listener chạy đồng bộ; nếu sau này listener thành async thì `overlapPlacement` luôn `false`. Nên đổi sang hàm truy vấn trực tiếp thay vì event round‑trip. (Sẽ tự biến mất nếu chuyển spawn sang server.)
- **`mapCode` trong `CoinPickupSystem`** được chụp từ `bootstrap.map_code` lúc khởi tạo — đúng, vì mỗi lần đổi map là một `GameScene` mới. Không cần sửa; chỉ lưu ý đừng cache system này xuyên scene.

---

## 5. Checklist nghiệm thu

- [ ] **P0‑A/B**: đứng ở `winter`, nhặt vài coin, warp sang `dark_village` **ngay lập tức** rồi warp về → số coin **giữ nguyên cộng dồn**, không tụt về mốc cũ. Lặp round‑trip 10 lần liên tục, đúng nhịp ~1s (trùng flush) vẫn đúng.
- [ ] **P0‑A**: cố tình overlap coin ở map đích ở frame đầu (spawn cạnh coin) để `CmdCredit` đua trước `CmdJoin` → ví vẫn đúng.
- [ ] **P1‑C**: `go test ./internal/module/editor/... -race -count=20` không panic; log không còn `residents` âm; ví chỉ flush 1 lần mỗi lần rời map (kiểm qua `reward_events`).
- [ ] **P1 (3.1)**: gọi lặp `/editor/coin-pickup` cùng một coin/khống → chỉ cộng **một lần** (hoặc 0 nếu coin không tồn tại); không thể spam `gold`.
- [ ] **P2 (3.2)**: 2 tab cùng map thấy **chung** vị trí coin; A nhặt → B thấy coin biến mất (nhận `coin_picked`).
- [ ] Chạy lại 4 bài cốt lõi placement trong `docs/test_phase_3.md` (overspend / trùng ô / double‑delete / crash‑durability) — không hồi quy.

---

## 6. Thứ tự triển khai gợi ý

1. **Fix P0‑A + P0‑B** cùng lúc (chúng bổ trợ nhau: resolver live + bỏ evict theo flush). Đây là thứ khiến "đổi map sai coin" biến mất.
2. **Fix P1‑C** (một cửa leave duy nhất) để hết double‑flush/residents âm.
3. **3.1 server‑authoritative coin** để đóng exploit, rồi **3.2** để đồng bộ multiplayer.
4. Dọn ghi chú §4.

> Gợi ý an toàn khi refactor ví: mọi thay đổi coin cho player **có thể đang resident** vẫn phải đi qua actor (`CreditCoins`/place/delete) đúng như `placement_finalize_guide.md` Fix 3 đã chốt — đừng thêm đường `UPDATE characters SET coins` trực tiếp cho coin pickup.
