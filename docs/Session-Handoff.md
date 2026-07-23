# BigTown Session Handoff
**Đọc file này trước khi làm tiếp — tóm tắt trạng thái dự án và những gì vừa được implement, để session/người mới không phải đọc lại toàn bộ lịch sử chat.**

Cập nhật lần cuối: **Session 2** (production debug + improvements)

---

## 1. Trạng thái hiện tại (tóm tắt 1 phút)

MVP hoạt động: đăng nhập (local + Teams SSO), tạo character tự động, map `village_adventure` render bằng Phaser, multiplayer realtime (thấy/nghe người khác di chuyển + chat). **Toàn bộ code chưa commit vào git** — xem mục 3.

**Session 2 chính**: production debug (BE Render + FE Vercel) → tìm & sửa 7 nhóm vấn đề realtime/loading/session, cải thiện hiệu năng hot-path từ 3 round-trip DB xuống RAM thuần, thêm UI (tên nhân vật, fade tán cây che người chơi).

Chi tiết đầy đủ từng bước, từng quyết định kỹ thuật nằm ở các file `docs/*.md` — xem mục 8 để biết đọc theo thứ tự nào.

---

## 2. Session 2 — Tóm tắt nhanh (production debug + improvements)

### 2.1. Realtime bugs (đã sửa)
1. **Multi-ClientID per character**: 1 user mở 2 connection (ChatPanel + GameScene) → server xoá player khỏi room khi 1 connection rớt dù connection kia còn → sửa bằng cơ chế `Clients map[string]map[string]struct{}` (1 character → tập hợp ClientID), chỉ xoá player khi **cả tập hợp rỗng**.
2. **Race condition `isFirstConnection`**: 2 connection join gần đồng thời của cùng user → cả 2 đều thấy "chưa tồn tại" → sửa bằng cách dịch quyết định vào `MemoryRoomStore.JoinRoom` (cùng 1 lock).
3. **Remote player jitter khi va chạm**: gắn dynamic body lên sprite tween → Arcade tự sync lại position mỗi frame → xung đột tween → sửa bằng cách tách sprite (tween) khỏi va chạm (Zone + static body riêng).
4. **"Đi qua rồi giật lại"**: zone chặn remote player là hình vuông 16×12 (nhỏ hơn minDistance 24px server dùng) → local player lách vào rồi bị correction → sửa bằng cách đổi zone thành hình vuông 48×48 (= 2×minDistance) dùng AABB-vs-AABB chắc chắn hơn (hoặc thử hình tròn bán kính 26+ nếu muốn tự nhiên hơn).

### 2.2. Hiệu năng hot-path (đã sửa)
- **`player_move` RPC gọi DB 3 lần** (tra user, sync map, lấy bounds map) mỗi tick ~100ms → trễ 4-5s production → sửa:
  - Cache `GetDefaultMap` trong RAM (map tĩnh, không đổi lúc server chạy).
  - Thêm index RAM `PlayersByUser: map[string]string` (userID → characterID) trong `GameRoom`.
  - `MovePlayer` tra luôn RAM, không gọi DB → hot-path 100% RAM, không round-trip nào.

### 2.3. Loading chậm (đã cải thiện 1 phần)
- Song song hoá 2 API độc lập (`loadMyCharacter` + `getBootstrap`) ở `GameCanvas.vue` bằng `Promise.all` → giảm thời gian chờ từ tổng 2 lệnh xuống bằng lệnh chậm nhất.
- **Chưa fix hoàn toàn**: `main.ts` chặn `app.mount()` trên `tryRestoreSession()` → login screen trống 3-4s nếu token refresh bị reload cũng thất bại. Nghi vấn là Render free-tier cold start (sleep 15min, wake slow), chưa confirm.

### 2.4. Logout/F5 session (đã sửa gần hết)
- **Logout không navigate**: `auth.store.ts` `logout()` thiếu `catch` → exception chặn `router.push` ở `Navbar.vue` → sửa thêm `catch {}`.
- **F5 mất session + cross-site cookie**: logout không disconnect thật ở prod (FE Vercel, BE Render khác domain) → sửa code (logout best-effort), nhưng **còn thiếu step quan trọng**: set `COOKIE_SAME_SITE=None` + `COOKIE_SECURE=true` trên Render dashboard (không làm được, chờ user xác nhận).
- **2 nhân vật khi đổi tài khoản cùng tab**: `gameStore.characterId` không reset lúc logout → `Navbar.vue` thêm `gameStore.$reset()` ngay khi logout.

### 2.5. UI mới
- **Tên nhân vật trên đầu**: lấy `character.Name` từ DB qua broadcast `room_snapshot`/`player_joined`, render `Phaser.GameObjects.Text` trên đầu sprite. Cập nhật vị trí mỗi frame (đọc lại vị trí render thật, không tween — khớp sprite chính).
- **Fix tên "Player" mặc định**: `GetOrCreateForUser` (safety net tạo character cho user cũ chưa có) không tra `app_user.full_name` → sửa bằng cách thêm `port.UserReader` vào `CharacterUsecase`, tra tên thật trước khi tạo.
- **Fade DecorationAbove layer**: file mới `aboveLayerFadeSystem.ts`, mỗi frame kiểm tra tile nào overlap bounding box local player thì tween alpha xuống 0.35.

### 2.6. Tài liệu mới
- `docs/Realtime-Performance-Techniques.md`: 8 kỹ thuật chung (race condition, static vs dynamic body, server-authoritative, hot-path optimization, RAM index, song song API, cross-site cookie, best-effort error handling).
- `docs/Realtime-Performance-Fixes.md`: chi tiết 6 nhóm vấn đề + trạng thái (đã sửa/chưa làm/cần xác nhận).

---

## 3. Session 1 — Việc đã làm (tóm tắt từ chat cũ)

Theo đúng thứ tự trong `docs/Movement-Chat-Spawn-Plan.md`:

- **Character & map mặc định**: mỗi user có 1 character, tự động gán vào map hiện hành qua config `GAME_DEFAULT_MAP_CODE` (backend `.env`), tự đồng bộ lại mỗi lần login — đổi map sau này chỉ cần seed map mới + đổi 1 biến env, không cần sửa code (xem `docs/Architecture.md` mục 9.1).
- **Chat**: HTTP POST `/api/rooms/:roomId/chat/messages` → lưu DB (`chat_messages`) → publish qua Centrifuge. Không dùng client publish trực tiếp.
- **Map + Phaser**: `village_adventure.tmj` (asset có sẵn) được embed tileset tự động (`asset/tools/embed_tilesets.js`, vì Phaser không hỗ trợ external tileset reference) rồi copy vào `frontend/public/assets/`. Player animation soi pixel thật từ `Player.png` (lưới 6×10 @32px, không phải 12×20@16px như đoán ban đầu).
- **Movement realtime server-authoritative**: `realtime/room/` (RoomStore RAM + MemoryRoomStore), RPC `player_move` qua Centrifuge (không phải `OnPublish`), validate tốc độ/bounds/overlap (minDistance 24px), anti-overlap spawn khi join, correction gửi qua **personal channel** (Centrifuge server-side subscription, không qua response RPC).
- **Frontend Phaser refactor**: `GameScene.ts` từ 291 dòng → ~100 dòng, tách thành `systems/mapSystem.ts`, `systems/localPlayerController.ts`, `systems/remotePlayerManager.ts`, và `network/gameSocket.ts` tự parse toàn bộ event thô thành type đã gõ kiểu.
- **Audit kiến trúc BE**: đối chiếu module mới với `docs/Architecture.md` + `docs/Realtime-Room-State-Decisions.md`, sửa 2 chỗ lệch thật (field `ClientID` thay vì `UserID` trong `RoomPlayer`, thêm `characterId` vào event correction cho khớp doc).
- **Đã xác nhận có chủ đích giữ nguyên**: usecase (`AuthUsecase`, `CharacterUsecase`) giữ `*sql.DB` trực tiếp chỉ để `BeginTx`/`Commit` (không chạy query trực tiếp) — không phải Clean Architecture "thuần" nhưng khớp pattern có sẵn từ trước, user đã xác nhận giữ nguyên, không refactor.

---

## 4. Trạng thái git — QUAN TRỌNG

**Chưa commit nào cho Session 1 + Session 2.** `git log` gần nhất vẫn là `e3bb021` (cũ). Toàn bộ ~200+ file thay đổi nằm ở working tree. **Session 2 có uncommitted local changes ở một vài file config** (`backend/internal/platform/config/config.go`), chủ yếu là test data thay đổi, không ảnh hưởng logic. **Nếu muốn giữ lại, cần commit ngay** (vì working tree dễ mất nếu do lỗi tay hay crash).

**File mới/sửa chính** (Session 2): 
- Backend: `room/state.go` (thêm `Name`, `UserID`, `PlayersByUser`), `memory_store.go` (multi-ClientID logic, `GetPlayerByUserID`), `room_usecase.go` (bỏ DB khỏi `MovePlayer`), `character/port/user_reader.go` (mới), `character/usecase/character_usecase.go` (cache map, tra user real name).
- Frontend: `nameTagSystem.ts` (mới), `remotePlayerManager.ts` (static collision zone, name tag), `localPlayerController.ts` (name tag), `aboveLayerFadeSystem.ts` (mới), `GameScene.ts` (wire name tag + fade), `gameSocket.ts` (thêm `name` field), `Navbar.vue` (reset gameStore), `auth.store.ts` (catch logout), `GameCanvas.vue` (song song API).
- Doc: `Realtime-Performance-Techniques.md` (mới), `Realtime-Performance-Fixes.md` (mới).

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

## 5. Quyết định kiến trúc cần nhớ (tránh hỏi lại/làm lại)

**Movement & realtime**:
- **Server-authoritative**: client optimistic render (tween 100ms), gửi RPC throttle 100ms, server validate atomic (tốc độ/bounds/occupied), response ngay (ack) nhưng correction gửi riêng qua personal channel.
- **minDistance = 24px**: tra khoảng cách 2 tâm sprite (x/y tính từ DB), kiểm tra cả lúc spawn và mỗi move.
- **Collision client-side**: zone static body quanh remote player (hình vuông 48×48 = 2×minDistance, AABB-vs-AABB chắc chắn) để chặn local player trước khi server bắt buộc correction.
- **Multi-ClientID**: 1 character = tập hợp ClientID, chỉ xoá player khỏi room khi tập hợp rỗng (tất cả connection rớt).
- **Correction**: qua personal channel (server-side subscription), không qua response RPC.
- **Spawn**: luôn dùng spawn point mặc định (không lưu last_x/last_y), nếu chiếm thì dò vòng xoắn ốc.
- **Database truy cập**:
  - `JoinRoom`/`LeaveRoom`: gọi DB là bình thường (chỉ lần đầu/cuối phiên).
  - `MovePlayer` (hot-path): 100% RAM, không DB nào — cache `GetDefaultMap`, index `PlayersByUser`.
- **NPC hiện tại** trong map = flavor, không đánh. Enemy combat = chưa làm.
- **2 connection Centrifuge** (ChatPanel + GameScene) cùng user — biết, chấp nhận, chưa gộp.

---

## 6. Việc chưa làm / tồn đọng (phạm vi chưa giải quyết)

### Tồn đọng từ Session 2
- **Set `COOKIE_SAME_SITE=None` + `COOKIE_SECURE=true` trên Render**: code đã sửa xong (logout best-effort), nhưng env var thực thế Render dashboard chưa được set. Cần user tự làm hoặc cấp quyền.
- **Load time 3-4s lúc login/map**: song song hoá 1 phần, nhưng `main.ts` vẫn chặn `app.mount()` trên `tryRestoreSession()`. Nghi vấn Render cold start, chưa confirm bằng cách reload 2 lần để xem lần 2 nhanh hay không.
- **Remote collision zone**: hiện dùng hình vuông 48×48 (AABB), chứng kiến "nhích" từng chút khi circle. User prefer circle (tự nhiên hơn) nhưng cần điều chỉnh lực va chạm hoặc tăng bán kính. **Hành động tiếp**: thử hình tròn bán kính 28-30px (lớn hơn 26px lần trước) để chặn sớm hơn, giảm "nhích".

### Chưa làm theo kế hoạch gốc
- Enemy NPC thật (spawn, HP, `enemy_hit`, reward, `Player_Actions.png` animation).
- Position persistence (`characters.last_x/last_y`) — debounce update lúc đứng yên / rời room.
- Avatar builder, shop, inventory UI.
- Teams SSO auto-login trong `GameView`.
- Gộp 2 Centrifuge connection thành 1 (cân nhắc hiệu suất/độ phức tạp).

---

## 7. Cách chạy & test

```bash
# Backend: Postgres + seed + go run
cd backend && docker compose up -d
docker exec -i backend-postgres-1 psql -U postgres -d app_db < backend/internal/database/seed.sql
go run ./cmd/server  # :8080

# Frontend
cd frontend && npm run dev  # :5173

# Quick test (curl, không cần browser)
POST /api/auth/register → /api/auth/login → GET /api/characters/me (with Bearer token)
```

---

## 8. Tài liệu tham khảo (đọc theo thứ tự nếu cần hiểu sâu)

**Thiết kế & quyết định**:
1. `docs/Architecture.md` — tổng quan (mục 9.1: cách đổi map).
2. `docs/Storage-Design.md` — DB/RAM/FE state design.
3. `docs/Realtime-Room-State-Decisions.md` — chi tiết RoomStore/movement/chat (tài liệu chuẩn cho realtime).

**Implementation & optimization**:
4. `docs/Realtime-Performance-Techniques.md` — 8 kỹ thuật chung (race condition, body va chạm, hot-path, RAM index, v.v.).
5. `docs/Realtime-Performance-Fixes.md` — chi tiết 7 nhóm vấn đề + trạng thái (đã sửa/chưa/cần xác nhận).

**Code organization**:
6. `docs/Phaser-Frontend-Guide.md` — tổ chức code Phaser, systems architecture.
7. `docs/Movement-Chat-Spawn-Plan.md` — nhật ký chi tiết Session 1 (checklist A→I, audit, phát hiện Phaser).
