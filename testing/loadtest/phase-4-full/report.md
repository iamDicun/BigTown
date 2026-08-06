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
| 3 | **Placement** | ⏳ chưa chạy | — | < 800ms REST / < 1500ms delivery | — |
| 4 | **Bootstrap** | ⏳ chưa chạy | — | < 600ms p95 / < 1200ms p99 | — |

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

## 4. Chi tiết: Placement Load Test

⏳ *Đang chờ kết quả...*

*(sẽ cập nhật: place_rest_ms p95, place_deliver_ms p95, place_error, place_occupied, fanout ratio)*

---

## 5. Chi tiết: Bootstrap Load Test

⏳ *Đang chờ kết quả...*

*(sẽ cập nhật: bootstrap_ms p95/p99, http_req_failed, bootstrap_npc_count)*

---

## 6. Tổng kết Phase 4

| Chỉ số | Chat | Movement | Placement | Bootstrap |
|:---|:---:|:---:|:---:|:---:|
| **p95 chính** | ~900ms ✅ | ~285ms ✅ | — | — |
| **Lỗi** | 0.006% ✅ | 0% ✅ | — | — |
| **Room leak** | 0 ✅ | N/A (channel) | — | — |
| **Server ổn định** | ✅ | ✅ | — | — |

> **Ghi chú:** Kết quả sẽ được cập nhật dần sau mỗi lần chạy test. Các test còn lại (movement, placement, bootstrap) chạy lần lượt trên cùng hạ tầng.
