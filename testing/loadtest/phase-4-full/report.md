# Phase 4: Full Load Test — Grafana Cloud Strategy 3

**Ngày:** 2026-08-06 | **Backend:** Render (Singapore) | **Postgres:** Aiven (Nhật Bản)
**k6 Load Zone:** `amazon:us:columbus` (Ohio, Mỹ) | **Test Run ID:** 1296862
**Script version:** `Strategy3-LoadTest-Guide-update.md` (4 scripts: chat, movement, placement, bootstrap)

---

## 1. Tổng quan kết quả

| # | Test | Trạng thái | p95 chính | Threshold | Đánh giá |
|:---|:---|:---:|:---|:---|:---|
| 1 | **Chat** | ✅ PASS | ~900ms delivery | < 1000ms | ĐẠT |
| 2 | **Movement** | ✅ PASS | ~285ms RPC | < 500ms RPC | ĐẠT |
| 3 | **Placement** | ⚠️ FAIL (sát) | ~1000ms REST / ~1800ms deliver | < 800ms / < 1500ms | SÁT NGƯỠNG |
| 4 | **Bootstrap** | ⚠️ FAIL p95 / PASS p99 | ~890ms p95 / ~1080ms p99 | < 600ms p95 / < 1200ms p99 | p95 vượt ngưỡng |

---

## 2. Chi tiết: Chat Load Test

### 2.1 Cấu hình test

| Tham số | Giá trị |
|:---|:---|
| Executor | `constant-vus` |
| VUs | 100 |
| Duration | 5 phút |
| Rooms | 10 (10 VU/room) |
| Send interval | 2 giây/lần |
| Load zone | `amazon:us:columbus` (Ohio) |
| Backend | `bigtown-1.onrender.com` (Singapore) |
| DB | Aiven Postgres (Nhật Bản) |

### 2.2 Kết quả client-side (k6)

| Metric | Giá trị | Ngưỡng | Kết quả |
|:---|:---|:---|:---:|
| **Tổng chat gửi** (`chat_sent`) | ~16,300 | — | — |
| **Tổng chat nhận** (`chat_received`) | ~163,000 | — | — |
| **p95 delivery** (gửi → nhận) | **~900ms** ổn định | < 1000ms | ✅ PASS |
| **p99 delivery** | ~1050ms | — | — |
| **Median delivery** | ~560ms | — | — |
| **HTTP POST chat p95** | ~800-870ms | — | — |
| **WS connect errors** | 1 | < 5 | ✅ PASS |
| **cross_room_leak** | 0 | == 0 | ✅ PASS |
| **chat POST 201** checks | 100% (trừ 1 lỗi ở 16:33:36) | > 99% | ✅ PASS |
| **Tỉ lệ giao** (fanout) | ~100% | ~100% | ✅ PASS |

### 2.3 Bất thường ghi nhận

- **16:33:36 UTC** (~23:33:36 VN): 1 VU bị **WS connect error**, gây **1 request chat POST fail** và kéo mean delivery lên 3512ms (p95 45s) trong 3 giây đó. Nguyên nhân: mất kết nối TCP WAN (Ohio → Singapore), VU reconnect ngay lập tức (có thêm 1 "subscribed to room" ở cùng timestamp). Không ảnh hưởng tổng thể — 1 lỗi / 16,300 request = **99.994% thành công**.

### 2.4 Server-side metrics (Render container)

| Metric | Idle | Peak (dưới tải) | Sau test | Đánh giá |
|:---|:---|:---|:---|:---|
| **Goroutines** | ~95 | **337** | ~125 | ✅ Ổn định, không leak |
| **Heap Memory** | ~10.8 MB | **~32 MB** | ~22 MB | ✅ GC hoạt động tốt, không OOM |
| **GC runs/s** | ~0.05 | **0.2-0.25** | ~0.05 | ✅ Thấp, khỏe |
| **GC pause (avg)** | 0.05-0.15ms | **0.07-0.2ms** | — | ✅ 2 spike 6.7ms & 9ms khi kết thúc test |
| **HTTP req/s** | ~0.1 | **~50 req/s** ổn định | ~0.1 | ✅ Phẳng, không bóp |

- **Không có goroutine leak**: goroutines về 125 sau test (trước test ~95, chênh ~30 là các connection đang đóng dần).
- **Không có memory leak**: heap giảm từ 32MB về 22MB sau test.
- **GC rất nhẹ**: 0.2 runs/s, pause trung bình < 0.2ms (2 spike lớn 6.7ms & 9ms xảy ra ở cuối test khi 100 VU đồng loạt ngắt kết nối, bình thường).

### 2.5 So sánh với các Phase trước

| Phase | Strategy | Chat p95 delivery | Đánh giá |
|:---|:---|:---|:---|
| Phase 1 (Baseline) | Grafana Cloud | **1850 ms** | ❌ FAIL |
| Phase 3 (Tick + RAM Cache) | Grafana Cloud | **~960 ms** | ✅ PASS |
| **Phase 4 (Hiện tại)** | **Grafana Cloud** | **~900 ms** | ✅ PASS |

p95 Phase 4 cải thiện nhẹ (~60ms) so với Phase 3, dao động ổn định trong khoảng 800-1011ms suốt 5 phút.

### 2.6 Kết luận Chat test

**PASS toàn diện.** Hệ thống xử lý 100 VU chat liên tục qua 10 room với độ trễ p95 dưới ngưỡng 1000ms, tỉ lệ lỗi 0.006%, không rò room, không leak goroutine/memory. Khoảng cách địa lý Ohio → Singapore → Nhật không gây suy giảm đáng kể.

---

## 3. Chi tiết: Movement Load Test ✅

### 3.1 Cấu hình test

| Tham số | Giá trị |
|:---|:---|
| Executor | `constant-vus` |
| VUs | 100 |
| Duration | 5 phút |
| Rooms | 10 (10 VU/room) |
| Move interval | 100ms/lần |
| Load zone | `amazon:us:columbus` (Ohio) |
| Backend | `bigtown-1.onrender.com` (Singapore) |

### 3.2 Kết quả client-side (k6)

| Metric | Giá trị | Ngưỡng | Kết quả |
|:---|:---|:---|:---:|
| **Tổng RPC gửi** (`move_rpc_sent`) | ~300,000 | — | — |
| **p95 RPC latency** | **~285ms** (276-297ms) | < 500ms | ✅ PASS |
| **p99 RPC latency** | ~310ms | — | — |
| **Median RPC latency** | ~245ms | — | — |
| **Min RPC latency** | ~216ms | — | — |
| **WS connect errors** | 0 | < 5 | ✅ PASS |
| **Checks (broadcast characterId)** | 100% pass | > 99% | ✅ PASS |
| **Subscribed to room** | 100/100 VU | — | ✅ PASS |
| **room_state broadcast** | ~25k-28k players/3s | — | Đều, ổn định |
| **position_correction** | Từ 2,800 → 300/3s | — | Hội tụ tốt |

### 3.3 Bất thường

- **Không có lỗi nào.** 100 VU hoạt động xuyên suốt 5 phút không rớt kết nối, không RPC error, không timeout.
- `move_correction` giảm dần từ ~2,800/3s xuống ~300/3s: server điều chỉnh vị trí ban đầu (spawn 100,100 → phân tán dần), sau đó ổn định.
- `move_broadcast` (room_state) duy trì ~25k-28k players mỗi cửa sổ 3 giây — tương đương ~9k players/sec gộp trong tick 100ms, rất đều.

### 3.4 So sánh với các Phase trước

| Phase | Strategy | p95 RPC Latency | Đánh giá |
|:---|:---|:---|:---|
| Phase 1 (Baseline) | Render (local k6) | **155 ms** | ✅ PASS |
| Phase 2 (Actor) | Render (local k6) | **165 ms** | ✅ PASS |
| Phase 3 (Tick) | Render (local k6) | **165 ms** | ✅ PASS |
| **Phase 4 (Hiện tại)** | **Grafana Cloud (Ohio→SG)** | **~285 ms** | ✅ PASS |

Chênh ~120ms so với local→Render là hoàn toàn do độ trễ mạng WAN Ohio↔Singapore (~200ms RTT). Bản thân RPC xử lý phía server vẫn rất nhanh.

### 3.5 Kết luận Movement test

**PASS xuất sắc.** p95 RPC chỉ 285ms — chỉ bằng 57% ngưỡng cho phép (500ms). Không lỗi, không rớt kết nối, broadcast room_state gộp đều đặn. Hệ thống xử lý 1,000 RPC/s qua 10 room từ cách nửa vòng trái đất vẫn mượt.

---

## 4. Chi tiết: Placement Load Test ⚠️

### 4.1 Cấu hình test

| Tham số | Giá trị |
|:---|:---|
| Executor | `constant-vus` |
| VUs | 100 |
| Duration | 5 phút |
| Rooms | 10 (10 VU/room) |
| Place interval | 1 giây/lần |
| Load zone | `amazon:us:columbus` (Ohio) |
| Backend | `bigtown-1.onrender.com` (Singapore) |
| Yêu cầu đặc biệt | Đã nạp coin (100M/character) + restart backend |

### 4.2 Kết quả client-side (k6)

Test có 2 giai đoạn rõ rệt:

#### Giai đoạn 1: Ổn định (17:01:30 – 17:03:24, ~2 phút)

| Metric | Giá trị | Ngưỡng | Kết quả |
|:---|:---|:---|:---:|
| **Tổng place gửi** | ~11,500 | — | — |
| **p95 REST place** | **~1000ms** (920-1175ms) | < 800ms | ⚠️ FAIL (sát) |
| **Median REST place** | ~700ms | — | — |
| **p95 delivery** (place→broadcast) | **~1800ms** (1700-2200ms) | < 1500ms | ⚠️ FAIL |
| **Median delivery** | **~710ms** | — | ✅ Nhanh |
| **Place error** | 0 | < 10 | ✅ PASS |
| **WS connect errors** | 0 | < 5 | ✅ PASS |
| **Tỉ lệ broadcast** | ~2,700-3,500/3s (10x fanout) | — | Ổn định |

#### Giai đoạn 2: Suy giảm (17:03:24 – 17:06:57, ~3 phút)

| Metric | Giá trị |
|:---|:---|
| **Place error (timeout 60s)** | 292 |
| **WS connect errors** | 52 |
| **REST response** | Hàng loạt timeout 60 giây |
| **Delivery spike** | p95 lên tới 139,511ms (139 giây!) |

### 4.3 Phân tích

**Nguyên nhân gốc:** Actor model serialize mọi thao tác GHI theo từng map. Với 100 VU × 1 place/s = ~100 place/s tổng. Mỗi room actor xử lý ~10 place/s tuần tự (coin deduct → insert placement → reply → broadcast → write-behind DB). Đây là điểm nghẽn tự nhiên của kiến trúc actor.

**Tại sao median nhanh (710ms) nhưng p95 cao (1800ms):**
- Median ~710ms: khi actor không bận, request được xử lý ngay
- p95 ~1800ms: khi nhiều VU cùng room gửi cùng lúc, request xếp hàng chờ actor. Mỗi place mất ~70ms → 10 place xếp hàng = 700ms + network RTT ~500ms = ~1200ms. Đôi khi hàng dài hơn.
- Delivery delay thêm ~800ms so với REST (1800 vs 1000ms) do broadcast qua Centrifuge bị backpressure khi tải cao.

**Giai đoạn 2 (suy giảm):** Sau ~2 phút, hàng đợi actor tích tụ → REST bắt đầu timeout (60s). Đồng thời WS connections bắt đầu rớt (52 errors) do broadcast backlog quá lớn. Đây là hiệu ứng domino: actor chậm → broadcast chậm → WS timeout → reconnect → thêm tải.

### 4.4 Kết luận Placement test

**⚠️ SÁT NGƯỠNG — cần tối ưu thêm.** p95 REST (1000ms) vượt ngưỡng 800ms ~25%, p95 delivery (1800ms) vượt ngưỡng 1500ms ~20%. Median vẫn tốt (710ms). Hệ thống sập sau ~2 phút tải liên tục do actor serialization.

**Hướng cải thiện tiềm năng:**
- Tách coin deduct ra khỏi actor path (dùng atomic SQL UPDATE)
- Dùng write-behind batch cho placement (gom nhiều insert vào 1 transaction)
- Thêm rate limiter hoặc queue riêng cho broadcast để tránh backpressure lan sang REST

---

## 5. Chi tiết: Bootstrap Load Test ⚠️

### 5.1 Cấu hình test

| Tham số | Giá trị |
|:---|:---|
| Executor | `ramping-arrival-rate` (open model) |
| Stages | 10→50 RPS (30s) → 200 RPS (1m ramp) → 200 RPS (2m hold) → 0 (30s cooldown) |
| Max VUs | ~100 |
| Load zone | `amazon:us:columbus` (Ohio) |
| Backend | `bigtown-1.onrender.com` (Singapore) |
| Ghi chú | `npc_spawns` = 0 (loadtest maps không seed NPC) |

### 5.2 Kết quả client-side (k6)

| Metric | Tải thấp (< 50 RPS) | Đỉnh (~166 RPS) | Ngưỡng | Kết quả |
|:---|:---|:---|:---|:---:|
| **p95 bootstrap** | ~450ms | **~890ms** | < 600ms | ⚠️ FAIL |
| **p99 bootstrap** | ~800ms | **~1080ms** | < 1200ms | ✅ PASS |
| **Median bootstrap** | ~375ms | ~560ms | — | — |
| **Min bootstrap** | ~361ms | ~375ms | — | — |
| **http_req_failed** | 0% | **0%** | < 1% | ✅ PASS |
| **bootstrap_bad** | 0 | **0** | < 10 | ✅ PASS |
| **Checks pass** | 100% | **100%** | > 99% | ✅ PASS |
| **npc_spawns** | 0 | 0 | — | Map loadtest ko có NPC |

### 5.3 Phân tích

**Đây là điểm cần chú ý:** Bootstrap p95 vượt ngưỡng ~50% ở tải đỉnh (890ms vs 600ms). Tuy nhiên p99 vẫn dưới 1200ms và tỉ lệ lỗi = 0%.

**Đặc điểm của bootstrap endpoint:**
- Đây là request HTTP thuần, không dùng WS, không có actor
- Server phải: parse JWT → load character → load map metadata → load NPC spawns → tạo reply
- Với RAM cache cho `GetByUserID`, phần character load đã được tối ưu (tương tự chat test)
- Map metadata load và NPC spawns load vẫn query DB mỗi lần

**Diễn biến theo tải:**
| RPS | p95 | Ghi chú |
|:---|:---|:---|
| ~10-50 | **~450ms** | PASS thoải mái |
| ~100 | **~450-500ms** | PASS |
| ~166 (đỉnh) | **~880-910ms** | FAIL – DB read bắt đầu nghẽn |
| Cooldown | **~410ms** | Hồi phục ngay |

**Kết luận:** Bootstrap chịu được ~100 RPS với p95 < 600ms. Ở 166 RPS, DB read (maps + NPC spawns) trở thành bottleneck. Median vẫn thấp (560ms) chứng tỏ hầu hết request nhanh, chỉ một số bị nghẽn DB pool. **Có thể cần cache map metadata** nếu muốn p95 dưới 600ms ở 200 RPS. Tuy nhiên 166 RPS bootstrap tương đương rất nhiều người chơi login đồng thời — đây là kịch bản "thundering herd" hiếm gặp.

---

## 6. Tổng kết Phase 4

| Chỉ số | Chat | Movement | Placement | Bootstrap |
|:---|:---:|:---:|:---:|:---:|
| **p95 chính** | ~900ms ✅ | ~285ms ✅ | ~1000ms / ~1800ms ⚠️ | ~890ms p95 / ~1080ms p99 ⚠️ |
| **Lỗi** | 0.006% ✅ | 0% ✅ | 292 + 52 WS ⚠️ | 0% ✅ |
| **Room leak** | 0 ✅ | N/A | N/A | N/A |
| **Server ổn định** | ✅ | ✅ | ⚠️ Sập sau 2 phút | ✅ |

> Bootstrap p95 vượt ngưỡng 600ms → **~890ms** ở đỉnh 166 RPS. p99 ~1080ms vẫn PASS. Nguyên nhân: DB read maps + NPC spawns mỗi request. Có thể cache map metadata nếu cần. 0 lỗi tuyệt đối.
