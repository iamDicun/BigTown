

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