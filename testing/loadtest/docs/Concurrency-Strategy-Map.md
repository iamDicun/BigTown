# BigTown — Bản đồ chọn chiến lược Concurrency theo Use Case

Tài liệu tham chiếu khi refactor: mỗi loại công việc trong hệ dùng **cơ chế đồng bộ nào** và **vì sao**. Nguyên tắc xuyên suốt: *khớp cơ chế với **bản chất** của công việc, không dùng một búa cho mọi đinh.*

---

## 1. Heuristic chọn nhanh (đọc từ trên xuống, dừng ở dòng khớp đầu tiên)

1. Công việc **không đụng state dùng chung** (stateless, mỗi request độc lập)? → **Không cần gì**, để goroutine-per-request của Gin lo. Trần đồng thời thật đã bị pool DB (25) chặn sẵn.
2. Đụng **state dùng chung, sửa nhanh, phần lớn là đọc**? → **`sync.RWMutex`** (hoặc `sync.Mutex` nếu ghi nhiều).
3. State **stateful, nóng, và mỗi thực thể độc lập nhau** (mỗi room, mỗi trận đấu…)? → **Actor / goroutine sở hữu state, giao tiếp qua channel** (bỏ lock).
4. **Fan-out nhiều việc đồng thời đập vào một tài nguyên có TRẦN** (DB connection, API ngoài, file)? → **Semaphore / worker pool** để chặn số lượng đồng thời. ⟵ *đúng như bạn nói*.
5. Nhiều goroutine **cùng lúc gọi một phép tính/đọc TỐN KÉM y hệt nhau**? → **`singleflight`** để gộp thành một lần, chia sẻ kết quả.
6. Trong **một** request cần chạy **nhiều việc con độc lập song song** rồi chờ tất cả? → **`errgroup`** (có giới hạn + gom lỗi).
7. Dữ liệu **đọc-là-chính, đổi hiếm** (map metadata, character options…)? → **Cache trong RAM** (+ RWMutex hoặc `sync.Map`/otter), không hỏi DB mỗi lần.
8. Sự kiện **tần suất cao cần gộp lại** (movement broadcast)? → **Batch theo tick** (Ticker trong actor).
9. Cần **chống lạm dụng / bảo vệ tài nguyên khỏi client** (spam chat/move)? → **Rate limit / throttle per-client**.
10. Mọi công việc có I/O? → luôn **truyền `context` có timeout/cancel**, đừng nuốt cancellation.

> **Goroutine KHÔNG phải tài nguyên khan hiếm** — nó rẻ. Ta *pool cái khan hiếm* (connection DB, quota API), không pool goroutine. Đây là lý do worker pool chỉ dùng cho mục (4).

---

## 2. Bản đồ theo use case thực tế của BigTown

| Use case (file) | Bản chất | Chiến lược | Vì sao / Ghi chú |
|---|---|---|---|
| **REST đọc**: get users, leaderboard, `characters/me`, `characters/options`, chat history, bootstrap | Stateless, đọc DB | **Goroutine-per-request** (mặc định) + **cache** cho phần đọc-là-chính | Không cần lock. `characters/options`, bootstrap, map metadata → cache (mục 4.1). |
| **REST ghi**: register, login, create character, send chat | Stateless, ghi DB (transaction) | **Goroutine-per-request** + transaction | Đã đúng. `login/register` chạy **bcrypt** (nặng CPU cố ý) — nếu login-storm, cân nhắc semaphore chặn số bcrypt đồng thời (mục 4.3). |
| **AuthMiddleware blacklist check** (mỗi request protected) | Đọc DB **mỗi** request | **Cache TTL ngắn** cho token blacklist | Hiện mỗi API protected là 1 query blacklist → hotspot ẩn. Cache 30–60s giảm tải rõ rệt (mục 4.6). |
| **Teams JWKS fetch** (`microsoft_token_verifier.go`) | Đọc HTTP ngoài, đổi hiếm | Đã có **RWMutex cache + TTL** ✓ → thêm **`singleflight`** | Khi cache hết hạn, N request đồng thời cùng bắn ra Microsoft (thundering herd). singleflight gộp còn 1 (mục 4.2). |
| **Room state**: join/leave/move/snapshot (`memory_store.go`) | Stateful, nóng, mỗi room độc lập | **Actor per-room** (bỏ global mutex) | Chính là migration đã viết. Room A không được chặn room B. |
| **`GetMapByCode` trên hot path move/join** | Đọc DB **mỗi tick di chuyển** | **Cache map metadata** (actor giữ sẵn) | ⚠️ Hiện gọi DB mỗi move (10/s/player). Đọc-là-chính, đổi hiếm → cache. Cách sạch nhất: room actor nạp 1 lần lúc tạo, giữ trong state (mục 4.1). |
| **Movement broadcast** (publish mỗi RPC) | Sự kiện tần suất cao | **Batch theo tick** trong actor | Gộp nhiều move trong 100ms thành 1 event → giảm tải Centrifuge (mục 4.4). |
| **Chat persist** (INSERT mỗi tin) | Ghi DB, theo *nhịp gửi* | Thường **để nguyên**; nếu nhiều room → **semaphore** ghi | Fan-out chỉ ở khâu broadcast (Centrifuge lo), còn INSERT chỉ 1/lần-gửi. Chỉ chặn khi số room lớn làm INSERT dồn pool (mục 4.5). |
| **Persist vị trí định kỳ** (roadmap) | Fan-out N room → DB | **Semaphore / worker pool** | Kinh điển của mục (4): 50 room flush cùng lúc sẽ vỡ pool 25 nếu không chặn (mục 4.5). |
| **NPC/enemy AI tick** (roadmap) | Stateful theo room | **Nằm trong actor per-room** | Actor tự chạy tick, xử lý NPC của room nó — không cần pool riêng. |
| **Gọi Microsoft Graph / webhook / thông báo ngoài** (roadmap) | I/O ngoài, có quota | **Semaphore + timeout + retry** | Mọi call ra ngoài phải bị chặn số đồng thời và có timeout. |
| **Bootstrap** đọc nhiều nguồn trong 1 request (nếu mở rộng) | Nhiều việc con độc lập | **`errgroup`** | Đọc map + character + config song song rồi chờ tất cả, gom lỗi (mục 4.7). |
| **Centrifuge connection lifecycle** (đọc/ghi mỗi conn) | Do thư viện quản | **Không đụng** goroutine; chỉ **sửa context** | Đang truyền `context.Background()` trong handler → nên propagate + timeout (mục 4.8). |
| **Chống spam move/chat từ 1 client** | Bảo vệ tài nguyên | **Rate limit per-client** | Client lỗi/độc có thể bắn move/chat dồn dập; chặn ở transport (mục 4.9). |


| Chỉ số quan sát được                                                    | Nguyên nhân                                                                               | Giải pháp                                                                                 |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `cross_room_leak > 0`                                                   | Rò channel — Centrifuge publish sai room hoặc logic room isolation hỏng                   | **Dừng mọi việc**, sửa correctness trước. Không quan tâm hiệu năng khi còn leak > 0.      |
| `1 core CPU ~100%, các core khác rảnh`                                  | Global mutex contention — mọi goroutine xếp hàng sau 1 lock                               | Chuyển sang **Actor per-room migration**.                                                 |
| `go_goroutines` tăng đơn điệu, không giảm sau test                      | Goroutine leak — actor không dừng khi room rỗng hoặc connection không được cleanup        | Kiểm tra vòng đời actor (`stopIfEmpty`) và context propagation (4.8).                     |
| `pg query rate` cao bất thường trên hot path `move`                     | `GetMapByCode` gọi DB mỗi tick di chuyển (≈10 lần/s/player)                               | Cache map metadata (4.1) — actor giữ `MapInfo` trong state.                               |
| `pg WaitCount` / `WaitDuration` tăng                                    | Nghẽn pool DB (25 connections) — quá nhiều `INSERT`/query đồng thời                       | Dùng semaphore để chặn fan-out vào DB (4.3/4.5) hoặc batch insert.                        |
| Tỉ lệ giao `< 100%` nhiều + `ws write errors` phía server               | Backpressure — client đọc chậm hoặc WebSocket buffer đầy, server phải drop message        | Tick-based broadcast (4.4): gom movement mỗi **100 ms**, publish một lần thay vì mỗi RPC. |
| `p95 delivery` tăng dần theo thời gian (cùng chiều với `go_goroutines`) | Goroutine leak (xác nhận chéo với chỉ số trên)                                            | Áp dụng giải pháp xử lý goroutine leak như trên.                                          |
| `p95 delivery` tăng + `1 core CPU 100%`                                 | Global mutex contention (xác nhận chéo)                                                   | Áp dụng Actor per-room như trên.                                                          |
| `p95 delivery` tăng + `pg WaitCount` tăng                               | Nghẽn DB (xác nhận chéo)                                                                  | Áp dụng giải pháp giảm tải DB như trên.                                                   |
| `ws_connect_errors` cao hoặc connection rớt dần                         | Token hết hạn giữa lúc test, `CheckOrigin` chặn, hoặc hết file descriptor                 | Tăng `-ttl`, cấu hình `ORIGIN`, đặt `ulimit -n 65535`.                                    |
| `chat_delivery_ms p95` không đổi sau khi sửa                            | Hoặc sửa nhầm chỗ, hoặc nút thắt thực sự nằm ở tầng khác (mạng, TLS, Centrifuge internal) | Dừng tối ưu, đo lại và xác định đúng bottleneck trước khi tiếp tục.                       |

---

## 3. Quy tắc worker pool / semaphore — nói cho chính xác

Trực giác của bạn đúng: **worker pool/semaphore chỉ để chặn use case đụng vào tài nguyên có trần.** Chốt lại thành quy tắc dùng được:

- **Dùng khi**: bạn *fan-out* (tạo nhiều việc song song) và các việc đó cùng đập vào một thứ **có giới hạn cứng**: connection DB (bạn giới hạn 25), rate limit API bên thứ ba, số file handle, băng thông đĩa.
- **KHÔNG dùng để**: "tiết kiệm goroutine" — goroutine rẻ, không phải thứ cần tiết kiệm. Đặt pool lên hot path chỉ để giới hạn goroutine sẽ tự tạo ra hàng đợi và **thêm** độ trễ.
- **Đại lượng để chọn size**: size của semaphore ≈ **trần của tài nguyên đích**, không phải số CPU. Ghi DB → size ≤ (pool DB trừ phần dành cho request thường). API ngoài → size ≤ quota cho phép.
- **Semaphore (channel) thường đủ**, gọn hơn "worker pool" đầy đủ. Chỉ dựng pool n-worker + job queue khi cần thêm: hàng đợi có giới hạn, retry, back-pressure tường minh.

---

## 4. Deep-dive các case không hiển nhiên (kèm skeleton)

### 4.1 Cache map metadata — bỏ `GetMapByCode` khỏi hot path

Vấn đề: `MovePlayer`/`JoinRoom` gọi DB đọc map mỗi lần. Map đổi rất hiếm → đọc-là-chính. Hai cách:

**Cách gọn nhất (khuyến nghị): actor giữ map trong state.** Khi tạo room actor, nạp map một lần và giữ luôn — move sau đó không chạm DB:

```go
type GameRoom struct {
    // ... như cũ ...
    MapInfo port.MapInfo // nạp 1 lần lúc tạo actor, dùng cho bounds/spawn
}
```

`MovePlayer` đọc `gr.MapInfo.MaxPixelX()` ngay trong actor, zero DB call/tick.

**Cách tổng quát: cache đọc-là-chính có TTL** (dùng cho cả REST bootstrap):

```go
type mapCache struct {
    mu   sync.RWMutex
    data map[string]port.MapInfo
    exp  map[string]time.Time
    src  port.MapReader
    ttl  time.Duration
}
func (c *mapCache) GetMapByCode(ctx context.Context, code string) (port.MapInfo, error) {
    c.mu.RLock()
    if m, ok := c.data[code]; ok && time.Now().Before(c.exp[code]) {
        c.mu.RUnlock(); return m, nil
    }
    c.mu.RUnlock()
    m, err := c.src.GetMapByCode(ctx, code) // + singleflight (4.2) nếu muốn
    if err != nil { return m, err }
    c.mu.Lock(); c.data[code] = m; c.exp[code] = time.Now().Add(c.ttl); c.mu.Unlock()
    return m, nil
}
```

### 4.2 `singleflight` — chống thundering herd cho JWKS (và map loader)

Khi cache JWKS hết hạn, nhiều login Teams đồng thời cùng bắn ra Microsoft. Gộp thành một:

```go
import "golang.org/x/sync/singleflight"

var g singleflight.Group
func (v *MicrosoftTokenVerifier) fetchKeysOnce(ctx context.Context) (map[string]*rsa.PublicKey, error) {
    val, err, _ := g.Do("jwks:"+v.tenantID, func() (any, error) {
        return v.fetchKeys(ctx) // chỉ 1 goroutine thực sự gọi ra ngoài
    })
    if err != nil { return nil, err }
    return val.(map[string]*rsa.PublicKey), nil
}
```

### 4.3 / 4.5 Semaphore chặn fan-out vào tài nguyên có trần

Mẫu chung (dùng cho persist vị trí, batch ghi chat, login-storm bcrypt, call API ngoài):

```go
sem := make(chan struct{}, 10) // = trần cho phép của tài nguyên đích

func guarded(fn func() error) error {
    sem <- struct{}{}            // xin slot (block nếu đã đủ 10 cái chạy)
    defer func() { <-sem }()
    return fn()
}
// vd: 50 room actor gọi guarded(persistToDB) => tối đa 10 INSERT đồng thời, không vỡ pool 25.
```

### 4.6 Cache token blacklist (giảm 1 query/request protected)

AuthMiddleware đang hỏi DB blacklist mỗi request. Cache TTL ngắn:

```go
// key = jti/tokenID; TTL ~ min(30s, thời gian còn lại của token).
// Đọc RWMutex trước, miss mới hỏi DB. Khi logout, chủ động set entry "blacklisted".
```

### 4.7 `errgroup` — song song các việc con trong 1 request

```go
import "golang.org/x/sync/errgroup"
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(4) // chặn số goroutine con
var mp port.MapInfo; var ch entity.Character
g.Go(func() error { var e error; mp, e = maps.GetMapByCode(ctx, code); return e })
g.Go(func() error { var e error; ch, e = chars.GetByUserID(ctx, uid); return e })
if err := g.Wait(); err != nil { return err } // gom lỗi + tự cancel phần còn lại
```

### 4.8 Sửa context propagation ở transport Centrifuge

Hiện các handler dùng `context.Background()`:

```go
// thay vì context.Background():
ctx, cancel := context.WithTimeout(client.Context(), 3*time.Second)
defer cancel()
roomUsecase.JoinRoom(ctx, roomID, client.UserID(), client.ID())
```

Client disconnect → query DB đang chạy bị hủy theo, không giữ connection vô ích.

### 4.9 Rate limit per-client (chống spam move/chat)

```go
// token bucket theo client.UserID(): vd move ≤ 20/s, chat ≤ 5/s.
// Vượt -> trả ErrorLimitExceeded ở OnRPC / từ chối ở tầng chat usecase.
// Dùng golang.org/x/time/rate.Limiter một cái/ client, dọn khi disconnect.
```

---

## 5. Thứ tự refactor đề xuất (khớp với vòng đo trước/sau)

Chạy load test bản hiện tại trước để có baseline, rồi áp dụng theo thứ tự **lợi ích/công sức giảm dần**, đo lại sau mỗi bước để thấy đúng cái gì cải thiện:

1. **Cache map metadata / actor giữ map (4.1)** — bỏ DB call khỏi hot path move. Rẻ, lợi lớn, dễ thấy trên dashboard (pg query rate của move sụt).
2. **Actor per-room (migration)** — bỏ global mutex. Xem CPU trải đều nhiều core + p95 giảm.
3. **Sửa context propagation (4.8)** — chống giữ connection rác khi client rớt.
4. **Cache blacklist (4.6)** — giảm 1 query/request protected.
5. **singleflight JWKS (4.2)** — chỉ quan trọng khi có nhiều login Teams đồng thời.
6. **Batch movement broadcast (4.4)** — khi publish rate là điểm nghẽn Centrifuge.
7. **Semaphore cho persist/ghi (4.3/4.5)** — khi (và chỉ khi) thêm persist định kỳ hoặc thấy `pg WaitCount` tăng.
8. **Rate limit (4.9)** — bảo vệ, làm khi chuẩn bị mở cho người dùng thật.

> Mẹo đọc kết quả: mỗi bước nên cải thiện **một** chỉ số cụ thể trên dashboard (mục 3 của LoadTest-Guide). Nếu đổi code mà chỉ số kỳ vọng không nhúc nhích, hoặc bạn sửa nhầm chỗ, hoặc nút thắt thật nằm ở nơi khác — dừng lại đo trước khi làm tiếp.

