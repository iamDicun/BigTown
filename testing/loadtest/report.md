# Báo Cáo Tổng Kết Load Test & Tối Ưu Hóa Hiệu Năng Realtime

Tài liệu này tổng hợp và so sánh kết quả load test qua các Phase tối ưu hóa và các Chiến lược đo lường khác nhau của dự án BigTown.

---

## 1. Bản Đồ So Sánh Các Giai Đoạn (Phases Comparison)

| Chỉ số / Metric | Phase 1: Baseline | Phase 2: Actor per-room | Phase 3: Tick + RAM Cache | Phase 4: Full Test (Grafana) |
| :--- | :--- | :--- | :--- | :--- |
| **Trạng thái** | Đã hoàn thành Str. 1, 2, 3 | Đã hoàn thành Str. 1 & 2 | Đã hoàn thành Str. 2 & 3 (ĐẠT ✅) | **Hoàn tất 4/4** |
| **Cơ chế lưu trữ** | `MemoryRoomStore` (Global Mutex) | `ActorRoomStore` (Actor model) | `ActorRoomStore` + gom Ticker + RAM Cache | `ActorRoomStore` + Ticker + RAM Cache |
| **Độ trễ Chat (p95)** | 88ms (Local) \| 1626ms (Render) \| 1850ms (Grafana) | 190ms (Local) \| 1602ms (Render) | 718ms (Render) \| ~960ms (Grafana ✅) | **~900ms** (Grafana ✅) |
| **Độ trễ Move RPC (p95)** | 7ms (Local) \| 155ms (Render) | 10ms (Local) \| 165ms (Render) | 165ms (Render ✅) | **~285ms** (Grafana ✅) |
| **Độ trễ Placement (p95)** | — | — | — | ~1000ms REST / ~1800ms deliver ⚠️ |
| **Độ trễ Bootstrap (p95)** | — | — | — | ~890ms p95 / ~1080ms p99 ⚠️ |
| **Rò rỉ kênh (Room Leak)**| 0 (PASS) | 0 (PASS) | 0 (PASS) | 0 (PASS ✅) |
| **Tần suất GC / Heap** | ~0.7 runs/s / ~45MB | ~0.6 runs/s / ~45MB | ~0.6 runs/s / ~45MB | ~0.2 runs/s / ~32MB ✅ |
| **Goroutines (idle→peak)** | — | — | — | 95→337→125 (không leak ✅) |

---

## 2. Chi Tiết Kết Quả Theo Chiến Lược (Strategies)

### Giai Đoạn 1: Baseline (Phase 1)
*   **Strategy 1 (Local):** p95 Chat 88ms | p95 Move 7ms. [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-1-local/report.html)
*   **Strategy 2 (Render):** p95 Chat 1626 ms (FAIL) | p95 Move 155 ms (PASS). [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-2-online/report.html)
*   **Strategy 3 (Grafana):** p95 Chat ~1850 ms (FAIL). [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-3-grafana/report.html)

---

### Giai Đoạn 2: Actor per-room Migration (Phase 2)
*   **Strategy 1 (Local):** p95 Chat 190 ms | p95 Move 10 ms. [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-2-actor/strategy-1-local/report.html)
*   **Strategy 2 (Render):** p95 Chat 1602 ms (FAIL) | p95 Move 165 ms (PASS). [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-2-actor/strategy-2-online/report.html)

---

### Giai Đoạn 3: Tick-based Broadcast + Async DB + RAM Cache (Phase 3)

#### **Strategy 2: Local to Render (k6 Local → Render Backend)**
*   **Mục đích:** Đánh giá hiệu năng sau khi áp dụng gộp phát tin (100ms), ghi Postgres bất đồng bộ kết hợp lưu RAM Cache thông tin người chơi.
*   **Kết quả đo lường thành công vượt bậc:**
    *   **Chat test:** Độ trễ **p95 giảm ngoạn mục xuống còn 718 ms**, vượt qua mốc yêu cầu kỹ thuật (&lt;1000ms). HTTP Request duration p95 giảm xuống chỉ còn **700 ms**, trung vị (median) đạt **230 ms**.
    *   **Nguyên nhân thành công:** Bộ đệm RAM cache cho hàm `GetByUserID` đã triệt tiêu hoàn toàn 2 câu lệnh SQL đồng bộ trên mỗi request gửi chat. DB Connection Pool không còn hiện tượng xếp hàng chờ kết nối.
    *   **Movement test:** 326,707 RPC gửi, p95 RPC giữ nguyên ở mức **165 ms (PASS)**, trong khi số gói tin broadcast truyền tải trên mạng lưới được Centrifuge tối ưu gộp giảm đi 10 lần.
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-3-tick-broadcast/strategy-2-online/report.html)

#### **Strategy 3: Grafana Cloud (Grafana k6 Cloud → Render Backend)**
*   **Mục đích:** Đo lường hiệu năng khi chịu tải phân tán toàn cầu từ Ohio (Mỹ) tới Render (Singapore) và Postgres (Nhật Bản).
*   **Kết quả đo lường đạt mốc lý tưởng:**
    *   **Chat test (Độ trễ p95):** Đạt khoảng **~960 ms** (PASS, đạt tiêu chuẩn &lt; 1000ms bất chấp khoảng cách địa lý nửa vòng Trái Đất).
    *   **Tỷ lệ thành công:** Đạt **99.99%** (chỉ lỗi 2 request trên tổng số 16,271 request do mất kết nối TCP mạng WAN - Connection Reset).
     *   **Movement test:** Hoạt động gom nhịp phát tin hoạt động hoàn hảo, client nhận gói gộp trơn tru.

---

### Giai Đoạn 4: Full Load Test — 4 loại test (Phase 4)

#### **Chat Test: PASS ✅**
*   **Cấu hình:** 100 VU, 10 room, 5 phút, Grafana Cloud (Ohio → Render Singapore → Postgres Nhật)
*   **Kết quả:**
    *   p95 delivery: **~900ms** (PASS, ngưỡng < 1000ms)
    *   p99 delivery: ~1050ms | Median: ~560ms
    *   Tổng: ~16,300 chat gửi, ~163,000 chat nhận
    *   Tỉ lệ lỗi: **0.006%** (1 lỗi WS connect do mất TCP WAN)
    *   cross_room_leak: **0** (PASS)
    *   Server: Goroutines 95→337→125 (không leak), Heap 10.8→32→22MB, GC pause < 0.2ms
*   **Báo cáo chi tiết:** [phase-4-full/report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-4-full/report.html)

#### **Movement, Placement, Bootstrap Test: ⏳**
Đang chờ chạy và cập nhật kết quả.

#### **Movement Test: PASS ✅**
*   **Cấu hình:** 100 VU, 10 room, 5 phút, Grafana Cloud (Ohio → Render Singapore)
*   **Kết quả:**
    *   p95 RPC latency: **~285ms** (PASS, ngưỡng < 500ms). Chỉ bằng 57% ngưỡng.
    *   p99 RPC latency: ~310ms | Median: ~245ms
    *   Tổng: ~300,000 RPC gửi, **0 lỗi, 0 WS error**
    *   Broadcast `room_state`: ~9k players/sec gộp qua tick 100ms, rất đều
    *   `position_correction` hội tụ từ 2,800 → 300/3s: server điều chỉnh vị trí ổn định
*   **Báo cáo chi tiết:** [phase-4-full/report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-4-full/report.html)

#### **Placement, Bootstrap Test: ⏳**
Đang chờ chạy và cập nhật kết quả.

#### **Placement Test: ⚠️ SÁT NGƯỠNG**
*   **Cấu hình:** 100 VU, 10 room, place mỗi 1s, Grafana Cloud (Ohio → Render Singapore)
*   **Giai đoạn ổn định (2 phút đầu):**
    *   p95 REST place: **~1000ms** (FAIL sát, ngưỡng < 800ms)
    *   p95 delivery: **~1800ms** (FAIL, ngưỡng < 1500ms) | Median: ~710ms (OK)
    *   0 lỗi, WS ổn định
*   **Giai đoạn suy giảm (3 phút sau):** 292 timeout place, 52 WS error, delivery spike 139s
*   **Nguyên nhân:** Actor serialize GHI theo map — 10 place/s/room actor tạo hàng đợi. Đề xuất: tách coin deduct atomic SQL, write-behind batch, rate limiter broadcast.
*   **Báo cáo chi tiết:** [phase-4-full/report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-4-full/report.html)

#### **Bootstrap Test: ⚠️ FAIL p95, PASS p99**
*   **Cấu hình:** `ramping-arrival-rate` 10→50→200→0 RPS, max ~100 VU, Grafana Cloud (Ohio → Render SG)
*   **Kết quả:**
    *   p95 ở đỉnh 166 RPS: **~890ms** (FAIL, ngưỡng < 600ms)
    *   p99 ở đỉnh: **~1080ms** (PASS, ngưỡng < 1200ms) | Median: ~560ms
    *   **0 lỗi, 0 fail, 100% checks pass** — hệ thống không sập
    *   p95 ở tải thấp (~100 RPS): chỉ ~450ms (PASS), dưới đỉnh ~100 RPS rất khỏe
*   **Nguyên nhân:** DB read maps + NPC spawns mỗi request gây nghẽn ở RPS cao. Có thể cache map metadata.
*   **Báo cáo chi tiết:** [phase-4-full/report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-4-full/report.html)
