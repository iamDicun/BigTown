# BigTown Implementation Plan — Spawn, Movement, Chat UI
**Batch kế tiếp sau khi chốt movement model / correction / spawn source. Dùng file này để track tiến độ.**

---

## 0. Quyết định đã chốt (không đổi lại ở batch này)

- NPC hiện có trong `asset/Maps/village_adventure.tmj` (layer `NPCSpawns`: sheep/chicken/cow/pig/fisher/villager...) = **flavor/decoration, không đánh được**. Enemy combat (slime/skeleton theo `npc_types`) sẽ thêm **sau**, spawn riêng, không dùng chung layer này.
- Movement: **pixel/free movement**, chống overlap bằng `minDistance = 24px`.
- Correction event (khi server reject movement): publish qua **personal Centrifuge channel theo userID**, tách khỏi room broadcast.
- Vị trí spawn khi join room: **luôn dùng spawn point mặc định của map** (`maps.spawn_x/spawn_y`), không đọc `characters.last_x/last_y`. Hai cột này vẫn giữ trong schema để dành cho debounced persistence sau, chỉ là join-room không đọc chúng.
- Rogue Phase 1 code (chat module + character module, do 1 agent tự ý viết ngoài yêu cầu) đang nằm trong `git stash@{0}` ("rogue-agent-phase1-wip"), **chưa áp dụng vào working tree**.

## 0.1 Một điểm cần xác nhận lại

Yêu cầu gốc có câu "nhân vật xuất hiện trong map (không dùng vị trí nhau)". Hiểu là: khi nhiều player join cùng lúc, họ **không được đứng đè lên nhau** tại spawn point mặc định — nếu vị trí spawn đang bị chiếm (trong phạm vi `minDistance` 24px với player khác), server phải dịch sang vị trí trống gần nhất, không phải mỗi người tự chọn vị trí khác nhau. Plan bên dưới làm theo cách hiểu này (mục F).

---

## 1. Mục tiêu batch này

User login → vào `GameView` → thấy nhân vật của mình đứng trong map `village_adventure` (không đè lên player khác) → di chuyển bằng phím, người khác thấy mình di chuyển mượt qua interpolation → chat được qua khung chat bên phải, có thể **expand/collapse**.

Ngoài phạm vi: enemy NPC thật, debounced position persistence, avatar/shop/inventory UI, Teams SSO auto-login.

---

## 2. Việc cần làm — theo thứ tự

### A. Restore & audit Phase 1 stash
- [x] `git stash pop` lại code chat + character (module `character/`, `chat/`, service FE, sửa `centrifuge.go`/`schema.sql`/`auth usecase`).
- [x] Đọc lại toàn bộ diff, chạy `go build ./...`, `go vet ./...` (backend) và `vue-tsc --noEmit` (frontend) — sạch, giữ nguyên toàn bộ.

### B. FE network layer: Axios + .env
- [x] Thêm dependency `axios` vào `frontend/package.json`.
- [x] Viết lại `frontend/src/shared/api/http.ts`: dùng `axios.create({ baseURL: import.meta.env.VITE_API_BASE_URL, withCredentials: true })`, giữ nguyên public API `http.get/post/put/patch/delete`, `ApiError`, logic refresh-once-retry 401.
- [x] Bỏ fallback hardcode `'http://localhost:8080/api'` — throw lỗi rõ nếu thiếu `VITE_API_BASE_URL` (áp dụng cả `http.ts` và `gameSocket.ts`).
- [x] `.env`/`.env.example` đã có `VITE_API_BASE_URL`, không cần thêm biến. Build production (`npm run build`) sạch.

### C. Seed dữ liệu `maps` cho village_adventure
- [x] Seed 1 row vào `maps` — đã chạy `backend/internal/database/seed.sql` vào DB dev. File idempotent, đã thêm vào `docker-compose.yml` (`02_seed.sql`).
- [x] Thêm config `GAME_DEFAULT_MAP_CODE` (env + `platform/config/config.go`, section `Game` mới, default `village_adventure`). Điểm cấu hình duy nhất — xem `docs/Architecture.md` mục 9.1.
- [x] Character get-or-create VÀ mỗi lần gọi `GET /api/characters/me` đều đồng bộ lại `characters.map_id` theo `GAME_DEFAULT_MAP_CODE` (`CharacterUsecase.syncMap`/`CharacterRepository.SyncMapID`) — đã test thật: character có `map_id = NULL` (giả lập map cũ) tự sync lại đúng map ở lần gọi kế tiếp.
- [x] Refactor `RealtimeUsecase.GetBootstrap`: bỏ hardcode `"starter-town"`, đọc `maps` qua `port.MapReader` (bind bằng `CharacterUsecase.GetDefaultMap`) — đã test thật qua HTTP, trả đúng `map_code=village_adventure`, tilemap/tileset key, spawn point, `default_channel=room:village_adventure`.
- [x] Ghi chú kỹ thuật: map thực tế dùng tile **16x16** (không phải 32x32 như draft đầu Phaser-Frontend-Guide) — Phaser sẽ zoom camera ~2x, world/tile coordinate vẫn tính theo 16px.

### D. Copy asset vào `frontend/public`
- [x] `asset/Maps/village_adventure.tmj` → `frontend/public/assets/maps/`.
- [x] 10 tileset PNG → `frontend/public/assets/tiles/` (flatten, không phân biệt thư mục gốc `Tiles/`/`Outdoor decoration/`).
- [x] `asset/Player/Player.png` → `frontend/public/assets/player/`. **Đã sửa lại nhận định ban đầu**: soi pixel thật cho thấy đây là lưới **6 cột × 10 hàng, khung 32×32px** (không phải 12×20 @16px như ghi ban đầu). Mapping animation theo hàng (frame index = row×6 + col):
  - Hàng 0 (frame 0-5): idle-down
  - Hàng 1 (frame 6-11): walk-down
  - Hàng 2 (frame 12-17): idle-up
  - Hàng 3 (frame 18-23): walk-up
  - Hàng 4 (frame 24-29): walk-left (side profile rõ ràng)
  - walk-right/idle-right: dùng lại frame hàng 4 + `sprite.setFlipX(true)` (không dùng hàng 5 vì hàng đó là biến thể back-view không rõ ràng, flip an toàn hơn)
  - Hàng 6-9: attack/hurt (dùng cho combat sau, **ngoài phạm vi batch này**)
- [x] `asset/Player/Player_Actions.png` → cùng thư mục, chưa dùng trong batch này (cũng là khung 32×32, chưa map animation).
- [x] **Phát hiện quan trọng**: Phaser **không hỗ trợ external tileset reference** (`.tsj` qua `source`) — `node_modules/phaser/src/tilemaps/parsers/tiled/ParseTilesets.js` in cảnh báo và bỏ qua tileset đó. Đã viết `asset/tools/embed_tilesets.js`: đọc `village_adventure.tmj` gốc (giữ external ref để còn mở lại bằng Tiled), inline nội dung từng `.tsj` vào thẳng mảng `tilesets`, ghi bản đã embed đè lên `frontend/public/assets/maps/village_adventure.tmj` (bản duy nhất frontend load). Chạy lại script này mỗi khi map gốc đổi.

### E. Phaser thật (thay stub hiện tại)
- [x] Thêm dependency `phaser` vào `frontend/package.json`.
- [x] `GameCanvas.vue`: mount `Phaser.Game` thật lúc mount (gọi `GET /api/characters/me` rồi `GET /api/realtime/bootstrap` trước khi tạo game), `destroy(true)` lúc unmount.
- [x] `BootScene`/`PreloadScene`/`GameScene`: data (bootstrap) truyền qua `scene.start(key, data)` + `init(data)`, không dùng `registry` (tránh race lúc Phaser auto-start scene đầu). Load tilemap JSON đã embed + tileset theo đúng tên parse từ `tileset_asset_key`, không hardcode danh sách.
- [x] Collision đọc từ object layer `Collision` bằng `physics.add.staticGroup()` + `add.zone()` cho từng object.
- [x] Animation `Player.png`: xác nhận đúng layout **6 cột × 10 hàng, khung 32×32px** bằng cách cắt/phóng to từng hàng để soi (không đoán từ ảnh preview nhỏ) — xem mapping ở mục D.
- [x] Spawn local player theo `bootstrap.spawn_x/spawn_y` (không hardcode trong `create()`); camera follow + `setBounds`/`world.setBounds` theo `map_width/map_height × 16px`, zoom 2x.
- [x] `go build`, `go vet`, `vue-tsc -b`, `npm run build` đều sạch. Đã curl kiểm tra toàn bộ 12 asset path (tilemap + 10 tileset + player) qua dev server, tất cả trả 200.
- [ ] **Chưa xác nhận bằng mắt**: không có công cụ trình duyệt để tự chụp/quan sát canvas render (sprite frame đúng hướng, tile không lệch, collision không xuyên tường). Backend (`:8080`) và frontend dev (`:5173`) đang chạy sẵn — cần bạn tự mở trình duyệt kiểm tra golden path trước khi coi bước này là "xong" hoàn toàn.

### F. Backend RoomStore + join/movement RPC
(theo đúng thứ tự mục 12, bước 4-13 của `docs/Realtime-Room-State-Decisions.md`)
- [x] `realtime/room/state.go`, `store.go` (interface `RoomStore`), `memory_store.go` (mutex, RAM, key theo `characterID`).
- [x] `JoinRoom`: lấy `spawn_x/spawn_y` từ map mặc định; nếu bị chiếm (trong `minDistance` 24px) → dò vòng xoắn ốc quanh spawn point tìm vị trí trống. **Đã test thật**: 2 client join cùng lúc, client 2 tự động bị dịch từ (384,512) sang (408,512) — đúng 24px, không đè lên client 1.
- [x] RPC `player_move` qua `client.OnRPC` (Centrifuge RPC, method `"player_move"`) — **không** dùng `OnPublish` tạm (đi thẳng Hướng B theo khuyến nghị). Validate: resolve character từ `client.UserID()` đã xác thực (không tin `characterId` client gửi), đã join room chưa, trong map bounds, tốc độ hợp lý (so với `LastSeenAt`), `minDistance` 24px với player khác. Hợp lệ → update `RoomStore` + `node.Publish` broadcast `player_move` vào `room:<mapCode>`. Không hợp lệ → publish `player_position_correction` vào personal channel, **không** update `RoomStore`, RPC vẫn ack thành công (correction đi kênh riêng, không qua response RPC — đúng quyết định đã chốt).
- [x] Personal channel: dùng **server-side subscription** của Centrifuge (`ConnectReply.Subscriptions`, không phải tự dựng cơ chế) — mọi client tự động nhận publication trên `personal:<userID>` ngay từ lúc connect, không cần tự subscribe. Xác nhận qua `node.Subscribe`/`ConnectReply.Subscriptions` là API chuẩn của centrifuge-go v0.38.0 (đã đọc source để chắc chắn, không đoán).
- [x] Đổi default room channel từ hardcode `room:starter-town` sang `room:<map.code>` thật (`room:village_adventure`) — đã làm ở mục C qua `RealtimeUsecase.GetBootstrap`.
- [x] `player_joined`/`player_left` broadcast khi subscribe/unsubscribe; `room_snapshot` gửi riêng cho client vừa join qua `SubscribeReply.Options.Data` (không broadcast cho cả room).
- [x] `go build`/`go vet` sạch. **Đã test end-to-end thật bằng WebSocket client** (script Node dùng `centrifuge-js`, 2 user thật, không phải suy luận): join snapshot đúng, anti-overlap spawn đúng, move hợp lệ broadcast đúng tới người khác, move quá nhanh/ra ngoài map/đè người khác đều bị reject đúng lý do qua personal channel, `player_left` broadcast đúng khi unsubscribe.
- [ ] **Lưu ý nhỏ phát hiện khi test**: client vừa join sẽ nhận luôn `player_joined` của chính mình (vì broadcast xảy ra sau khi subscribe ack xong). Không phải bug, nhưng FE (mục G) cần xử lý idempotent (bỏ qua hoặc no-op nếu `characterId` trùng chính mình) thay vì tạo sprite trùng.
- [ ] **Giới hạn MVP đã ghi nhận**: `OnDisconnect` giả định chỉ có 1 room/map tại một thời điểm (dùng `DefaultRoomID()`), chưa track client đang ở room nào — đủ dùng cho MVP 1 map, cần sửa lại nếu sau này có nhiều room cùng lúc.

### G. FE nối movement thật
- [x] `movementSystem.ts` thêm `createMovementThrottle`/`recordMovement`/`tickMovementThrottle`/`flushMovementThrottle` (throttle 100ms + coalesce latest event, gửi ngay không đợi tick khi vừa dừng). Nối vào `GameScene.update()`: local render ngay bằng Arcade velocity, gửi qua **Centrifuge RPC `player_move`** (`centrifuge.rpc()`, không phải `subscription.publish` — khớp OnPublish deny-all đã áp từ Phase 1).
- [x] Remote player: `GameScene` giữ `Map<characterId, Sprite>`, dựng sprite từ `room_snapshot` (lúc join) + `player_joined`/`player_left`, tween 100ms khi nhận `player_move` cho player khác (dùng thẳng Phaser tween theo đúng ví dụ trong `docs/Phaser-Frontend-Guide.md` mục 11, không dùng `interpolationSystem.ts` cũ vì Phaser tween đã tự lo phần nội suy — file cũ giữ nguyên, không xoá, chỉ không wire vào).
- [x] Correction (`player_position_correction` qua personal channel): snap thẳng local player về vị trí server trả (không tween, tránh cảm giác trôi lệch tiếp), đồng thời xoá `latestMovement` đang chờ gửi để không ghi đè lại correction ngay tick sau.
- [x] Sửa nốt hardcode `room:starter-town` còn sót ở FE (`gameSocket.ts`) — giờ bắt buộc truyền `channel` tường minh lấy từ `bootstrap.default_channel`, không còn giá trị mặc định sai.
- [x] `go build`/`go vet`/`npm run build` sạch. **Đã test lại toàn bộ luồng RPC/correction/broadcast qua script WebSocket thật ở mục F** (cùng wire format `centrifuge.rpc('player_move', ...)` mà FE dùng) — xác nhận backend nhận đúng.
- [ ] **Chưa xác nhận bằng mắt trên browser thật**: throttle 100ms, tween remote player, animation theo hướng, correction snap — tất cả mới verify bằng đọc code + test backend qua script, chưa chạy được trong trình duyệt thật (không có công cụ browser). Cần bạn tự mở `http://localhost:5173`, đăng nhập 2 tab, thử di chuyển để xác nhận mượt/đúng hướng trước khi coi bước này "xong" hoàn toàn.
- [ ] **Giới hạn đã biết**: `ChatPanel.vue` và `GameScene` mỗi bên tự mở 1 Centrifuge connection riêng (2 connection cho cùng 1 user) — không sai nhưng dư, gây "double join" vô hại lúc mount (JoinRoom gọi 2 lần, lần sau ghi đè lần trước). Chưa gộp làm 1 connection dùng chung ở batch này; cân nhắc gộp qua `game.store.ts` nếu thấy cần sau.

### H. ChatPanel — expand/collapse bên phải
- [x] Vị trí hiện tại (`GameView.vue` đã đặt `aside.game-overlay` ở top-right) — giữ nguyên, đúng yêu cầu "bên phía tay phải".
- [x] Thêm state `collapsed` cho `ChatPanel.vue`: nút toggle (▾/▸) trên header; collapsed chỉ còn header (grid-template-rows: auto), expand hiện đầy đủ khung chat + input như cũ.
- [x] Check "mine" dùng `characterId` (`gameStore.characterId`) — **đã đúng sẵn** từ code Phase 1 audit ở mục A, không phải sửa lại (bug `authStore.userId` cũ đã được thay từ trước).
- [x] `npm run build` sạch.

### I. NPC hiện tại trong village_adventure.tmj
- [x] Ghi rõ trong `GameScene.ts` (comment tại `create()`): layer `NPCSpawns` hiện tại (animal/villager) là flavor, cố tình không đọc/render, không đụng `npc_types`/`map_npc_spawns`/`enemy_hit`.
- [x] Enemy combat thật sẽ có spawn riêng (thêm sau, không nằm trong batch này) — backend cũng chưa có code nào đọc `map_npc_spawns` (bảng đang trống), nhất quán với quyết định này.

---

## 4. Tổng kết batch

Toàn bộ A→I đã xong về code, build/test được xác nhận qua: `go build`/`go vet` sạch nhiều lần, `vue-tsc -b`/`npm run build` sạch, test HTTP end-to-end (register/login/characters/me/bootstrap/map-resync), test WebSocket end-to-end bằng script Node thật (join/anti-overlap spawn/movement hợp lệ+reject/correction/leave).

## 6. Audit kiến trúc BE so với `docs/Architecture.md`

Đối chiếu toàn bộ module mới (`character`, `chat`, `realtime/room`, `realtime/port`, `realtime/usecase`) với `Architecture.md` mục 6-7 và `Realtime-Room-State-Decisions.md` (được mục 7 tham chiếu chi tiết). Kết quả:

**Đã đúng, không cần sửa:**
- Layering Clean Architecture: usecase không import `gin`/`centrifuge` ở bất kỳ đâu (đã grep xác nhận) — transport/delivery mới là nơi phụ thuộc framework.
- `RoomStore` interface + `MemoryRoomStore` nằm trong `realtime/room/` đúng file layout mà `Realtime-Room-State-Decisions.md` mục 12 liệt kê từng bước.
- Gameplay dùng Centrifuge RPC (Hướng B) thay vì `OnPublish` tạm — đúng khuyến nghị mục 8.
- Wire format `player_move`/`room_snapshot` khớp ví dụ JSON trong doc.

**2 chỗ lệch đã sửa:**
1. `RoomPlayer` dùng field `UserID` (không đọc lại ở đâu — dữ liệu chết) thay vì `ClientID` như model đã chốt ở mục 3 (Realtime-Room-State-Decisions.md). Đã đổi thành `ClientID`, lưu đúng `client.ID()` (Centrifuge connection ID) thay vì `client.UserID()`. `RoomUsecase.JoinRoom` nhận thêm tham số `clientID`.
2. Event `player_position_correction` thiếu field `characterId` so với ví dụ JSON ở mục 6. Đã thêm `CharacterID` vào `MovementRejection` (usecase) và `positionCorrectionEvent` (transport DTO), cùng type FE `PlayerPositionCorrectionEvent`.

Đã build/vet lại sạch và test WebSocket end-to-end lại từ đầu — xác nhận `characterId` xuất hiện đúng trong correction payload thực tế.

**Ghi chú không phải lỗi** (đã cân nhắc rồi quyết định giữ nguyên):
- `MapInfo` entity đặt trong `character/entity` dù về khái niệm thuộc "map" — do doc không định nghĩa module `map` riêng và `character` là nơi cần dữ liệu này đầu tiên; không phải vi phạm layering, chỉ là chỗ đặt file có thể gọn hơn nếu sau này tách module `map` riêng.
- `RoomStore.LeaveRoom` không nhận `clientID` (chỉ `characterID`) — đúng y hệt signature doc đề xuất ở mục 4; không track "connection nào đang sở hữu room-player slot" khi rời phòng. Nghĩa là nếu cùng 1 user mở 2 tab, tab đóng trước có thể xoá nhầm player dù tab kia vẫn còn — edge case hẹp (không ảnh hưởng test 2 user khác nhau), doc cũng không yêu cầu xử lý ở mức interface này nên chưa làm thêm.

**Việc còn lại duy nhất cần bạn tự làm**: mở `http://localhost:5173` bằng trình duyệt thật để xác nhận bằng mắt (không có công cụ browser để tự làm việc này) — sprite render đúng, animation đúng hướng, tile không lệch, collision không xuyên tường, di chuyển 2 tab thấy nhau mượt, chat gửi/nhận được, nút expand/collapse hoạt động.

## 5. Sửa sau khi bạn test

- **Chat panel collapse chưa đóng khung ngoài**: lỗi CSS Grid — `.chat-panel` nằm trong `GameView.vue`'s `.game-overlay` (grid `auto 1fr`), mặc định grid item stretch full chiều cao track `1fr`. Đã thêm `align-self: start` cho `.chat-panel.collapsed` để khung ngoài co theo nội dung (chỉ còn header) thay vì giữ nguyên chiều cao track.
- **Refactor `GameScene.ts`** (291 dòng → 101 dòng) để dễ mở rộng khi thêm NPC/combat/HP bar sau này, theo đúng gợi ý "khi dài ra thì tách sang systems/" đã có sẵn trong `docs/Phaser-Frontend-Guide.md` mục 19:
  - `systems/mapSystem.ts` — dựng tilemap/tileset/layer/collision group (`buildMap()`).
  - `systems/localPlayerController.ts` — sprite + input + animation + throttle + gửi RPC + correction/resync của local player (class `LocalPlayerController`).
  - `systems/remotePlayerManager.ts` — quản lý sprite các player khác theo `characterId` (class `RemotePlayerManager`).
  - `network/gameSocket.ts` — chuyển toàn bộ type-guard "raw JSON → event đã gõ kiểu" vào đây, expose callback riêng theo từng loại event (`onRoomSnapshot`/`onPlayerJoined`/`onPlayerLeft`/`onPlayerMove`/`onPlayerChat`/`onCorrection`) thay vì 1 callback `onRoomEvent(event: unknown)` chung rồi tự đoán ở nơi gọi.
  - `GameScene.ts` giờ chỉ còn wiring: gọi các system trên, nối callback, `update()` chỉ gọi `localPlayer.update(...)`.
  - Xoá `systems/interpolationSystem.ts` (dead code, 0 chỗ dùng — đã thay bằng Phaser tween trực tiếp trong `RemotePlayerManager`).
  - Khi thêm feature mới (NPC, combat, HP bar, chat bubble...): thêm 1 system mới (class hoặc hàm thuần) trong `systems/`, wire vào `GameScene.create()` — không sửa trực tiếp logic có sẵn trong các system khác.
  - Lưu ý kỹ thuật: tsconfig frontend bật `erasableSyntaxOnly` — **không dùng** constructor parameter property shorthand (`constructor(private x: T)`), phải khai báo field rồi gán tay trong constructor.

---

## 3. Ngoài phạm vi batch này

- Enemy NPC thật (spawn, HP, `enemy_hit`, reward, animation từ `Player_Actions.png`/spritesheet enemy riêng).
- Debounced position persistence (`characters.last_x/last_y`) khi đứng yên/rời room/logout.
- Avatar builder, shop, inventory UI.
- Teams SSO auto-login trong `GameView`.
