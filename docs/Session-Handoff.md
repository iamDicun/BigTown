# BigTown Session Handoff
**Đọc file này trước khi làm tiếp — tóm tắt trạng thái dự án và những gì vừa được implement, để session/người mới không phải đọc lại toàn bộ lịch sử chat.**

---

## 1. Trạng thái hiện tại (tóm tắt 1 phút)

MVP đã có: đăng nhập (local + Teams SSO), tạo character tự động, map `village_adventure` render bằng Phaser với player di chuyển 4 hướng, multiplayer realtime (thấy người khác di chuyển, join/leave), chat (HTTP POST + broadcast qua Centrifuge), khung chat expand/collapse.

**Toàn bộ code đã viết nhưng CHƯA commit vào git** — xem mục 3. Cũng CHƯA được xác nhận bằng mắt trên trình duyệt thật (không có công cụ browser trong phiên làm việc) — xem mục 5.

Chi tiết đầy đủ từng bước, từng quyết định kỹ thuật nằm ở **`docs/Movement-Chat-Spawn-Plan.md`** — file đó là nhật ký chi tiết (checklist A→I + audit kiến trúc), file này chỉ là bản tóm tắt điều hướng.

---

## 2. Việc đã làm trong session vừa rồi

Theo đúng thứ tự trong `docs/Movement-Chat-Spawn-Plan.md`:

- **Character & map mặc định**: mỗi user có 1 character, tự động gán vào map hiện hành qua config `GAME_DEFAULT_MAP_CODE` (backend `.env`), tự đồng bộ lại mỗi lần login — đổi map sau này chỉ cần seed map mới + đổi 1 biến env, không cần sửa code (xem `docs/Architecture.md` mục 9.1).
- **Chat**: HTTP POST `/api/rooms/:roomId/chat/messages` → lưu DB (`chat_messages`) → publish qua Centrifuge. Không dùng client publish trực tiếp.
- **Map + Phaser**: `village_adventure.tmj` (asset có sẵn) được embed tileset tự động (`asset/tools/embed_tilesets.js`, vì Phaser không hỗ trợ external tileset reference) rồi copy vào `frontend/public/assets/`. Player animation soi pixel thật từ `Player.png` (lưới 6×10 @32px, không phải 12×20@16px như đoán ban đầu).
- **Movement realtime server-authoritative**: `realtime/room/` (RoomStore RAM + MemoryRoomStore), RPC `player_move` qua Centrifuge (không phải `OnPublish`), validate tốc độ/bounds/overlap (minDistance 24px), anti-overlap spawn khi join, correction gửi qua **personal channel** (Centrifuge server-side subscription, không qua response RPC).
- **Frontend Phaser refactor**: `GameScene.ts` từ 291 dòng → ~100 dòng, tách thành `systems/mapSystem.ts`, `systems/localPlayerController.ts`, `systems/remotePlayerManager.ts`, và `network/gameSocket.ts` tự parse toàn bộ event thô thành type đã gõ kiểu.
- **Audit kiến trúc BE**: đối chiếu module mới với `docs/Architecture.md` + `docs/Realtime-Room-State-Decisions.md`, sửa 2 chỗ lệch thật (field `ClientID` thay vì `UserID` trong `RoomPlayer`, thêm `characterId` vào event correction cho khớp doc).
- **Đã xác nhận có chủ đích giữ nguyên**: usecase (`AuthUsecase`, `CharacterUsecase`) giữ `*sql.DB` trực tiếp chỉ để `BeginTx`/`Commit` (không chạy query trực tiếp) — không phải Clean Architecture "thuần" nhưng khớp pattern có sẵn từ trước, user đã xác nhận giữ nguyên, không refactor.

---

## 3. Trạng thái git — QUAN TRỌNG

**Chưa có commit nào cho toàn bộ việc trên.** `git log` gần nhất vẫn là `86cd8a4` (trước khi bắt đầu session). Toàn bộ thay đổi đang nằm ở working tree (`git status` sẽ thấy rất nhiều file M/??). Nếu muốn giữ lại, cần commit trước khi làm gì có rủi ro mất dữ liệu (checkout, reset, clean...).

File mới quan trọng (chưa track): `backend/internal/module/character/`, `backend/internal/module/chat/`, `backend/internal/module/realtime/{room,port}/`, `backend/internal/database/seed.sql`, `frontend/public/assets/`, `frontend/src/features/game/{phaser/createGame.ts,phaser/playerAnimations.ts,systems/{mapSystem,localPlayerController,remotePlayerManager}.ts,services/}`, `asset/tools/`, và các doc `docs/Movement-Chat-Spawn-Plan.md`, `docs/Session-Handoff.md` (file này).

---

## 4. Cách chạy dev

```bash
# 1. Postgres (docker) — đã có sẵn container backend-postgres-1, hoặc:
cd backend && docker compose up -d

# 2. Seed map village_adventure (idempotent, chạy lại vô hại):
docker exec -i backend-postgres-1 psql -U postgres -d app_db -f - < backend/internal/database/seed.sql

# 3. Backend:
cd backend && go run ./cmd/server
# -> :8080, cần backend/.env có GAME_DEFAULT_MAP_CODE=village_adventure (đã có sẵn)

# 4. Frontend:
cd frontend && npm run dev
# -> :5173
```

Test nhanh không cần trình duyệt: `POST /api/auth/register` → `/api/auth/login` → `GET /api/characters/me` / `GET /api/realtime/bootstrap` (cần Bearer token).

---

## 5. Chưa verify được — cần làm tiếp

- **Chưa mở trình duyệt thật để xác nhận bằng mắt** (không có công cụ browser trong phiên trước). Cần: mở `http://localhost:5173`, đăng nhập, xem sprite/animation/tile/collision có đúng không, mở 2 tab (2 user khác nhau) xem có thấy nhau di chuyển mượt không, thử chat, thử nút expand/collapse.
- Đã fix 1 bug người dùng báo: khung chat collapse không co khung ngoài (CSS Grid `align-self`) — **đã sửa, build sạch, nhưng cũng chưa tự mắt xác nhận lại trên browser.**

---

## 6. Quyết định kiến trúc cần nhớ (tránh hỏi lại/làm lại)

- **Movement**: pixel/free, `minDistance = 24px`, throttle gửi 100ms, coalesce latest event, gửi ngay khi vừa dừng.
- **Correction**: qua personal channel (`personal:<userID>`, Centrifuge server-side subscription), không qua response RPC.
- **Spawn**: luôn dùng spawn point mặc định của map (không đọc `characters.last_x/last_y`); nếu bị chiếm thì dò vòng xoắn ốc tìm chỗ trống.
- **NPC hiện tại trong map** (sheep/chicken/villager...) = flavor/decoration, **không đánh được**. Enemy combat thật (`npc_types`/`map_npc_spawns`/`enemy_hit`) là việc **chưa làm**, để sau.
- **Đổi map sau này**: seed map mới + đổi `GAME_DEFAULT_MAP_CODE`, không sửa code gì khác (xem `docs/Architecture.md` mục 9.1).
- **2 connection Centrifuge riêng** (ChatPanel + GameScene) cho cùng 1 user — biết và chấp nhận, chưa gộp lại (xem mục G trong Movement-Chat-Spawn-Plan.md).

---

## 7. Việc chưa làm / hướng tiếp theo

Xem `docs/Movement-Chat-Spawn-Plan.md` mục 3 "Ngoài phạm vi batch này":

- Enemy NPC thật (spawn, HP, `enemy_hit`, reward, animation từ `Player_Actions.png`).
- Debounced position persistence (`characters.last_x/last_y`) khi đứng yên/rời room/logout.
- Avatar builder, shop, inventory UI.
- Teams SSO auto-login trong `GameView`.
- Cân nhắc gộp 2 connection Centrifuge (ChatPanel + GameScene) làm 1.

---

## 8. Tài liệu tham khảo (đọc theo thứ tự nếu cần hiểu sâu)

1. `docs/Architecture.md` — tổng quan kiến trúc, mục 9.1 quan trọng (cách đổi map).
2. `docs/Storage-Design.md` — thiết kế DB/RAM/Frontend state.
3. `docs/Realtime-Room-State-Decisions.md` — quyết định chi tiết RoomStore/movement/chat (nguồn tham chiếu chính cho phần realtime).
4. `docs/Phaser-Frontend-Guide.md` — hướng dẫn tổ chức code Phaser.
5. `docs/Movement-Chat-Spawn-Plan.md` — **nhật ký chi tiết nhất của session vừa rồi**, checklist A→I, audit kiến trúc, mọi phát hiện kỹ thuật (Phaser không hỗ trợ external tileset, sprite layout thật, v.v.).
