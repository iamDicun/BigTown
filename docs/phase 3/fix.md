

## Commit c094fb0 — phần sync ví: ĐÚNG ✓

Bạn đã áp đúng cả chuỗi: bỏ `EvictOnlineCoins` trong `writer.flush`; tiêm `CoinResolver` để `getOrLoadWallet` đọc ví live (`rm.GetCoins`) thay vì DB trần; `CmdJoin` seed‑if‑absent; `liveConns` + `scheduleEvict` grace 3s; `WarpPlayer` không còn tự fire `OnPlayerLeave` (hết double‑leave); thêm guard chống `residents` âm; test đã cập nhật chữ ký mới. Logic này khép kín được race "đổi map sai coin".

## Coin spawner — kiến trúc ổn, nhưng có lỗi cần xử lý

**🔴 P0 — Frontend KHÔNG build được (phải sửa ngay).** Diff đã **xoá nhầm** type `DecorationDeletedEvent` khỏi `gameEvents.ts` (chỗ đó bị thay bằng `CoinSpawnedEvent`), nhưng type này vẫn còn được tham chiếu ở `gameEvents.ts:121` (union `RoomEvent`) và `gameSocket.ts` (import dòng 16, option dòng 32, dùng dòng 84, và `isDecorationDeletedEvent` dòng 149). `vue-tsc`/`tsc` sẽ fail. Khôi phục lại type là xong:

```ts
export type DecorationDeletedEvent = {
  type: 'decoration_deleted'
  placementId: string
}
```

**🟠 P1 — Coin spawn đè lên tile bị chặn (regression).** `isTileFreeForCoin` chỉ check bounds + `occupied` (placements), **không** check collision layer của tilemap (tường/nước/cây) vì server không có dữ liệu đó. Bản client cũ có check `map.getTileAt(collisionLayerName)`. Hệ quả: coin lọt vào tường/trên nước → **không nhặt được**. Hướng đúng: nạp collision layer từ file map vào actor lúc `loadFromDB` (server‑authoritative); phương án tạm là lọc phía client trước khi `addCoin`. Không sửa thì sẽ có coin "ma" rải rác không lấy được.

**🟡 P2 — Claim thua race trả HTTP 500.** Trong `CmdClaimCoin`, coin đã bị nhặt trả `errors.New("coin không tồn tại...")` — không phải sentinel nên `mapErr` rơi vào `apperror.Internal` → **500**. Nên trả `ErrNotFound` (hoặc thêm `ErrCoinGone` map sang 404/409) để gọn log/monitoring. Client đã prediction nên chỉ là cosmetic, nhưng 500 sai ngữ nghĩa.

**🟡 P2 — Spawner chạy mãi trên map trống.** Actor không bao giờ bị gỡ khỏi `rm.actors`; ticker 10s vẫn spawn + broadcast `coin_spawned` vào phòng 0 người. Nên gate `tickCoins`/`spawnInitialCoins` bằng `len(m.residents) > 0` để đỡ phí publish + RAM.

**🟡 P2 — `tickCoins` giữ `coinsMu` khi gửi `outbound`.** Nếu `outbound` đầy (broadcast chậm), actor block **trong lúc giữ lock** → `GetSpawnedCoins` (đường HTTP) bị stall. Gom coin mới vào slice, `Unlock()`, rồi mới broadcast.

**⚪ Ghi chú nhỏ:** claim chưa check khoảng cách server‑side (coin_id broadcast cho mọi người → client sửa đổi có thể claim từ xa) — nhưng exploit coin vô hạn **đã đóng** (mỗi coin single‑claim, giá do server quyết), nên đây chỉ là hardening để sau. Ngoài ra `EvictOnlineCoins`, `CreditCoins`, `CmdCredit` giờ là dead code ở prod — giữ `CreditCoins`/`CmdCredit` như "cửa ví chính thức" (doc Fix 3) hoặc xoá tuỳ bạn; `EvictOnlineCoins` xoá được.