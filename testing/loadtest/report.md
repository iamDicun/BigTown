# Báo Cáo Tổng Kết Load Test & Tối Ưu Hóa Hiệu Năng Realtime

Tài liệu này tổng hợp và so sánh kết quả load test qua các Phase tối ưu hóa và các Chiến lược đo lường khác nhau của dự án BigTown.

---

## 1. Bản Đồ So Sánh Các Giai Đoạn (Phases Comparison)

| Chỉ số / Metric | Phase 1: Baseline (Chưa tối ưu) | Phase 2: Actor per-room | Phase 3: Tick Broadcast + RAM Cache |
| :--- | :--- | :--- | :--- |
| **Trạng thái (Status)** | **Đã hoàn thành Str. 1, 2, 3** | **Đã hoàn thành Str. 1 & 2** | **Đã hoàn thành Str. 2 & 3 (ĐẠT ✅)** |
| **Cơ chế lưu trữ** | `MemoryRoomStore` (Global Mutex) | `ActorRoomStore` (Actor model) | `ActorRoomStore` + gom Ticker + RAM Cache |
| **Độ trễ Chat (p95)** | **88 ms** (Local) \| **1626 ms** (Render) \| **1850 ms** (Grafana) | **190 ms** (Local) \| **1602 ms** (Render) | **718 ms** (Render) \| **~960 ms** (Grafana - ĐẠT ✅) |
| **Độ trễ Move RPC (p95)** | **7 ms** (Local) \| **155 ms** (Render) | **10 ms** (Local) \| **165 ms** (Render) | **165 ms** (Render - PASS ✅) |
| **Phân bổ tải CPU** | Các Core chạy rất nhẹ (< 6% CPU) | Core chạy nhỉnh hơn nhẹ ở tải thấp (Overhead của channel) | Giảm gộp 90% gói tin Centrifuge xuống client |
| **Rò rỉ kênh (Room Leak)**| **0** (PASS) | **0** (PASS) | **0** (PASS) |
| **Tần suất GC / Heap** | ~0.7 runs/s / ~45MB | ~0.6 runs/s / ~45MB | ~0.6 runs/s / ~45MB |

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
