# BigTown — Hướng dẫn Load Test (Strategy 3: Grafana Cloud + backend public)

Tài liệu này chỉ dùng **Strategy 3** — chạy k6 trên hạ tầng Grafana Cloud, bắn vào **URL backend đã public**. Không dùng local (strategy 1) hay laptop→deploy (strategy 2).

Bộ script gồm 4 cái (2 cũ + 2 mới):

| Script | Đo gì | Loại |
| --- | --- | --- |
| `chat_load_test.js` | Chat REST → broadcast `player_chat`, cách ly room (`cross_room_leak=0`), độ trễ gửi→nhận | cũ |
| `movement_load_test.js` | RPC `player_move` qua WS, latency ack, nhận `room_state` | cũ |
| `placement_load_test.js` | Đặt vật phẩm: REST place → actor → broadcast `decoration_placed` (đường GHI + broadcast) | **mới** |
| `bootstrap_load_test.js` | `GET /api/realtime/bootstrap` spike/ramp (login/warp, giờ kèm `npc_spawns`) | **mới** |

---

## 0. Trạng thái script cũ (đã đối chiếu với backend hiện tại)

- **`chat_load_test.js` — còn dùng tốt.** Contract không đổi: `POST /api/rooms/:room/chat/messages` trả **201**, event `player_chat` có `roomId`, protocol Centrifuge connect/subscribe giữ nguyên.
- **`movement_load_test.js` — còn chạy, nhưng lưu ý một điểm.** RPC `player_move {x,y,direction,moving}` và ack `{rpc:{}}` vẫn đúng, nên **latency RPC vẫn đo chính xác**. Tuy nhiên hot path broadcast giờ là **tick 100ms `room_state`** (chỉ chứa player "dirty" từ lần flush trước), **không còn** push `player_move` mỗi lần. Hệ quả:
  - Nhánh đếm `data.type === 'player_move'` trong script giờ **không bao giờ chạy** (không sao, vô hại).
  - Metric `move_broadcast` và dòng "tỉ lệ giao" trong summary **không còn phản ánh fanout như trước** (room_state gộp nhiều player vào 1 message theo tick). Đọc `move_rpc_latency` và `ws_connect_errors` là chính; bỏ qua "tỉ lệ giao" của movement.
- **`tokens.json` trong repo đã HẾT HẠN** (exp ~2026-07-30). **Bắt buộc sinh lại** trước khi test, nếu không WS sẽ bị `ErrorTokenExpired` (xem mục 2).

---

## 1. Chuẩn bị một lần

### 1.1 Cài k6 + đăng nhập Grafana Cloud

```bash
# cài k6 (macOS ví dụ)
brew install k6

# lấy token ở: Grafana Cloud > k6 > Personal API token
k6 cloud login --token <GRAFANA_CLOUD_K6_TOKEN>
```

### 1.2 Điều kiện phía backend public

- URL công khai, ví dụ `https://api.big-town.example`. Grafana Cloud load generator phải **truy cập được** (không nằm sau VPN/nội bộ).
- **CheckOrigin**: backend chỉ cho origin trong `ALLOWED_ORIGINS` (hoặc origin rỗng). k6 không tự gửi `Origin`; nếu bị chặn 403 khi upgrade WS, truyền `-e ORIGIN=https://big-town.vercel.app` (một origin hợp lệ).
- WS path là `/connection/websocket` (nằm ở gốc, KHÔNG dưới `/api`).

---

## 2. Sinh seed + token (khớp nhau, chạy từ `backend/`)

Helper `cmd/loadtest-gen` sinh cả `seed.sql` lẫn `tokens.json` từ cùng một pattern UUID nên luôn khớp.

```bash
cd backend
# -ttl (phút) PHẢI lớn hơn tổng thời lượng test + thời gian chờ hàng đợi cloud.
# 60 phút là an toàn.
JWT_SECRET=<đúng secret của môi trường deploy> \
  go run ./cmd/loadtest-gen -users=100 -rooms=10 -ttl=60 -out=../testing/loadtest/scripts

# Nạp seed vào Postgres của môi trường deploy
psql "postgres://user:pass@<db-host>:5432/bigtown?sslmode=disable" \
  -f ../testing/loadtest/scripts/seed.sql
```

> `tokens.json` được k6 đóng gói kèm script khi upload (qua `open()` + `SharedArray`), không cần thao tác thêm trên cloud.

### 2.1 Nạp coin (BẮT BUỘC cho `placement_load_test.js`)

Seed loadtest tạo character với `coins=0`, mà đặt vật phẩm cần `coins >= giá item` → nếu không nạp, mọi lần đặt sẽ lỗi `insufficient coins`. Nạp thẳng bằng SQL:

```sql
UPDATE characters
SET coins = 100000000
WHERE user_id IN (SELECT id FROM app_user WHERE email LIKE 'loadtest+%');
```

> Lưu ý: coin còn được cache trong RAM của room actor. Nếu backend đã chạy và actor đã "ôm" ví cũ (coins=0), nạp DB xong hãy **restart backend** (hoặc đảm bảo actor nạp lại ví) trước khi test đặt vật phẩm.

---

## 3. Chạy trên Grafana Cloud

Đặt biến môi trường cho gọn:

```bash
export HOST=api.big-town.example
export WSS=wss://$HOST/connection/websocket
export HTTPS=https://$HOST
export ORG=https://big-town.vercel.app     # origin hợp lệ nếu CheckOrigin chặn
cd testing/loadtest/scripts
```

### 3.1 Chat

```bash
k6 cloud run chat_load_test.js \
  -e WS_URL=$WSS -e BASE_URL=$HTTPS -e ROOMS=10 -e ORIGIN=$ORG
```

### 3.2 Movement

```bash
k6 cloud run movement_load_test.js \
  -e WS_URL=$WSS -e ROOMS=10 -e ORIGIN=$ORG
```

### 3.3 Đặt vật phẩm (mới) — nhớ đã nạp coin ở 2.1

```bash
k6 cloud run placement_load_test.js \
  -e WS_URL=$WSS -e BASE_URL=$HTTPS -e ROOMS=10 -e ORIGIN=$ORG \
  -e PLACE_EVERY_MS=1000
```

### 3.4 Bootstrap spike (mới)

```bash
k6 cloud run bootstrap_load_test.js \
  -e BASE_URL=$HTTPS -e ROOMS=10 -e PEAK_RPS=200
```

Mỗi lần chạy sẽ in link tới **dashboard cloud** (biểu đồ theo thời gian, thresholds, và các custom metric của ta). Kết thúc còn có summary in ra stdout + file `*_summary.json`.

---

## 4. Đọc kết quả — ngưỡng PASS & ý nghĩa

### 4.1 Chat (`chat_load_test.js`)

| Metric | Ngưỡng | Ý nghĩa |
| --- | --- | --- |
| `cross_room_leak` | **== 0** (bắt buộc) | Rò tin sang room khác — lỗi correctness, phải sửa trước mọi thứ. |
| `chat_delivery_ms` p95 | < 1000ms | Độ trễ gửi→nhận. |
| `http_req_failed` | < 1% | POST chat lỗi. |
| Tỉ lệ giao (summary) | ~100% | `chat_received / (chat_sent × members/room)`. Thấp = drop/backpressure. |

### 4.2 Movement (`movement_load_test.js`)

| Metric | Ngưỡng | Ý nghĩa |
| --- | --- | --- |
| `move_rpc_latency` p95 | < 500ms | Round-trip ack RPC — **chỉ số chính**. |
| `move_rpc_error` | < 5% | RPC lỗi/timeout. |
| `ws_connect_errors` | < 5 | Rớt kết nối. |
| "tỉ lệ giao" | *bỏ qua* | Không còn ý nghĩa do broadcast chuyển sang tick `room_state`. |

### 4.3 Đặt vật phẩm (`placement_load_test.js`)

| Metric | Ngưỡng | Ý nghĩa |
| --- | --- | --- |
| `place_rest_ms` p95 | < 800ms | Latency REST place = thời gian qua actor + trừ coin + reply. p95 phồng khi actor bị nghẽn (serialize theo map). |
| `place_deliver_ms` p95 | < 1500ms | place→nhận `decoration_placed`. Chênh lớn so với `place_rest_ms` = nghẽn broadcast/backpressure. |
| `place_error` | < 10 | Lỗi hạ tầng (KHÔNG tính `place_occupied`). |
| `place_occupied` | — | Ô đã bị chiếm; bình thường khi stress, chỉ theo dõi. |
| Tỉ lệ giao (summary) | ~100% | `place_broadcast / (place_sent × members/room)`. |

### 4.4 Bootstrap (`bootstrap_load_test.js`)

| Metric | Ngưỡng | Ý nghĩa |
| --- | --- | --- |
| `bootstrap_ms` p95 / p99 | < 600 / < 1200ms | Latency đọc map + npc_spawns. Tăng theo RPS = DB/read là điểm nghẽn. |
| `http_req_failed` | < 1% | 5xx/timeout khi dồn tải. |
| `bootstrap_bad` | < 10 | 200 nhưng thiếu field (spawn_x/tick_rate_ms). |
| `bootstrap_npc_count` | — | Số `npc_spawns` trung bình — theo dõi độ nặng payload. |

> Muốn tìm **điểm gãy** thay vì chỉ pass/fail: tăng `-e PEAK_RPS` (bootstrap) hoặc đổi executor sang `ramping-vus` (chat/movement/placement) và quan sát mốc p95 bắt đầu dựng đứng.

---

## 5. Ghép với server-side metrics (khuyến nghị mạnh)

Số phía k6 chỉ cho biết "chậm ở đâu đó". Muốn biết **vì sao**, ghép cùng lúc với metrics server (`/metrics`, Centrifuge + Go runtime + Postgres pool). Bảng panel và cách đọc (CPU 1 core 100% = contention lock, `go_goroutines` tăng đơn điệu = leak, `pg WaitCount` tăng = nghẽn pool) đã có sẵn trong `docs/LoadTest-Guide.md` mục 3 — vẫn áp dụng nguyên vẹn. Với 2 test mới:

- **Placement**: theo dõi thêm độ trễ **write-behind persist** và Postgres pool (mỗi lần đặt là 1 INSERT placement + UPDATE coin). Nếu `place_rest_ms` thấp nhưng DB backlog tăng → write-behind đang dồn hàng.
- **Bootstrap**: theo dõi query maps/npc_spawns; cân nhắc cache nếu p95 dựng sớm.

---

## 6. Dọn dữ liệu sau test

```sql
-- Xoá placement do loadtest tạo (theo character loadtest)
DELETE FROM map_placements
WHERE character_id IN (
  SELECT c.id FROM characters c JOIN app_user u ON u.id = c.user_id
  WHERE u.email LIKE 'loadtest+%'
);

-- Xoá user loadtest (cascade sang characters, chat_messages)
DELETE FROM app_user WHERE email LIKE 'loadtest+%';

-- Xoá map loadtest nếu muốn
DELETE FROM maps WHERE code LIKE 'loadtest-map-%';
```

---

## 7. Có thể thêm sau (chưa làm script)

- **Coin pickup** (`POST /api/editor/coin-pickup`): test correctness chống double-claim (2 người nhặt cùng 1 coin, chỉ 1 thắng). Cần seed/spawn coin trước nên phức tạp hơn — làm sau nếu cần.
- **Kịch bản hỗn hợp**: 1 VU vừa giữ WS + di chuyển + thỉnh thoảng chat + thỉnh thoảng đặt vật phẩm, để mô phỏng tải thật thay vì tách từng loại.
