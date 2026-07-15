# Module `example` — boilerplate copy-paste, không phải nghiệp vụ thật

Module này tồn tại để **copy folder rồi đổi tên** khi bắt đầu module mới, thay vì đọc
`ARCHITECTURE_GUIDE.md` và tự gõ lại từ đầu. Nó thực hiện đúng chain:

```
routes.go → delivery/handler.go → usecase/*.go → port/repository.go → repository/memory_repository.go
```

Cố tình khác 1 chỗ so với `user`/`auth`: **repository dùng in-memory map (`sync.RWMutex` +
`map[string]entity.Item`), không dùng `*sql.DB`**, để module này build và chạy được ngay không cần
Postgres. Khi copy sang module thật, việc đầu tiên cần đổi là viết lại `repository/*.go` bằng
`*sql.DB` giống `internal/module/user/repository/user_repository.go`.

Module **đã được đăng ký** trong `internal/app/app.go` (khác với repo gốc mà file này được nhân bản
từ đó) vì boilerplate này không phải service production — cứ `go run ./cmd/server` là gọi thử được
ngay:

```
GET  /api/example/items
GET  /api/example/items/:id
POST /api/example/items    { "name": "Cái gì đó" }
```

Xoá cả folder `internal/module/example` + dòng đăng ký nó trong `app.go` khi không cần nữa.

## Transaction / cross-module thật thì xem ở đâu

Đọc `internal/module/auth/usecase/register.go` (transaction 2 bảng, xem mục 6
`ARCHITECTURE_GUIDE.md`) và `internal/module/auth/port/user_reader.go` +
`internal/module/auth/provider.go` (cross-module interface tự định nghĩa, xem mục 5).
