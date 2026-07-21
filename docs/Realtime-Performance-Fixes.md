# Cải thiện Realtime & Thời gian load — chi tiết đã làm

Ghi lại cụ thể các thay đổi đã thực hiện trong đợt debug production (BE Render + FE Vercel: lag di
chuyển, trễ 4-5s, giật vị trí, state reset, load chậm). Đối chiếu với `docs/Realtime-Room-State-Decisions.md`
và `docs/Movement-Chat-Spawn-Plan.md` khi cần biết quyết định gốc.

---

## 1. Player join/leave bị nhân đôi, state reset khi có nhiều connection cùng lúc

**Hiện tượng gốc**: 1 người chơi lúc thấy lúc không thấy người khác, va chạm lúc check lúc không,
state phòng thỉnh thoảng bị reset.

**Nguyên nhân**: Mỗi tab/component (`ChatPanel`, `GameScene`) tự mở 1 kết nối Centrifuge riêng cho
cùng 1 user. Server coi mỗi kết nối là 1 "player join" độc lập; khi 1 trong 2 kết nối disconnect,
handler xoá luôn player khỏi `RoomStore` dù kết nối còn lại vẫn sống.

**Đã sửa** (`backend/internal/module/realtime/room/state.go`, `memory_store.go`):
- `GameRoom` có thêm `Clients map[string]map[string]struct{}` — 1 `CharacterID` map tới **tập hợp**
  `ClientID` (Centrifuge connection ID), không phải 1-1.
- `JoinRoom`: nếu character đã tồn tại trong room, chỉ thêm `ClientID` vào tập hợp, **không** ghi đè
  vị trí, **không** phát `player_joined` lần 2.
- `LeaveRoom`: chỉ thực sự xoá player khỏi room khi tập hợp `ClientID` rỗng (không còn kết nối nào).

**Bug còn sót lại đã fix thêm** (`room_usecase.go`, `memory_store.go` `JoinRoom`): quyết định
`isFirstConnection` trước đó được tính từ 1 lần đọc snapshot **trước** khi insert, không cùng 1 lock
với thao tác ghi → 2 kết nối join gần như đồng thời của cùng user vẫn có thể cùng thấy "chưa tồn
tại" và cùng báo `isFirstConnection = true`. Sửa bằng cách chuyển toàn bộ quyết định này vào bên
trong `MemoryRoomStore.JoinRoom`, cùng 1 lock với việc insert/lookup.

**Bug phụ đã fix cùng lúc**: `resolveSpawnPosition`/`isOccupied` (`room_usecase.go`) không loại trừ
chính character đang join ra khỏi danh sách "vị trí đã bị chiếm" → thêm tham số `excludeCharacterID`.

---

## 2. Đi xuyên qua người chơi khác trước khi bị đẩy lại (server correction tới sau)

**Hiện tượng gốc**: Nhân vật khác trên màn hình mình có thể đi xuyên qua rồi mới bị "giật" về đúng vị
trí, dù backend đã validate va chạm.

**Nguyên nhân**: Đây không phải bug — là hệ quả tất yếu của mô hình server-authoritative (client tự
vẽ ngay theo input, server validate bất đồng bộ rồi mới sửa nếu sai). Ở production, RTT thật khiến độ
trễ giữa "server phát vị trí mới" và "client khác nhận được" lớn hơn hẳn local, nên khoảng thời gian
"đi xuyên qua nhau trước khi bị sửa" lộ rõ hơn nhiều.

**Đã thêm** (không phải sửa lỗi, mà giảm cảm giác khó chịu): va chạm vật lý client-side giữa local
player và remote player (`GameScene.ts`: `this.physics.add.collider(...)`), để bị chặn ngay trên màn
hình của chính mình thay vì đợi correction từ server.

**Tác dụng phụ phát sinh và đã fix**: cách làm va chạm đầu tiên (gắn dynamic body `setImmovable(true)`
thẳng lên sprite remote player) khiến sprite đó bị **rung/giật liên tục** khi local player chạm vào —
vì dynamic body (dù immovable) vẫn bị Arcade physics tự đồng bộ lại vị trí từ transform mỗi frame,
xung đột với tween đang chạy để nội suy vị trí remote player mượt. Sửa bằng cách tách riêng
(`remotePlayerManager.ts`): sprite hiển thị giữ nguyên là `scene.add.sprite` + tween như cũ (không
gắn physics), còn va chạm dùng 1 `Zone` riêng với **static body** (snap thẳng theo vị trí server xác
nhận, không tween) — static body không tham gia bước đồng bộ mỗi frame nên không còn xung đột.

---

## 3. Fade lớp `DecorationAbove` khi che nhân vật (tính năng mới, không phải bug fix)

**Đã thêm**: `frontend/src/features/game/systems/aboveLayerFadeSystem.ts` — mỗi frame kiểm tra tile
`DecorationAbove` nào đang overlap bounding box của local player thì tween alpha xuống 0.35, tile
không còn overlap thì tween alpha về lại 1. Idempotent (tile đang đúng trạng thái không bị tween lại
liên tục). Cắm vào `GameScene.ts` (`create()`/`update()`) và `mapSystem.ts` (expose layer
`DecorationAbove` ra ngoài `buildMap()`).

---

## 4. Realtime bị trễ nặng (4-5s mới thấy vị trí cập nhật, đổi region Singapore không cải thiện)

**Hiện tượng gốc**: Di chuyển xong 4-5 giây sau người khác mới thấy cập nhật; thỉnh thoảng bị giật về
vị trí cũ. Đổi region compute BE sang Singapore chỉ giúp các API call nhanh hơn, **không** giúp
realtime.

**Nguyên nhân thật sự (đã xác nhận qua đọc code, không phải RTT WebSocket)**: Mỗi RPC `player_move`
(client gửi tối đa 10 lần/giây khi đang di chuyển liên tục) khi tới `RoomUsecase.MovePlayer` trước
đó gọi:
1. `CharacterResolver.GetOrCreateForUser(userID)` → 1 SELECT + 1 UPDATE (đồng bộ `map_id`, dù gần
   như không bao giờ đổi) vào Postgres.
2. `MapReader.GetDefaultMap()` → 1 SELECT nữa (metadata map, dữ liệu tĩnh) chỉ để lấy bounds validate.

→ 3 round-trip DB cho **mỗi gói di chuyển**, trong đó có 1 write, toàn bộ chạy đồng bộ trước khi
server trả lời/broadcast. Local: DB cùng máy, chi phí ~0, không phát hiện được. Production: mỗi
round-trip DB tốn hàng chục-hàng trăm ms → xử lý 1 tick lâu hơn chính khoảng cách giữa 2 tick (100ms)
→ RPC dồn hàng đợi ngày càng trễ, đúng khớp hiện tượng "4-5s sau mới cập nhật" và "giật về vị trí cũ"
(correction dựa trên state đã lỗi thời tới sau các move mới hơn). Đổi region compute không giúp gì vì
nghẽn nằm ở số lượt gọi DB trong logic xử lý, không phải RTT mạng tới WebSocket server.

**Đã sửa**:
- `backend/internal/module/character/usecase/character_usecase.go` — `GetDefaultMap` cache kết quả
  trong RAM (`sync.RWMutex` + con trỏ cache) sau lần query đầu tiên; các lần gọi sau đọc thẳng RAM.
- `backend/internal/module/realtime/room/state.go` — `RoomPlayer` thêm field `UserID`; `GameRoom`
  thêm index phụ `PlayersByUser map[string]string` (userID → characterID).
- `backend/internal/module/realtime/room/store.go` + `memory_store.go` — thêm method
  `GetPlayerByUserID` tra thẳng RAM (O(1) qua `PlayersByUser`), ghi index lúc `JoinRoom`, xoá index
  lúc `LeaveRoom` (chỉ khi player thực sự rời phòng, không phải rớt 1 trong nhiều `ClientID`).
- `backend/internal/module/realtime/usecase/room_usecase.go` — `MovePlayer` bỏ hẳn lời gọi
  `GetOrCreateForUser`, thay bằng `store.GetPlayerByUserID(ctx, roomID, userID)`. Kết quả: `MovePlayer`
  chạy **100% trong RAM**, không còn round-trip DB nào trên hot path mỗi tick di chuyển.

**Chưa làm / cân nhắc thêm nếu vẫn còn trễ sau khi deploy fix trên**:
- Nới lỏng `minDistancePx` (hiện 24px) hoặc thêm hysteresis (2 ngưỡng riêng cho "bắt đầu chặn" và
  "bắt đầu cho phép lại") để giảm tần suất bị correction khi RTT còn cao — chỉ mới giải thích hướng,
  chưa implement, đang chờ xem sau fix DB có còn cần thiết không.

---

## 5. Trang login/load map chậm 3-4 giây mỗi bước

**Hiện tượng gốc**: Vào màn login mất 3-4s mới hiện, bấm đăng nhập mất 3-4s mới xong, đăng nhập xong
load map thêm 3-4s nữa.

**Đã sửa** (`frontend/src/features/game/components/GameCanvas.vue`): `onMounted` trước đó gọi tuần tự
`gameStore.loadMyCharacter()` rồi mới `realtimeService.getBootstrap()` — 2 API độc lập, không cái nào
cần kết quả cái kia. Đổi sang `Promise.all([...])` chạy song song, giảm thời gian chờ load map từ
tổng 2 lệnh gọi xuống còn bằng lệnh gọi chậm nhất.

**Chưa xác nhận / nghi vấn chính**: `frontend/src/main.ts` gọi
`authStore.tryRestoreSession().finally(() => app.mount('#app'))` — toàn bộ app Vue (kể cả trang
login) bị chặn hiển thị cho tới khi request `/auth/refresh` trả lời xong. Đây là nghi vấn chính cho
"màn login cũng mất 3-4s mới hiện" nhưng **chưa sửa** — nghi ngờ nguyên nhân sâu hơn là Render free
tier cold start (ngủ sau 15 phút không hoạt động, request đầu tiên sau khi ngủ dậy chậm). Đã đề nghị
bạn tự test: reload trang 2 lần liên tiếp nhanh, nếu lần 2 nhanh hẳn thì xác nhận đúng là cold start
— khi đó hướng xử lý là nâng plan Render luôn chạy (always-on) hoặc thêm cơ chế ping giữ server thức,
không phải sửa code realtime.

---

## 6. Bug logout/F5 mất session

**Hiện tượng gốc**: Logout đôi khi không chuyển về màn login (trên prod thì luôn luôn không); F5 luôn
mất session.

**Nguyên nhân**: Cookie `refresh_token` dùng `SameSite=Lax`, `Secure=false` — đúng cho môi trường
local (FE/BE cùng miền `localhost`, khác cổng) nhưng **sai** cho production vì FE (Vercel) và BE
(Render) là 2 domain gốc hoàn toàn khác nhau (cross-site thật). `SameSite=Lax` không gửi cookie kèm
request XHR/fetch cross-site (chỉ gửi khi điều hướng trang), nên `/auth/refresh` và `/auth/logout`
trên prod luôn thiếu cookie → luôn lỗi.

**Đã sửa (code)**: `frontend/src/features/auth/stores/auth.store.ts` — `logout()` trước đó không có
`catch`, nên khi gọi `/auth/logout` lỗi (thiếu cookie), exception văng ra ngoài chặn luôn
`router.push({name:'login'})` ở `Navbar.vue` phía sau. Thêm `catch {}` rỗng để logout local (xoá
access token trong bộ nhớ) luôn thành công bất kể server có phản hồi đúng hay không.

**Chưa sửa — cần bạn tự làm trên Render dashboard (tôi không có quyền truy cập)**: đổi biến môi
trường thật `COOKIE_SAME_SITE=None` và `COOKIE_SECURE=true` cho service backend trên Render. Local
vẫn giữ nguyên `COOKIE_SECURE=false`/`COOKIE_SAME_SITE=Lax` — 2 bộ giá trị khác nhau cho 2 môi trường
là đúng, không phải thiếu nhất quán (đã ghi chú giải thích trong `backend/.env.example`).

---

## Trạng thái tổng hợp

| Vấn đề | Trạng thái |
|---|---|
| Player join/leave nhân đôi, state reset | Đã sửa |
| Race condition `isFirstConnection` | Đã sửa |
| Đi xuyên người chơi khác trước khi bị đẩy | Đã giảm nhẹ (thêm collider client-side) |
| Remote player rung/giật khi va chạm | Đã sửa (bug tự gây ra bởi bước trên, đã fix) |
| Fade `DecorationAbove` | Đã thêm (tính năng mới) |
| Realtime trễ 4-5s / giật vị trí | Đã sửa (bỏ DB khỏi hot path `MovePlayer`) — **cần deploy để xác nhận** |
| minDistance/hysteresis giảm correction dưới RTT cao | Chưa làm, chờ đánh giá sau khi deploy fix DB |
| Load login/map chậm 3-4s (song song hoá API) | Đã sửa 1 phần |
| Load login chậm do `tryRestoreSession` chặn mount / cold start Render | Chưa xác nhận nguyên nhân cuối, chưa sửa |
| Logout không navigate về login | Đã sửa (code) |
| F5/logout mất session do SameSite cross-site | Đã sửa code phía đọc lỗi; **còn thiếu bước đổi env var thật trên Render** |
