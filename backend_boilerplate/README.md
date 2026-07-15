# Backend Boilerplate

Copy folder này sang project khác để bắt đầu — không phải một phần của IT Asset & Hardware Tracking
System, chỉ dùng chung git repo cho tiện dev. **Không import gì từ `../backend`.**

Đã có sẵn, chạy được thật (không phải chỉ có structure rỗng):

- `internal/app`, `internal/middleware`, `internal/apperror`, `internal/response`, `internal/security`,
  `internal/database`, `internal/platform/config` — hạ tầng dùng chung, copy nguyên từ project gốc,
  không đổi gì (domain-agnostic sẵn).
- `internal/module/user` — module profile mẫu (CRUD tối thiểu: `GET /api/users`, role `Admin`).
- `internal/module/auth` — JWT login/register/refresh/logout đầy đủ, dùng transaction thật
  (`db.BeginTx`) khi Register ghi cả `user` lẫn `credential`, và minh hoạ cross-module interface
  (`UserReader`) để gọi sang module `user`.
- `internal/module/example` — module mẫu **in-memory** (không cần DB) để copy-paste khi thêm module
  mới, đã đăng ký sẵn trong `app.go` nên chạy `go run ./cmd/server` là gọi thử được ngay.
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
GET  /api/example/items
POST /api/example/items   { "name": "..." }
```

## Việc cần làm khi mang sang project mới

1. Đổi tên (tuỳ chọn): `go mod edit -module <ten-moi>` rồi tìm-thay `"backend/` → `"<ten-moi>/` trong
   toàn bộ `.go` files. Nếu giữ tên `backend` thì không cần làm gì.
2. Xoá `internal/module/example` khi không cần module mẫu nữa (xem
   `internal/module/example/README.md`).
3. Sửa `internal/database/schema.sql` để thêm bảng nghiệp vụ thật — **không sửa 4 bảng có sẵn**
   (`app_user`, `credential`, `refresh_token`, `token_blacklist`) trừ khi cũng sửa code
   `module/user` + `module/auth` cho khớp.
4. Đọc `ARCHITECTURE_GUIDE.md` trước khi thêm module nghiệp vụ đầu tiên — có checklist + danh sách
   không nên làm ở cuối file.
5. `middleware.CORSMiddleware()` đang hardcode origin `http://localhost:5173` — sửa theo frontend
   thật của bạn.

## Lệnh kiểm tra

```sh
go build ./...
go vet ./...
gofmt -l .
```
