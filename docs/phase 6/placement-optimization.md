# BigTown — Thử nghiệm Tối ưu Placement: DB Transaction

**Ngày:** 2026-08-07 | **Phase:** 5 | **Kết quả:** ❌ Reverted — không khả thi với hạ tầng hiện tại

---

## Kết luận sau thử nghiệm

**DB transaction làm placement chậm hơn actor batch write-behind** trong điều kiện hạ tầng Render (Singapore) + Aiven Postgres (Nhật Bản, ~80ms RTT).

| | Actor + batch write (Phase 3) | DB Transaction (Phase 5) |
|:---|:---|:---|
| REST p95 | ~1200ms | ~5000-10000ms |
| Delivery p95 | ~5000ms | ~15000-55000ms |
| WS errors | 2 | 23 |

### Nguyên nhân

Mỗi DB transaction = 2 round-trip (UPDATE + INSERT) × 80ms = 160ms network tối thiểu. Với 100 req/s và connection pool ~20, hàng đợi DB vượt xa actor queue (1 batch/s gộp 512 ops thành 1 round-trip).

### Bài học

- CPU không phải bottleneck
- Mạng không phải bottleneck  
- **Khoảng cách địa lý app server ↔ DB server** mới là yếu tố quyết định
- Batch write-behind phù hợp với DB ở xa, DB transaction chỉ phù hợp khi DB cùng datacenter
- Với hạ tầng hiện tại (Render 1 core + Aiven JP), actor batch write-behind là tối ưu, không còn cách cải thiện placement nào khác

---

## 1. Bối cảnh & dữ liệu

Sau 3 phase load test placement (100 VU, 10 room, Grafana Cloud + local), kết quả nhất quán:

| Môi trường | REST p95 | REST median | Delivery p95 |
|:---|:---|:---|:---|
| Grafana Cloud (Ohio → SG) | ~1200ms | ~870ms | ~5000ms |
| Local (VN → SG) | **1213ms** | 714ms | 4368ms |

Hai kết luận quan trọng:

1. **Mạng không phải bottleneck** — local (ping ~30ms) p95 gần bằng Grafana (ping ~200ms)
2. **CPU không max** — server chỉ xử lý ~100 req/s, goroutine ~350, heap ~50MB
3. **Gốc rễ là actor command queue** — 10 lệnh/s/room xếp hàng tuần tự trong 1 goroutine, Go scheduler trên 1 core khiến hàng đợi tích tụ

### So sánh chi tiết thời gian

| Thành phần | Thời gian | Ghi chú |
|:---|:---|:---|
| Network RTT (VN→SG) | ~50ms | Không đổi |
| Actor command queue wait | **~600ms** | ← Bottleneck chính |
| Actor xử lý (validate + coin + RAM) | ~0.6ms | Rất nhẹ |
| HTTP overhead (parse, serialize, middleware) | ~10ms | Không đáng kể |
| **Tổng median** | **~660ms** | Khớp với 714ms local median |

---

## 2. Nguyên nhân actor chậm

Actor goroutine không dùng nhiều CPU (~0.6ms/lệnh), nhưng trên 1 core nó phải cạnh tranh với:

| Goroutine | CPU usage | Tần suất |
|:---|:---|:---|
| Writer `flush()` (DB transaction) | 10-50ms | Mỗi 1 giây |
| Centrifuge broadcast (~100 msg/s) | 5-10ms/s | Liên tục |
| HTTP server goroutines | 5ms/s | Liên tục |
| Go scheduler context switch | ~0.01ms | Hàng nghìn/s |

Go scheduler trên 1 core không đảm bảo actor goroutine được chạy ngay khi có lệnh mới → lệnh tích tụ trong channel → p95 tăng.

**Dùng DB transaction thay actor sẽ chuyển phần việc nặng nhất (coin deduct + insert placement) sang Postgres — chạy trên CPU riêng, xử lý song song qua connection pool.**

---

## 3. Giải pháp đề xuất

### 3.1 Luồng hiện tại

```
HTTP POST /api/editor/place
  → AuthMiddleware
  → EditorUsecase.PlaceItem()
    → resolve character (RAM cache)
    → get item (DB)
    → a.SendCmd(Cmd{Place})        ← gửi lệnh vào actor channel
      ↓ [block chờ]
    Actor goroutine (tuần tự):
      1. Validate tọa độ (RAM)
      2. Check occupied (RAM)
      3. Check coin (RAM: wallets)
      4. Trừ coin (RAM)
      5. Ghi placement (RAM: occupied + byID)
      6. Reply ← kênh reply
      7. Broadcast → outbound channel
      8. dirty <- persistOp → writer → 1s sau INSERT DB
      ↓ [unblock]
  → HTTP 200 (~700-1200ms)
```

### 3.2 Luồng đề xuất

```
HTTP POST /api/editor/place
  → AuthMiddleware
  → EditorUsecase.PlaceItem()
    → resolve character (RAM cache)
    → get item (RAM cache hoặc DB)
    → tx = db.Begin()
      → UPDATE coins = coins - price WHERE id=X AND coins >= price
      → INSERT INTO map_placements ON CONFLICT (map_id,x,y) DO NOTHING
      → Nếu cả 2 OK → tx.Commit()
         Nếu thất bại → tx.Rollback() → trả lỗi
    → broadcast decoration_placed (push channel riêng)
    → HTTP 200 (dự kiến ~300-400ms)
```

### 3.3 Thay đổi trong code

| File | Thay đổi |
|:---|:---|
| `editor_usecase.go:PlaceItem()` | Gọi DB transaction thay vì `a.SendCmd(CmdPlace)` |
| `map_actor.go` | Xóa `handlePlace`, giữ `occupied` grid cho occupancy check |
| `writer.go` | Không cần `opPlace` trong batch (đã INSERT trực tiếp) |
| `room_manager.go` | Thêm method `CheckOccupied(mapCode, x, y) bool` |

---

## 4. Tại sao DB transaction tốt hơn

| | Actor (hiện tại) | DB transaction |
|:---|:---|:---|
| **Serialize** | 1 goroutine/room, FIFO channel | Row lock, nhiều worker Postgres |
| **100 req/s** | Xếp hàng trong channel | Connection pool 25, xử lý song song |
| **CPU** | Tranh 1 core với tất cả goroutine khác | CPU riêng của Postgres server |
| **p95 dự kiến** | 1200ms | **~300-400ms** |
| **Consistency** | Write-behind delay 1s, mất data nếu crash | ACID transaction, không mất |
| **Race RAM-DB** | CÓ (1s gap) | KHÔNG (DB là source of truth) |
| **Code complexity** | ~200 dòng actor | ~50 dòng SQL + handler |
| **Scale** | Không scale (1 goroutine) | Connection pool = scale theo DB |

### Connection pool: 25 là đủ

- 100 req/s × 25ms/transaction = 2.5 connections trung bình
- Peak: 25 connections cho phép 25 transaction đồng thời
- Không cần tăng lên 50

---

## 5. Tradeoff & rủi ro

| Rủi ro | Mức độ | Giải pháp |
|:---|:---|:---|
| DB load tăng (100 txn/s) | Thấp | Postgres xử lý được, mỗi txn nhẹ (1 UPDATE + 1 INSERT) |
| Network latency DB (SG→Nhật ~80ms) | Trung bình | Vẫn nhanh hơn actor queue 660ms; connection pool giảm blocking |
| Mất actor làm nơi tập trung logic | Thấp | Actor vẫn giữ occupancy grid + broadcast |
| Transaction conflict (cùng ô) | Thấp | `ON CONFLICT DO NOTHING` xử lý tự động |

---

## 6. Kế hoạch triển khai

1. **Bước 1:** Thêm method `PlaceItemTx(db, input)` trong usecase — gọi DB transaction
2. **Bước 2:** Xóa `handlePlace` trong actor, giữ `CheckOccupied` và broadcast
3. **Bước 3:** Xóa `opPlace` trong writer (không cần batch INSERT placement nữa)
4. **Bước 4:** Test lại placement load test
5. **Bước 5:** So sánh kết quả, quyết định merge

### Tiêu chí thành công

- REST p95 < 800ms (ngưỡng PASS)
- WS errors < 5
- Không crash trong 5 phút test
- Không regression ở chat/movement/bootstrap
