# Phase 4: Full Load Test — Grafana Cloud Strategy 3

**Ngày:** 2026-08-06/07 | **Backend:** Render (Singapore) | **Postgres:** Aiven (Nhật Bản)
**k6 Load Zone:** `amazon:us:columbus` (Ohio, Mỹ) | **Script:** Strategy3-LoadTest-Guide-update.md

---

## 1. Tổng quan kết quả

| # | Test | Trạng thái | p95 chính | Threshold | Đánh giá |
|:---|:---|:---:|:---|:---|:---|
| 1 | **Chat** | ✅ PASS | ~900ms delivery | < 1000ms | ĐẠT |
| 2 | **Movement** | ✅ PASS | ~285ms RPC | < 500ms | ĐẠT XUẤT SẮC |
| 3 | **Placement** | ⚠️ SÁT NGƯỠNG | ~1200ms REST / ~5000ms deliver | < 800ms / < 1500ms | Actor bottleneck |
| 4 | **Bootstrap** | ✅ PASS | ~340ms p95 / ~600ms p99 | < 600ms / < 1200ms | ĐẠT (sau cache NPC) |

---

## 2. Chi tiết: Chat Load Test ✅

### 2.1 Cấu hình & kết quả

| Metric | Giá trị | Ngưỡng | Kết quả |
|:---|:---|:---|:---:|
| **p95 delivery** | **~900ms** | < 1000ms | ✅ |
| **p99 delivery** | ~1050ms | — | — |
| **Median delivery** | ~560ms | — | — |
| **WS connect errors** | 1 | < 5 | ✅ |
| **cross_room_leak** | 0 | == 0 | ✅ |
| **chat POST 201** | 99.994% | > 99% | ✅ |

### 2.2 Server-side (Render container)

| Metric | Idle | Peak | Sau test |
|:---|:---|:---|:---|
| Goroutines | ~95 | 337 | ~125 |
| Heap Memory | 10.8 MB | ~32 MB | ~22 MB |
| GC runs/s | ~0.05 | 0.2-0.25 | ~0.05 |
| GC pause | 0.05-0.15ms | 0.07-0.2ms | — |

**Kết luận:** PASS toàn diện. Không leak, không rò room. 100 VU chat qua 10 room, Ohio→Singapore→Nhật vẫn ổn định.

---

## 3. Chi tiết: Movement Load Test ✅

### 3.1 Kết quả

| Metric | Giá trị | Ngưỡng | Kết quả |
|:---|:---|:---|:---:|
| **p95 RPC latency** | **~285ms** | < 500ms | ✅ (57% ngưỡng) |
| **p99 RPC** | ~310ms | — | — |
| **Median RPC** | ~245ms | — | — |
| **WS errors** | **0** | < 5 | ✅ |
| **Checks pass** | 100% | > 99% | ✅ |

**Kết luận:** PASS xuất sắc. 1,000 RPC/s qua 10 room từ Ohio vẫn mượt. Broadcast `room_state` gộp ổn định ~9k players/s.

---

## 4. Chi tiết: Placement Load Test ⚠️

### 4.1 So sánh 3 phase

| | Phase 1 (gốc) | Phase 2 (+batch INSERT) | Phase 3 (+cache collides, batch DELETE) |
|:---|:---|:---|:---|
| **REST p95** | ~1000ms | ~1200ms | **~1150-1300ms** |
| **Delivery p95** | ~1800ms | ~5000-8000ms | **~4000-8000ms** |
| **WS errors** | 52 | 89 | **2** ✅ |
| **Test bị sập?** | Sau ~2 phút | Sau ~2 phút | **Không sập** ✅ |

### 4.2 Phân tích

- **REST không cải thiện**: Actor serialize 10 place/s/room vẫn là bottleneck. Cache collides + batch DELETE không đụng đến actor path.
- **WS ổn định hơn hẳn**: Batch INSERT/DELETE giảm writer flush time → bớt CPU contention → Centrifuge ít giật.
- **Không crash**: Test chạy đủ 5 phút, không timeout hàng loạt.

### 4.3 Tối ưu đã làm

| # | Fix | File | Hiệu quả |
|:---|:---|:---|:---|
| 1 | **Batch INSERT placement** (gom N→1 query) | `writer.go` | Giảm round-trip DB |
| 2 | **Cache `collides`** (bỏ DB query trong actor) | `map_actor.go` | Actor delete không chạm DB |
| 3 | **Batch DELETE + reward_events** | `writer.go` | Giảm round-trip DB |
| 4 | **Race condition fix** (editor load từ RAM) | `editor_usecase.go`, `map_actor.go` | Hết lệch RAM/DB |
| 5 | **Metrics backpressure** | `editor_metrics.go` | Theo dõi queue writer |

### 4.4 Local test (VN → Singapore) — xác nhận bottleneck

| Metric | Grafana Cloud (Ohio) | Local (VN) | Khác biệt |
|:---|:---|:---|:---|
| **REST p95** | ~1200ms | **1213ms** | 🔄 Gần giống hệt |
| **REST median** | ~870ms | **714ms** | ↓ 18% |
| **Delivery p95** | ~5000ms | **4368ms** | ↓ 13% |
| **WS errors** | 2 | 5 | Tương đương |

**REST p95 không đổi dù mạng gần hơn 4x — chứng minh bottleneck là server, không phải mạng.** CPU không max, heap 50MB, goroutine 350-450. Actor command queue mới là gốc rễ: 10 lệnh/s/room xử lý tuần tự, dù mỗi lệnh chỉ ~0.6ms CPU, nhưng đến không đều + Go scheduler 1 core → tích tụ queue → p95 ~1200ms.

**Hướng tiếp theo:** Muốn REST p95 < 800ms cần atomic SQL coin deduct (bỏ actor khỏi đường GHI).

---

## 5. Chi tiết: Bootstrap Load Test ✅

### 5.1 So sánh trước/sau cache NPC

| | Phase 1 (chưa cache NPC) | Phase 2 (cache NPC + preload) | Cải thiện |
|:---|:---|:---|:---|
| **p95 ở đỉnh** | **890ms** ❌ | **~340ms** ✅ | ↓ 62% |
| **p99 ở đỉnh** | ~1080ms | ~600ms | ↓ 44% |
| **Median ở đỉnh** | ~560ms | **~301ms** | ↓ 46% |
| **http_req_failed** | 0% | **0%** | ✅ |
| **Đỉnh throughput** | ~166 RPS | ~180-210 RPS | ↑ |

### 5.2 Kết quả chi tiết (Phase 2)

| Metric | Tải thấp (< 50 RPS) | Đỉnh (~200 RPS) | Ngưỡng | Kết quả |
|:---|:---|:---|:---|:---:|
| **p95** | ~380ms | **~340ms** | < 600ms | ✅ PASS |
| **p99** | ~800ms | **~600ms** | < 1200ms | ✅ PASS |
| **Median** | ~301ms | **~301ms** | — | Gần = network RTT |
| **Min** | ~290ms | ~290ms | — | — |

**Kết luận:** Cache NPC spawns trong RAM + preload startup đưa p95 từ 890ms xuống 340ms, vượt ngưỡng 600ms thoải mái. Bootstrap giờ là 100% in-memory sau request đầu tiên, 0 DB query.

---

## 6. Tổng kết Phase 4

| Chỉ số | Chat | Movement | Placement | Bootstrap |
|:---|:---:|:---:|:---:|:---:|
| **p95 chính** | ~900ms ✅ | ~285ms ✅ | ~1200ms / ~5000ms ⚠️ | ~340ms ✅ |
| **Lỗi** | 0.006% ✅ | 0% ✅ | ~3-7/3s (ổn định) | 0% ✅ |
| **WS ổn định** | ✅ (1 lỗi) | ✅ (0 lỗi) | ✅ (2 lỗi) | N/A |
| **Đã tối ưu** | RAM cache user | Tick broadcast | Batch write + cache collides | Cache NPC + preload |

### Các tối ưu đã deploy

| # | Tối ưu | Test ảnh hưởng | Kết quả |
|:---|:---|:---|:---|
| 1 | **RAM cache `GetByUserID`** | Chat, Movement | Chat p95 1850→900ms ✅ |
| 2 | **Tick broadcast 100ms (`room_state`)** | Movement | Gộp 90% gói tin ✅ |
| 3 | **Cache NPC spawns + preload** | Bootstrap | p95 890→340ms ✅ |
| 4 | **Batch INSERT placement** | Placement | Writer ổn định hơn |
| 5 | **Cache `collides` + batch DELETE** | Placement | WS errors 89→2 ✅ |
| 6 | **Editor load từ RAM (fix race)** | Placement | Hết lệch UI/DB |

### Còn lại

| Vấn đề | Nguyên nhân | Hướng fix |
|:---|:---|:---|
| Placement REST p95 ~1200ms | Actor serialize GHI, 1 core Render | Atomic SQL coin deduct |
| Placement delivery p95 ~5000ms | Broadcast backlog khi tải cao | Tách broadcast khỏi actor path |
