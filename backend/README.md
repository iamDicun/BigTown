# BigTown Backend

Backend Golang cho BigTown MVP. Cấu trúc giữ theo Clean Architecture dạng module trong `internal/module`.

Đã có sẵn, chạy được thật (không phải chỉ có structure rỗng):

- `internal/app`, `internal/middleware`, `internal/apperror`, `internal/response`, `internal/security`,
  `internal/database`, `internal/platform/config` — hạ tầng dùng chung, copy nguyên từ project gốc,
  không đổi gì (domain-agnostic sẵn).
- `internal/module/user` — module profile người dùng (CRUD tối thiểu: `GET /api/users`, role `Admin`).
- `internal/module/auth` — JWT login/register/refresh/logout và Teams SSO login đầy đủ, dùng transaction thật
  (`db.BeginTx`) khi Register ghi cả `user` lẫn `credential`, và minh hoạ cross-module interface
  (`UserReader`) để gọi sang module `user`.
- `internal/module/realtime` — Centrifuge WebSocket transport, room channel bootstrap và realtime boundary.
- `internal/module/leaderboard` — read model bảng xếp hạng từ `characters.score`.
- `docker-compose.yml` — Postgres, tự chạy `internal/database/schema.sql` lúc khởi tạo container.

## Chạy thử

```sh
cp .env.example .env      # sửa JWT_SECRET trước khi dùng thật
docker compose up -d      # Postgres + tự migrate schema.sql
go run ./cmd/server
```

```
POST /api/auth/register   { "full_name": "...", "email": "...", "password": "12345678" }
POST /api/auth/login      { "email": "...", "password": "12345678" }
POST /api/auth/teams      { "sso_token": "<teams_sso_token>" }
POST /api/auth/refresh    (cookie refresh_token tự gửi kèm)
POST /api/auth/logout     (Bearer access_token + cookie refresh_token)
GET  /api/users           (Bearer access_token, role Admin — role mặc định khi register là "User",
                            tự update role trong DB nếu muốn test route này)
GET  /api/realtime/bootstrap (Bearer access_token)
GET  /api/leaderboard        (Bearer access_token)
WS   /connection/websocket   (Centrifuge token = access_token)
```

## Việc cần làm tiếp cho BigTown

1. Cấu hình `TEAMS_CLIENT_ID` và `TEAMS_TENANT_ID` khi chạy trong Microsoft Teams.
2. Implement repository/usecase cho `characters`, `items`, `inventory`, `equipment`, `maps`, `npc_types`.
3. Thêm validation gameplay vào `internal/module/realtime` trước khi cho client publish movement/combat thật.
4. Đọc `ARCHITECTURE_GUIDE.md` trước khi thêm usecase nghiệp vụ mới.
5. `middleware.CORSMiddleware()` đang hardcode origin `http://localhost:5173` — sửa theo frontend
   thật của bạn.

## Lệnh kiểm tra

```sh
go build ./...
go vet ./...
gofmt -l .
```
