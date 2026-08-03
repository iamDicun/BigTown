

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

## docs/phase 4

### proximity_voice_design.md — phù hợp tốt ✓, chỉnh vài chỗ

Bám đúng hạ tầng thật (personal channel, `GameRoom.Players` có X/Y/ClientID/UserID, `OnPublish` đã chặn, `OnRPC` switch, movement pipeline). Quyết định "media qua WebRTC, Centrifuge chỉ signaling + proximity" là đúng đắn, và tách khỏi editor/coin cũng đúng. Cần chỉnh:

- Tên hàm lệch với code thật: helper personal channel là **`sendPersonalEvent`** (không phải `publishPersonal`); lấy user id dùng **`client.UserID()`** (không có `userIDFromClient`).
- **Điểm quan trọng nhất (đang dưới‑spec):** `recomputeVoiceNeighbors(room, mover)` và `ResolveVoiceRelay` **không truy cập thẳng `GameRoom.Players` được**, vì store là `ActorRoomStore` (đóng gói trong actor, chỉ vào qua `dispatch`). Phải hoặc tính neighbor **ngay trong room actor**, hoặc tính trong usecase từ `store.GetSnapshot`. Tin tốt: `MovePlayer` **đã gọi `GetSnapshot` sẵn** cho check minDistance → tái dùng đúng snapshot đó, khỏi tốn thêm round‑trip.
- **Hysteresis cần chỗ lưu state:** để ra diff `add`/`remove` với 2 bán kính, phải nhớ audible‑set trước đó của mỗi player — doc chưa nói lưu ở đâu. Chốt: sống trong room actor (per‑room), cập nhật dưới lock của actor.
- Cadence: `handlePlayerMove` hiện không broadcast tức thời (có tick 100ms); recompute theo mỗi move (đã throttle 10/s) là ổn, chỉ cần thống nhất.

### Teams-SSO-Setup-Guide.md — phù hợp ✓, khớp code thật

Không phải doc lạc. Backend đúng như doc khẳng định: `teams/microsoft_token_verifier.go` (verify `aud == clientID` qua `jwt.WithAudience`, lấy `oid`/`tid`), `usecase/teams_login.go`, route `POST /api/auth/teams` (có prefix `/api`), config `TEAMS_CLIENT_ID`/`TEAMS_TENANT_ID`, entity `user_identity`. Frontend **chưa** tích hợp Teams SDK — đúng như doc nói phần còn lại là cấu hình Microsoft + SDK frontend + manifest.

Một điểm cần bổ sung khi làm (khớp cảnh báo §9 của chính doc): `TEAMS_TENANT_ID` mặc định `"common"` và verifier hiện **chỉ enforce `aud`**, chưa thấy kiểm chặt `claims.TID` thuộc tenant cho phép. Nếu không muốn user từ tenant Microsoft bất kỳ đăng nhập được, phải thêm `claims.TID == allowedTenant` (hoặc set tenant cụ thể + enforce). Code hiện chưa chặn — nhớ vá.

---

Nếu muốn, mình có thể viết sẵn patch cho lỗi P0 frontend + P1 collision‑aware spawn (nạp collision layer server‑side) để bạn dán thẳng vào. Ưu tiên số 1 vẫn là khôi phục `DecorationDeletedEvent` vì hiện frontend đang gãy build.