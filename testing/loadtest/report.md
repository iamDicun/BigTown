# Báo Cáo Tổng Kết Load Test & Tối Ưu Hóa Hiệu Năng Realtime

Tài liệu này tổng hợp và so sánh kết quả load test qua các Phase tối ưu hóa và các Chiến lược đo lường khác nhau của dự án BigTown.

---

## 1. Bản Đồ So Sánh Các Giai Đoạn (Phases Comparison)

| Chỉ số / Metric | Phase 1: Baseline (Chưa tối ưu) | Phase 2: Actor per-room | Phase 3: Tick Broadcast |
| :--- | :--- | :--- | :--- |
| **Trạng thái (Status)** | **Đã hoàn thành Str. 1, 2, 3** | **Đã hoàn thành Str. 1 & 2** | *Chưa thực hiện* |
| **Cơ chế lưu trữ** | `MemoryRoomStore` (Global Mutex) | `ActorRoomStore` (Actor model) | `ActorRoomStore` + gom Ticker |
| **Độ trễ Chat (p95)** | **88 ms** (Local) \| **1626 ms** (Render) \| **1850 ms** (Grafana) | **190 ms** (Local) \| **1602 ms** (Render - FAIL) | *Đang chờ* |
| **Độ trễ Move RPC (p95)** | **7 ms** (Local) \| **155 ms** (Render) | **10 ms** (Local) \| **165 ms** (Render - PASS) | *Đang chờ* |
| **Phân bổ tải CPU** | Các Core chạy rất nhẹ (< 6% CPU) | Core chạy nhỉnh hơn nhẹ ở tải thấp (Overhead của channel) | Dự kiến giảm tải Centrifuge |
| **Rò rỉ kênh (Room Leak)**| **0** (PASS) | **0** (PASS) | *Đang chờ* |
| **Tần suất GC / Heap** | ~0.7 runs/s / ~45MB | ~0.6 runs/s / ~45MB | *Đang chờ* |

---

## 2. Chi Tiết Kết Quả Theo Chiến Lược (Strategies)

### Giai Đoạn 1: Baseline (Phase 1)

#### **Strategy 1: Local Only (k6 → Backend Local)**
*   **Mục đích:** Đảm bảo tính đúng đắn (correctness) và làm mốc đối chứng (baseline).
*   **Kết quả đo lường:** p95 Chat 88ms | p95 Move 7ms.
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-1-local/report.html)

#### **Strategy 2: Local to Render (k6 Local → Render Backend)**
*   **Mục đích:** Đo hiệu năng thực tế của server qua môi trường mạng WAN và hạ tầng container Render.
*   **Kết quả đo lường:** p95 Chat 1626 ms (FAIL) | p95 Move 155 ms (PASS).
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-2-online/report.html)

#### **Strategy 3: Grafana Cloud (Grafana k6 Cloud → Render)**
*   **Mục đích:** Bắn tải từ xa (Mỹ) để đo lường độ ổn định của hạ tầng mạng liên lục địa và database.
*   **Kết quả đo lường:** p95 Chat ~1850 ms (FAIL).
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-3-grafana/report.html)

---

### Giai Đoạn 2: Actor per-room Migration (Phase 2)

#### **Strategy 1: Local Only (k6 → Backend Local)**
*   **Mục đích:** Đánh giá độ trễ cơ sở của mô hình Actor mới tại môi trường Local.
*   **Kết quả đo lường:** p95 Chat 190 ms | p95 Move 10 ms.
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-2-actor/strategy-1-local/report.html)

#### **Strategy 2: Local to Render (k6 Local → Render Backend)**
*   **Mục đích:** Đánh giá hiệu năng của mô hình Actor Store mới trên môi trường trực tuyến Render.
*   **Kết quả đo lường:**
    *   **Chat test:** 16,188 tin gửi. Độ trễ **p95 là 1602 ms (FAIL)**. Trực tiếp xác nhận bottleneck nằm ở DB Write Latency chứ không phải ở Concurrency Lock của room (giữ nguyên so với Phase 1: 1626ms).
    *   **Movement test:** 326,707 RPC di chuyển. Độ trễ **p95 là 165 ms (PASS)**, tăng nhẹ không đáng kể do chi phí channel/context switch (~10ms) so với Phase 1 (155ms).
    *   **Tính ổn định:** Ghi nhận 1 lỗi HTTP POST Chat (0.006%), còn lại checks pass 100%. `cross_room_leak` = 0 (PASS).
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-2-actor/strategy-2-online/report.html)

---

### Giai Đoạn 3: Tick-based Broadcast (Phase 3)
*   *Chưa thực hiện* - Mục tiêu gom nhóm các gói tin di chuyển trong vòng 100ms trước khi broadcast nhằm giảm tải xử lý IO trên Centrifuge.

---

## 3. Hướng Dẫn Cập Nhật Báo Cáo
Khi hoàn thành các đợt load test tiếp theo, hãy cập nhật số liệu tương ứng vào các thư mục kết quả tương ứng (`phase-X-YYYY/strategy-Z-WWWW/results/*.json`) và cập nhật bảng so sánh trong file này để có bức tranh toàn cảnh về tiến trình tối ưu hóa hiệu năng.
