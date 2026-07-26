# BigTown

**Web game 2D multiplayer real-time** — nhiều người chơi cùng vào một map chung, tạo nhân vật pixel art, di chuyển thấy nhau theo thời gian thực, chat trong game, đánh NPC lấy điểm và leo bảng xếp hạng.

🔗 Demo: [big-town.vercel.app](https://big-town.vercel.app)

BigTown là bản MVP nhằm kiểm chứng phần lõi của sản phẩm: **đồng bộ vị trí real-time giữa nhiều người chơi, vòng lặp gameplay đơn giản và dữ liệu người chơi được lưu bền vững**. Ứng dụng hỗ trợ cả đăng nhập thường (email/password) lẫn **Microsoft Teams SSO**, hướng tới việc nhúng vào Microsoft Teams.

---

## Mục lục

- [Tính năng](#tính-năng)
- [Công nghệ](#công-nghệ)
- [Kiến trúc tổng thể](#kiến-trúc-tổng-thể)
- [Các màn hình / view](#các-màn-hình--view)
- [Cấu trúc thư mục](#cấu-trúc-thư-mục)
- [Mô hình real-time](#mô-hình-real-time)
- [Cơ sở dữ liệu](#cơ-sở-dữ-liệu)
- [API](#api)
- [Chạy dev](#chạy-dev)
- [Biến môi trường](#biến-môi-trường)
- [Triển khai](#triển-khai)
- [Trạng thái & lộ trình](#trạng-thái--lộ-trình)
- [Tài liệu chi tiết](#tài-liệu-chi-tiết)

---

## Tính năng

- **Đăng nhập & hồ sơ người chơi** — tài khoản local (email/password) hoặc Teams SSO; mỗi user có một nhân vật với điểm và tiền được lưu trong database.
- **Tạo & tuỳ biến nhân vật** — chọn lớp nhân vật từ các sprite pixel art có sẵn (hunter, knight, tanker, wizard).
- **Map 2D multiplayer real-time** — nhiều người chơi cùng map, di chuyển bằng phím, thấy nhau di chuyển mượt qua interpolation; hiển thị tên nhân vật trên đầu, tán cây tự mờ đi khi che người chơi.
- **Chat trong game** — panel chat có thể thu/mở, tin nhắn broadcast tới cả phòng và hiển thị bubble trên đầu nhân vật.
- **Leaderboard** — bảng xếp hạng theo điểm.
- **Âm thanh** — nhạc nền và sound effect.
- **NPC** — hiện có NPC "flavor" (động vật/dân làng) trong map. *Enemy combat (đánh NPC lấy điểm) đang trong lộ trình, chưa hoàn thiện.*

---

## Công nghệ

### Frontend
- **Vue 3** + **TypeScript** + **Vite**
- **Phaser 4** — game engine (render map, sprite, animation, camera, collision, game loop)
- **Pinia** — state management chia sẻ giữa Vue và game
- **centrifuge-js** — client WebSocket real-time
- **axios** — HTTP client

### Backend
- **Go 1.26** + **Gin** — HTTP API
- **Centrifuge** (`github.com/centrifugal/centrifuge`) — WebSocket transport, channel subscription, publish/broadcast
- **pgx** — driver PostgreSQL
- **JWT** — access/refresh token; **bcrypt** — hash mật khẩu

### Hạ tầng
- **PostgreSQL** — lưu trữ bền vững
- **Nginx** — reverse proxy, TLS termination, WSS (khi self-host / chạy trong Teams)
- **Docker Compose** — Postgres + auto-migrate schema

---

## Kiến trúc tổng thể

```text
Client Browser / Microsoft Teams
  │  HTTPS / WSS
  ▼
Nginx  (terminate TLS, serve static FE, proxy /api và /connection/websocket)
  │  HTTP / WS nội bộ
  ▼
Go Backend  (REST API + Centrifuge WebSocket + realtime room state trong RAM)
  │
  ▼
PostgreSQL
```

Backend đi theo mô hình **Vertical Modular Monolith + Clean Architecture nhẹ + Repository Pattern**. Mỗi module tự chứa đủ các tầng `entity / port / usecase / repository / delivery`. Luồng phụ thuộc một chiều:

```text
Router (Gin) → Delivery Handler → Usecase → Port (interface) → Repository → *sql.DB
```

Frontend theo **feature-based**, tách rõ ranh giới: **Vue** lo app shell và UI overlay (auth, chat, leaderboard); **Phaser** lo render và game loop (map, sprite, animation, collision).

---

## Các màn hình / view

| Route | View | Mô tả |
|---|---|---|
| `/` | Home | Trang chủ / landing |
| `/login` | `LoginView` | Đăng nhập email/password (và Teams SSO) |
| `/register` | `RegisterView` | Đăng ký tài khoản local |
| `/character/create` | `CharacterCreateView` | Tạo & chọn lớp nhân vật lần đầu |
| `/game` | `GameView` | Màn chơi chính — canvas Phaser + overlay chat/leaderboard |
| `/403` | `ForbiddenView` | Không đủ quyền truy cập |
| `/*` | `NotFoundView` | Route không tồn tại |

`GameView` là màn hình trung tâm: bên trong nó `GameCanvas.vue` mount instance Phaser thật, còn các overlay Vue (`ChatPanel`, `LeaderboardPanel`, `MapSwitcher`, `AudioSettingsPanel`) nằm chồng lên trên canvas.

---

## Cấu trúc thư mục

```text
BigTown/
├── asset/              # Asset gốc (pixel art), tool embed tileset
├── backend/            # Go API + Centrifuge realtime
│   ├── cmd/server/     # Entry point
│   └── internal/
│       ├── app/        # Bootstrap, container, wiring
│       ├── middleware/ # CORS, auth, error, logger, recovery, requestID
│       ├── security/   # JWT, password, token
│       ├── database/   # schema.sql, seed.sql, postgres.go
│       ├── platform/config/
│       └── module/     # auth, user, character, chat, realtime, leaderboard
├── frontend/           # Vue 3 + Phaser
│   ├── public/assets/  # maps (.tmj), tiles, player sprites, sounds
│   └── src/
│       ├── app/        # router, layouts, providers, views chung
│       ├── shared/     # api (http/token), audio, components, utils
│       └── features/
│           ├── auth/   # components / services / stores / views
│           └── game/   # phaser / systems / network / services / stores / views
└── docs/               # 9 tài liệu thiết kế (xem cuối README)
```

---

## Mô hình real-time

BigTown dùng mô hình **server-authoritative** (server là nơi quyết định cuối cùng):

1. **Client optimistic render** — Phaser vẽ local player ngay khi bấm phím để phản hồi tức thì.
2. **Throttled publishing + latest-event coalescing** — không gửi mỗi frame; chỉ giữ movement mới nhất và publish tối đa mỗi ~100ms.
3. **Server validate** tốc độ, biên map và chống trùng vị trí (`minDistance = 24px`), rồi mới broadcast. Gameplay đi qua **Centrifuge RPC**, không cho client publish trực tiếp.
4. **Correction** khi bị reject được gửi riêng qua **personal channel**, tách khỏi broadcast của phòng.
5. **Remote interpolation** — client khác dùng tween 100ms để hiển thị mượt.
6. **Debounced persistence** — vị trí chỉ ghi DB khi đứng yên/rời phòng/logout, không ghi mỗi tick.

Chat đi theo hướng khác: **HTTP POST → lưu DB → publish Centrifuge**, để kiểm soát nội dung và giữ lịch sử.

> 💡 **Đổi map** chỉ cần thay một biến môi trường `GAME_DEFAULT_MAP_CODE`. Backend tự đồng bộ lại map cho mọi người chơi ở lần login kế tiếp — không cần sửa code. Chi tiết trong `docs/Architecture.md` mục 9.1.

---

## Cơ sở dữ liệu

PostgreSQL là nguồn sự thật cho tài sản và tiến trình dài hạn; RAM của Go giữ trạng thái phòng đang chạy (vị trí, HP, cooldown, AI state của NPC); frontend chỉ render và gửi input.

Các bảng chính: `app_user`, `credential`, `refresh_token`, `token_blacklist`, `user_identities`, `characters`, `items`, `player_items`, `character_equipment`, `maps`, `npc_types`, `map_npc_spawns`, `reward_events`, `chat_messages`.

Schema được auto-migrate qua Docker Compose khi khởi tạo container Postgres.

---

## API

Tất cả endpoint có prefix `/api`. Route bảo vệ cần header `Authorization: Bearer <access_token>`.

**Auth (public)**
```text
POST /api/auth/register    { full_name, email, password }
POST /api/auth/login       { email, password }
POST /api/auth/teams       { sso_token }          # Teams SSO
POST /api/auth/refresh                            # cookie refresh_token
```

**Auth (protected)**
```text
POST /api/auth/logout
```

**Gameplay (protected)**
```text
GET  /api/characters/me
GET  /api/characters/options
POST /api/characters
GET  /api/realtime/bootstrap
GET  /api/leaderboard
GET  /api/rooms/:roomId/chat/messages
POST /api/rooms/:roomId/chat/messages
GET  /api/users                    # role Admin
```

**Realtime**
```text
WS   /connection/websocket         # Centrifuge, token = access_token
```

---

## Chạy dev

Yêu cầu: **Go 1.26+**, **Node.js**, **Docker**.

### 1. Backend

```sh
cd backend
cp .env.example .env          # sửa JWT_SECRET trước khi dùng thật
docker compose up -d          # Postgres + tự migrate schema.sql + seed.sql
go run ./cmd/server           # -> http://localhost:8080
```

### 2. Frontend

```sh
cd frontend
cp .env.example .env
npm install
npm run dev                   # -> http://localhost:5173
```

### Cổng mặc định

| Dịch vụ | Cổng |
|---|---|
| Backend (Go) | 8080 |
| Frontend (Vite) | 5173 |
| PostgreSQL | 5432 |
| Nginx (tuỳ chọn) | 8088 |

### Kiểm tra (backend)

```sh
go build ./...
go vet ./...
gofmt -l .
```

---

## Biến môi trường

### Backend (`backend/.env`)

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=postgres
DB_SSLMODE=require
SERVER_PORT=8080
CORS_ALLOWED_ORIGINS=http://localhost:5173
COOKIE_SECURE=false          # prod cross-site: true
COOKIE_SAME_SITE=Lax         # prod cross-site: None
JWT_SECRET=change-me
TEAMS_CLIENT_ID=
TEAMS_TENANT_ID=common
GAME_DEFAULT_MAP_CODE=village_adventure
```

> ⚠️ Khi deploy FE và BE khác domain (ví dụ Vercel + Render), **bắt buộc** đặt `COOKIE_SECURE=true` và `COOKIE_SAME_SITE=None`, nếu không cookie `refresh_token` sẽ không được gửi kèm request cross-site (F5 mất session, logout lỗi).

### Frontend (`frontend/.env`)

```env
VITE_API_BASE_URL=http://localhost:8080/api
```

Khi chạy sau Nginx cùng origin, dùng `VITE_API_BASE_URL=/api`.

---

## Triển khai

- **Frontend:** Vercel (static build từ Vite).
- **Backend:** Render (hoặc bất kỳ host Go + Postgres nào).
- **Self-host / Teams:** đặt Nginx phía trước để terminate TLS và proxy `/api` + `/connection/websocket` — xem `docs/Nginx-Deployment-Guide.md`.

---

## Trạng thái & lộ trình

### Đã hoạt động
- Đăng nhập local + Teams SSO, tự tạo nhân vật.
- Map `village_adventure` render bằng Phaser.
- Multiplayer real-time (di chuyển + chat), tên nhân vật, fade tán cây.
- Leaderboard (read model từ `characters.score`).
- Hot-path di chuyển đã tối ưu chạy hoàn toàn trong RAM.

### Đang trong lộ trình
- Enemy NPC combat thật (đánh NPC → HP → reward).
- Debounced position persistence (`characters.last_x/last_y`).
- Avatar builder / shop / inventory UI.
- Teams SSO auto-login trong `GameView`.
- Gộp 2 kết nối Centrifuge (ChatPanel + GameScene) thành một.
- Scale nhiều backend node qua Centrifuge Redis broker.

---

## Tài liệu chi tiết

Toàn bộ thiết kế và quyết định kỹ thuật nằm trong thư mục `docs/`:

| Tài liệu | Nội dung |
|---|---|
| `Architecture.md` | Tổng quan dự án, phạm vi MVP, deployment view, cách đổi map |
| `Storage-Design.md` | Thiết kế lưu trữ 3 lớp (Postgres / RAM / frontend), schema |
| `Realtime-Room-State-Decisions.md` | Quyết định về RoomStore, movement validation, chống trùng vị trí |
| `Realtime-Performance-Techniques.md` | Các kỹ thuật realtime & hiệu năng tổng quát |
| `Realtime-Performance-Fixes.md` | Chi tiết các bug đã fix trong đợt debug production |
| `Phaser-Frontend-Guide.md` | Hướng dẫn triển khai map, nhân vật, realtime UI bằng Phaser |
| `Nginx-Deployment-Guide.md` | Reverse proxy, HTTPS/WSS, chuẩn bị cho Teams |
| `Movement-Chat-Spawn-Plan.md` | Nhật ký triển khai spawn / movement / chat |
| `Session-Handoff.md` | Tóm tắt trạng thái dự án qua các session |

Backend còn có `backend/ARCHITECTURE_GUIDE.md` — **đọc trước khi thêm module nghiệp vụ mới**.