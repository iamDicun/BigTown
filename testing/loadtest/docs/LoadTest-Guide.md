# BigTown — Hướng dẫn Load Test realtime (k6) & Monitoring

Bộ này đo: **100 VU giữ WebSocket trong 5 phút**, mỗi VU gửi chat định kỳ, xác nhận **cùng room nhận được tin của nhau** và **khác room thì không** (metric `cross_room_leak` phải = 0), đồng thời đo **độ trễ gửi→nhận**.

Thư mục:

```
loadtest/
├── gen/main.go          # helper Go: sinh seed.sql + tokens.json (đặt vào backend/cmd/loadtest-gen/)
├── chat_load_test.js    # script k6
├── seed.sql             # (sinh ra) seed maps + users + characters
├── tokens.json          # (sinh ra) [{userId, token, room}]
└── LoadTest-Guide.md    # file này
```

---

## 0. Vì sao phải seed trước (đọc kỹ, đây là chỗ dễ sai nhất)

Ba ràng buộc bắt buộc từ code backend:

1. **Subscribe → `JoinRoom` → `GetMapByCode(roomID)`.** roomID phải là **map code có thật** trong bảng `maps`, không thì subscribe fail (`ErrorInternal`). ⇒ phải seed maps.
2. **`JoinRoom` và chat đều resolve character theo `userID`.** User không có character ⇒ fail. ⇒ phải seed user + character.
3. **Token WS/REST là JWT HS256 ký bằng `JWT_SECRET`.** ⇒ mint thẳng bằng chính `security.GenerateToken` của dự án, khỏi login qua API (đỡ tải DB auth, đỡ lệ thuộc password).

Helper `gen/main.go` sinh cả `seed.sql` lẫn `tokens.json` từ **cùng một pattern UUID cố định** nên chúng luôn khớp nhau — không tự tay chế UUID rời rạc.

### Các bước setup

```bash
# 1) Đặt helper vào backend rồi sinh seed + token (chạy TỪ backend/)
mkdir -p backend/cmd/loadtest-gen && cp loadtest/gen/main.go backend/cmd/loadtest-gen/
cd backend
JWT_SECRET=<đúng secret trong .env> go run ./cmd/loadtest-gen -users=100 -rooms=10 -ttl=30 -out=../loadtest

# 2) Nạp seed vào Postgres
psql "postgres://user:pass@host:5432/bigtown?sslmode=disable" -f ../loadtest/seed.sql

# 3) Chạy test (xem các chiến lược bên dưới)
cd ../loadtest
k6 run -e WS_URL=ws://localhost:8080/connection/websocket -e BASE_URL=http://localhost:8080 -e ROOMS=10 chat_load_test.js
```

> `-ttl` (phút) phải **lớn hơn** thời lượng test, nếu không token access hết hạn giữa chừng → WS bị `ErrorTokenExpired`. 30 phút là an toàn cho test 5 phút.
> Sau test, dọn dữ liệu: `DELETE FROM app_user WHERE email LIKE 'loadtest+%';` (cascade sẽ xoá characters; chat_messages tham chiếu character cũng cascade). Xoá maps loadtest riêng nếu muốn.

---

## 1. Mô hình phân bố room & tiêu chí đúng/sai

- 100 VU, `ROOMS=10` ⇒ **10 VU mỗi room**. VU `i` vào room `loadtest-map-(i%10)`.
- Cùng room: 10 người, mỗi tin một người gửi được **cả 10 người** (kể cả người gửi, vì Centrifuge phát cho mọi subscriber của channel) nhận.
- Khác room: cách ly bởi **channel** (`room:<code>`), Centrifuge không phát chéo.

Tiêu chí PASS (thresholds trong script):

| Metric | Ý nghĩa | Ngưỡng |
|---|---|---|
| `cross_room_leak` | Số lần một VU nhận tin mang `roomId` ≠ room của nó | **== 0** (bắt buộc) |
| `chat_delivery_ms` p95 | Độ trễ gửi→nhận | < 1000ms (tùy chỉnh theo hạ tầng) |
| `http_req_failed` | Tỉ lệ POST chat lỗi | < 1% |
| `checks` | Tỉ lệ check pass | > 99% |
| Tỉ lệ giao (summary) | `chat_received / (chat_sent × members_per_room)` | nên tiệm cận ~100% |

Tỉ lệ giao thấp bất thường = server drop message hoặc backpressure (WS write buffer đầy) — dấu hiệu nghẽn.

---

## 2. Ba chiến lược chạy

### 2.1 Local (backend + Postgres + k6 cùng một máy)

**Mục đích:** smoke test & kiểm *correctness* (leak = 0, luồng chạy thông), KHÔNG phải đo capacity thật.

```bash
# backend + db qua docker-compose sẵn có, rồi:
k6 run -e WS_URL=ws://localhost:8080/connection/websocket -e BASE_URL=http://localhost:8080 -e ROOMS=10 chat_load_test.js
```

Lưu ý:
- k6 và server **giành CPU của nhau** trên cùng máy → số latency/throughput **không đại diện** cho production. Đừng kết luận capacity từ đây.
- Tăng giới hạn file descriptor trước khi mở nhiều WS: `ulimit -n 65535`.
- Với 100 VU thì nhẹ, nhưng nếu sau này đẩy lên hàng nghìn, một máy sẽ hết cổng ephemeral / FD trước cả khi server nghẽn.
- Đây là nơi tốt nhất để **debug script** (bật `--http-debug` hoặc thêm `console.log`).

### 2.2 Deploy link, một máy của bạn (laptop → backend đã deploy)

**Mục đích:** đo capacity **thật** của server (server đã tách khỏi máy tạo tải), nhưng bị chặn trên bởi **năng lực một máy phát tải**.

```bash
k6 run \
  -e WS_URL=wss://<deploy-host>/connection/websocket \
  -e BASE_URL=https://<deploy-host> \
  -e ROOMS=10 \
  -e ORIGIN=https://big-town.vercel.app \
  chat_load_test.js
```

Lưu ý quan trọng:
- **CheckOrigin ở backend:** `allowOrigin` chỉ cho origin nằm trong `ALLOWED_ORIGINS` (hoặc origin rỗng). k6 mặc định không gửi `Origin`; nếu backend chặn, truyền `-e ORIGIN=<một origin hợp lệ>` (script sẽ set header). Nếu vẫn 403 khi upgrade, kiểm lại danh sách allowed origins của môi trường deploy.
- **Bạn đang đo cả chính máy mình.** Theo dõi CPU tiến trình `k6` và `ulimit -n`. Nếu k6 tự bão hòa (CPU máy bạn ~100%), p95 latency phồng lên là **do client**, không phải server → số liệu sai. 100 VU + WS + TLS thường vẫn ổn trên 1 laptop, nhưng hãy nhìn `k6` process, không chỉ nhìn kết quả.
- **TLS bắt tay** tốn hơn `ws://` local; jitter kết nối ban đầu là bình thường.
- Chạy k6 gần vùng deploy (độ trễ mạng thấp) để latency phản ánh xử lý server, không phải RTT internet.

### 2.3 Cloud — Grafana Cloud k6 (nhiều máy phát tải, có dashboard sẵn)

**Mục đích:** bỏ trần "một máy", phát tải phân tán từ nhiều VM/region, và có **dashboard hosted** tự động (biểu đồ theo thời gian, so sánh giữa các lần chạy, thresholds).

```bash
# đăng nhập một lần
k6 login cloud --token <GRAFANA_CLOUD_K6_TOKEN>

# chạy trên hạ tầng Grafana (nhiều load generator do Grafana quản lý)
k6 cloud run chat_load_test.js \
  -e WS_URL=wss://<deploy-host>/connection/websocket \
  -e BASE_URL=https://<deploy-host> \
  -e ROOMS=10 -e ORIGIN=https://big-town.vercel.app
```

Lưu ý:
- `tokens.json` được nạp qua `open()` + `SharedArray` nên **được đóng gói kèm script** khi upload lên cloud — không cần thao tác thêm. Nhớ `-ttl` đủ dài vì có thể chờ hàng đợi trước khi chạy.
- Load generator của Grafana phải **truy cập được** backend (URL công khai). Nếu backend chỉ mở nội bộ, dùng generator tự host (mục dưới).
- Dashboard cloud tự vẽ: VUs, request rate, ws sessions, p95/p99, và **các custom metric của ta** (`chat_delivery_ms`, `cross_room_leak`, `chat_sent/received`) cũng hiện.
- Có thể phân bổ VU theo nhiều region để mô phỏng người dùng nhiều nơi.

**Biến thể tự host generator trên VM cloud của bạn:** dựng vài VM (cùng region với deploy), cài k6, chạy `2.2` trên từng VM với phần VU chia nhỏ, rồi gom kết quả qua output chung (Prometheus remote write, mục 3). Rẻ hơn Grafana Cloud nhưng phải tự lo điều phối.

---

## 3. Monitoring & Dashboard

Đo ở **hai phía** mới đủ bức tranh: phía k6 (trải nghiệm client) và phía server (nguyên nhân gốc).

### 3.1 Phía k6 → Prometheus + Grafana

Stream metric k6 ra Prometheus (remote write) rồi vẽ bằng dashboard k6 chính thức:

```bash
K6_PROMETHEUS_RW_SERVER_URL=http://<prometheus>:9090/api/v1/write \
k6 run -o experimental-prometheus-rw \
  -e WS_URL=... -e BASE_URL=... -e ROOMS=10 chat_load_test.js
```

- Import dashboard k6 Prometheus (Grafana dashboard id **19665** — "k6 Prometheus") làm nền.
- Custom metric của ta xuất hiện dưới tên `chat_delivery_ms`, `cross_room_leak`, `chat_sent`, `chat_received` → thêm panel riêng cho chúng.
- Thay thế nhẹ hơn khi không có Prometheus: `-o json=result.json` rồi phân tích, hoặc `--out csv`.

### 3.2 Phía server (Go + Centrifuge + Postgres) → nơi tìm nguyên nhân

Dự án đã kéo sẵn `prometheus/client_golang` (qua Centrifuge). Expose `/metrics` và scrape bằng Prometheus. Những panel đáng giá nhất:

| Panel | Nguồn | Vì sao quan trọng |
|---|---|---|
| **Số WS connection đang mở** | Centrifuge node metrics | Xác nhận đúng 100 kết nối được giữ suốt 5 phút (không rớt dần). |
| **Messages published/sec** | Centrifuge | Throughput broadcast thực tế; so với `chat_sent × members`. |
| **`go_goroutines`** | Go runtime collector | **Cực quan trọng:** phát hiện goroutine leak. Nếu tăng đơn điệu không tụt sau khi VU rời → rò rỉ (liên quan trực tiếp phần actor migration). |
| **CPU theo từng core** | node_exporter / cAdvisor | **Chữ ký của global-mutex:** nếu một core bị ghim ~100% còn các core khác nhàn → đang bị serialize bởi 1 lock. Sau khi tách lock/actor, tải nên trải đều nhiều core. |
| **Postgres pool in-use / wait** | `sql.DBStats` (expose gauge) | Chat ghi DB mỗi tin (`chat_messages` INSERT). Nếu `WaitCount`/`WaitDuration` tăng → pool 25 là điểm nghẽn. |
| **GC pause / heap** | Go runtime | Loại trừ GC là thủ phạm khi latency phồng. |
| **p95/p99 delivery (từ k6)** | Prometheus (3.1) | Chồng cùng dashboard với server metrics để **tương quan nhân–quả** trên cùng trục thời gian. |

Ghép k6 metrics và server metrics trên **cùng một Grafana** (cùng Prometheus datasource) là mẹo quan trọng nhất: khi p95 latency vọt lên, bạn nhìn ngay xuống cùng mốc thời gian để biết là do CPU 1 core, do goroutine leak, hay do pool DB.

**Expose nhanh `sql.DBStats`** (nếu chưa có), ví dụ một collector nhỏ đọc `db.Stats()` mỗi 10s và set các Prometheus gauge: `OpenConnections`, `InUse`, `WaitCount`, `WaitDuration`. Đây là cách duy nhất thấy được pool 25 có đang nghẽn hay không.

### 3.3 Kịch bản đọc số liệu điển hình

- `cross_room_leak > 0` → **dừng, sửa correctness trước**, đừng quan tâm hiệu năng. Rò channel là lỗi nghiêm trọng, không phải vấn đề tải.
- p95 delivery tăng dần theo thời gian + `go_goroutines` tăng dần → **goroutine leak**.
- p95 tăng + **một** core 100%, các core khác rảnh → **global-mutex contention** ⇒ đúng động lực để làm actor migration.
- p95 tăng + `pg WaitCount` tăng → nghẽn **ghi DB chat**; cân nhắc batch insert / hàng đợi ghi (dùng semaphore như đã bàn), hoặc bỏ persist mỗi tin.
- Tỉ lệ giao < 100% nhiều + `ws write errors` server → **backpressure**: client đọc chậm hoặc buffer WS đầy.

---

## 4. Tinh chỉnh & mở rộng

- **Đổi tỉ lệ người/room:** chỉnh `-rooms` khi sinh seed **và** `-e ROOMS=` khi chạy k6 cho khớp. Ít room + nhiều người/room = fanout lớn (stress broadcast). Nhiều room = stress số channel.
- **Thêm di chuyển (player_move):** hiện script chỉ chat. Muốn stress hot path movement, gửi RPC qua WS: `socket.send(JSON.stringify({id: n, rpc: {method:'player_move', data:{x,y,direction,moving}}}))`. Cần seed maps có `width/height` đủ lớn (đã set 4000×4000).
- **Ramp thay vì cố định:** đổi executor sang `ramping-vus` để tìm điểm gãy (breaking point) thay vì soak 100.
- **So sánh trước/sau actor migration:** chạy **cùng script, cùng seed** trên bản `MemoryRoomStore` và bản `ActorRoomStore`, đặt hai lần chạy cạnh nhau trên dashboard. Kỳ vọng: p95 giảm, CPU trải đều nhiều core, throughput broadcast tăng. Đây chính là bằng chứng định lượng cho việc bỏ global mutex.
