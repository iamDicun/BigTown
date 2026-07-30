# Báo Cáo Tổng Kết Load Test & Tối Ưu Hóa Hiệu Năng Realtime

Tài liệu này tổng hợp và so sánh kết quả load test qua các Phase tối ưu hóa và các Chiến lược đo lường khác nhau của dự án BigTown.

---

## 1. Bản Đồ So Sánh Các Giai Đoạn (Phases Comparison)

| Chỉ số / Metric | Phase 1: Baseline (Chưa tối ưu) | Phase 2: Actor per-room | Phase 3: Tick Broadcast |
| :--- | :--- | :--- | :--- |
| **Trạng thái (Status)** | **Đã hoàn thành Str. 1 & 2** | *Chưa thực hiện* | *Chưa thực hiện* |
| **Cơ chế lưu trữ** | `MemoryRoomStore` (Global Mutex) | `ActorRoomStore` (Actor model) | `ActorRoomStore` + gom Ticker |
| **Độ trễ Chat (p95)** | **88 ms** (Local) \| **1626 ms** (Render - FAIL) | *Đang chờ* | *Đang chờ* |
| **Độ trễ Move RPC (p95)** | **7 ms** (Local) \| **155 ms** (Render - PASS) | *Đang chờ* | *Đang chờ* |
| **Phân bổ tải CPU** | Các Core chạy rất nhẹ (< 6% CPU) | Dự kiến trải đều đa Core | Dự kiến giảm tải Centrifuge |
| **Rò rỉ kênh (Room Leak)**| **0** (PASS) | *Đang chờ* | *Đang chờ* |
| **Tần suất GC / Heap** | ~0.7 runs/s / ~45MB | *Đang chờ* | *Đang chờ* |

---

## 2. Chi Tiết Kết Quả Theo Chiến Lược (Strategies)

### Giai Đoạn 1: Baseline (Phase 1)

#### **Strategy 1: Local Only (k6 → Backend Local)**
*   **Mục đích:** Đảm bảo tính đúng đắn (correctness) và làm mốc đối chứng (baseline).
*   **Kết quả đo lường:**
    *   **Chat test:** 100 VUs gửi tổng cộng 16,301 tin nhắn, nhận 163,009 tin nhắn (đúng tỷ lệ fanout 1:10 trong phòng). Độ trễ gửi-nhận trung bình 50.22ms, p95 là 88ms. Tỷ lệ lỗi 0%. `cross_room_leak` = 0.
    *   **Movement test:** 100 VUs di chuyển liên tục, sinh ra 329,708 RPC gửi lên và 2,588,677 broadcast tin nhắn nhận về. Độ trễ xử lý di chuyển cực thấp (avg 2.07ms, p95 7ms).
    *   **Tài nguyên hệ thống:** Mọi lõi CPU đều hoạt động cực kỳ nhẹ nhàng (dưới 6%). Ở mức tải 100 VUs, hệ thống chưa bị ghim cứng hay xảy ra nghẽn tranh chấp Lock (Lock Contention) do lượng tác vụ của 100 VUs còn nằm trong khả năng xử lý quá tốt của Go.
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-1-local/report.html)

#### **Strategy 2: Local to Render (k6 Local → Render Backend)**
*   **Mục đích:** Đo hiệu năng thực tế của server qua môi trường mạng WAN và hạ tầng container Render.
*   **Kết quả đo lường:**
    *   **Chat test:** 100 VUs gửi 16,214 tin nhắn, nhận 161,294 tin nhắn. Độ trễ truyền nhận chat **p95 vọt lên 1626 ms** (1.62s) $\rightarrow$ **FAIL** ngưỡng yêu cầu (&lt;1000ms).
    *   **Nguyên nhân nghẽn:** Độ trễ của HTTP POST Chat request trung bình là 1.13 giây, do database write latency ghi lịch sử chat vào Postgres trên Render hoặc CPU throttling trên Render Free/Starter.
    *   **Movement test:** 100 VUs gửi 326,648 RPC di chuyển qua WebSocket, nhận về 2,486,419 broadcasts. Độ trễ xử lý di chuyển **p95 là 155 ms** $\rightarrow$ **PASS** ngưỡng yêu cầu (&lt;500ms).
    *   **WS Connection Jitter:** Thời gian bắt tay kết nối WS (handshake) trung bình khoảng 343 ms (p95 là 511.64 ms).
*   **Báo cáo trực quan:** [report.html](file:///c:/Users/ADMIN/Documents/GitHub/BigTown/testing/loadtest/phase-1-baseline/strategy-2-online/report.html)

#### **Strategy 3: Grafana Cloud (Grafana k6 Cloud → Render)**
*   *Đã quyết định bỏ qua* - Vì Strategy 2 đã đủ thông số đối chứng trực tuyến và ghi nhận đầy đủ bottleneck ở tầng mạng/database.

---

### Giai Đoạn 2: Actor per-room Migration (Phase 2)
*   *Chưa thực hiện* - Mục tiêu loại bỏ Global Mutex bằng mô hình Actor cho từng Room độc lập.

---

### Giai Đoạn 3: Tick-based Broadcast (Phase 3)
*   *Chưa thực hiện* - Mục tiêu gom nhóm các gói tin di chuyển trong vòng 100ms trước khi broadcast nhằm giảm tải xử lý IO trên Centrifuge.

---

## 3. Hướng Dẫn Cập Nhật Báo Cáo
Khi hoàn thành các đợt load test tiếp theo, hãy cập nhật số liệu tương ứng vào các thư mục kết quả tương ứng (`phase-X-YYYY/strategy-Z-WWWW/results/*.json`) và cập nhật bảng so sánh trong file này để có bức tranh toàn cảnh về tiến trình tối ưu hóa hiệu năng.
