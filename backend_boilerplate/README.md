# BigTown Backend

Backend Golang cho BigTown MVP. Cấu trúc giữ theo Clean Architecture dạng module trong `internal/module`.

Đã có sẵn, chạy được thật (không phải chỉ có structure rỗng):

- `internal/app`, `internal/middleware`, `internal/apperror`, `internal/response`, `internal/security`,
  `internal/database`, `internal/platform/config` — hạ tầng dùng chung, copy nguyên từ project gốc,
  không đổi gì (domain-agnostic sẵn).
- `internal/module/user` — module profile người dùng (CRUD tối thiểu: `GET /api/users`, role `Admin`).
- `internal/module/auth` — JWT login/register/refresh/logout đầy đủ, dùng transaction thật
  (`db.BeginTx`) khi Register ghi cả `user` lẫn `credential`, và minh hoạ cross-module interface
  (`UserReader`) để gọi sang module `user`.
- `internal/module/game` — shell module cho MVP game: bootstrap, leaderboard placeholder, domain/runtime types.
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
POST /api/auth/refresh    (cookie refresh_token tự gửi kèm)
POST /api/auth/logout     (Bearer access_token + cookie refresh_token)
GET  /api/users           (Bearer access_token, role Admin — role mặc định khi register là "User",
                            tự update role trong DB nếu muốn test route này)
GET  /api/game/bootstrap  (Bearer access_token)
GET  /api/game/leaderboard (Bearer access_token)
```

## Việc cần làm tiếp cho BigTown

1. Implement repository/usecase cho `characters`, `items`, `inventory`, `equipment`, `maps`, `npc_types`.
2. Thêm WebSocket endpoint và Hub/Client runtime cho `internal/module/game`.
3. Đọc `ARCHITECTURE_GUIDE.md` trước khi thêm usecase nghiệp vụ mới.
4. `middleware.CORSMiddleware()` đang hardcode origin `http://localhost:5173` — sửa theo frontend
   thật của bạn.

## Lệnh kiểm tra

```sh
go build ./...
go vet ./...
gofmt -l .
```
