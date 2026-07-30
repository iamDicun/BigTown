# Báo Cáo Tổng Kết Load Test & Tối Ưu Hóa Hiệu Năng Realtime

Tài liệu này tổng hợp và so sánh kết quả load test qua các Phase tối ưu hóa và các Chiến lược đo lường khác nhau của dự án BigTown.

---

## 1. Bản Đồ So Sánh Các Giai Đoạn (Phases Comparison)

| Chỉ số / Metric | Phase 1: Baseline (Chưa tối ưu) | Phase 2: Actor per-room | Phase 3: Tick Broadcast |
| :--- | :--- | :--- | :--- |
| **Trạng thái (Status)** | **Đã hoàn thành Str. 1, 2, 3** | **Đã hoàn thành Str. 1** | *Chưa thực hiện* |
| **Cơ chế lưu trữ** | `MemoryRoomStore` (Global Mutex) | `ActorRoomStore` (Actor model) | `ActorRoomStore` + gom Ticker |
| **Độ trễ Chat (p95)** | **88 ms** (Local) \| **1626 ms** (Render) \| **1850 ms** (Grafana - FAIL) | **190 ms** (Local) | *Đang chờ* |
| **Độ trễ Move RPC (p95)** | **7 ms** (Local) \| **155 ms** (Render) | **10 ms** (Local) | *Đang chờ* |
| **Phân bổ tải CPU** | Các Core chạy rất nhẹ (< 6% CPU) | Core chạy nhỉnh hơn nhẹ ở tải thấp (Overhead của channel) | Dự kiến giảm tải Centrifuge |
| **Rò rỉ kênh (Room Leak)**| **0** (PASS) | **0** (PASS) | *Đang chờ* |
| **Tần suất GC / Heap** | ~0.7 runs/s / ~45MB | ~0.6 runs/s / ~45MB | *Đang chờ* |

---

## 2. Chi Tiết Kết Quả Theo Chiến Lược (Strategies)

### Giai Đoạn 1: Baseline (Phase 1)

#### **Strategy 1: Local Only (k6 → Backend Local)**
*   **Mục đích:** Đảm bảo tính đúng đắn (correctness) và làm mốc đối chứng (baseline).
*   **Kết quả đo lường:**
    *   **Chat test:** p95 latency 88ms. Tỷ lệ lỗi 0%. `cross_room_leak` = 0.
    *   **Movement test:** p95 RPC latency 7ms.
    *   **Tài nguyên hệ thống:** Mọi lõi CPU hoạt động nhẹ nhàng (dưới 6%).
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-1-local/report.html)

#### **Strategy 2: Local to Render (k6 Local → Render Backend)**
*   **Mục đích:** Đo hiệu năng thực tế của server qua môi trường mạng WAN và hạ tầng container Render.
*   **Kết quả đo lường:**
    *   **Chat test:** p95 latency 1626 ms (FAIL). HTTP POST Chat tốn trung bình 1.13s (DB write latency Sing-Japan).
    *   **Movement test:** p95 RPC latency 155 ms (PASS).
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-2-online/report.html)

#### **Strategy 3: Grafana Cloud (Grafana k6 Cloud → Render)**
*   **Mục đích:** Bắn tải từ xa (Mỹ) để đo lường độ ổn định của hạ tầng mạng liên lục địa và database.
*   **Kết quả đo lường:**
    *   **Giới hạn tài khoản:** 100 VUs (giới hạn của gói free).
    *   **Vị trí phát tải:** Ohio, Mỹ (load_zone: `amazon:us:columbus`).
    *   **Độ trễ Chat:** **p95 vọt lên ~1750 - 1850 ms (FAIL)**.
    *   **Độ trễ bắt tay WS:** ~750 - 950 ms (do chặng đường Mỹ - Singapore).
    *   **Giải thích kết quả:** Thao tác bị tích lũy độ trễ địa lý đi qua 2 chặng lớn: Mỹ $\rightarrow$ Singapore (Server) $\rightarrow$ Nhật Bản (Database), kéo tụt đáng kể hiệu năng phản hồi HTTP REST.
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-3-grafana/report.html)

---

### Giai Đoạn 2: Actor per-room Migration (Phase 2)

#### **Strategy 1: Local Only (k6 → Backend Local)**
*   **Mục đích:** Đánh giá độ trễ cơ sở của mô hình Actor mới tại môi trường Local.
*   **Kết quả đo lường:**
    *   **Chat test:** p95 là 190 ms, tăng nhẹ so với Phase 1 (88ms).
    *   **Movement test:** p95 là 10 ms, tăng nhẹ so với Phase 1 (7ms).
    *   **Lý do:** Ở tải thấp (100 VUs), chi phí (overhead) của Go channel và closure dynamic allocation trong mô hình Actor lớn hơn so với việc Lock/Unlock một `sync.Mutex` thô hoàn toàn không bị tranh chấp ở local.
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-2-actor/strategy-1-local/report.html)

#### **Strategy 2: Local to Render (k6 Local → Render Backend)**
*   *Chưa thực hiện* - Đang đợi deploy code Phase 2 lên Render để kiểm tra.

---

### Giai Đoạn 3: Tick-based Broadcast (Phase 3)
*   *Chưa thực hiện* - Mục tiêu gom nhóm các gói tin di chuyển trong vòng 100ms trước khi broadcast nhằm giảm tải xử lý IO trên Centrifuge.

---

## 3. Hướng Dẫn Cập Nhật Báo Cáo
Khi hoàn thành các đợt load test tiếp theo, hãy cập nhật số liệu tương ứng vào các thư mục kết quả tương ứng (`phase-X-YYYY/strategy-Z-WWWW/results/*.json`) và cập nhật bảng so sánh trong file này để có bức tranh toàn cảnh về tiến trình tối ưu hóa hiệu năng.
