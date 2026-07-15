# BigTown Backend — Architecture Guide

Guide kiến trúc backend cho BigTown MVP. Mục tiêu là giữ code theo Vertical Modular Monolith,
Clean Architecture nhẹ và Repository Pattern để sau này thêm gameplay hoặc scale realtime dễ hơn.

## 1. Kiến trúc

**Vertical Modular Monolith + Clean Architecture nhẹ + Repository Pattern + Module Provider**

Mỗi module tự chứa `entity`, `port`, `usecase`, `repository`, `delivery`, cộng `module.go`,
`provider.go`, `routes.go`. `internal/app/app.go` chỉ gọi `xxx.NewXxxModule(...)` rồi
`RegisterPublicRoutes`/`RegisterProtectedRoutes` — không tự `New` repo/usecase/handler của module
(ngoại lệ: xem mục 5).

## 2. Cấu trúc thư mục

```txt
cmd/server/main.go          // load env, config, DB, gọi app.New, Run
internal/
  app/{app.go, container.go}
  apperror/app_error.go     // AppError + factory (BadRequest, NotFound, TokenExpired, ...)
  response/response.go      // SuccessResponse[T] / ErrorResponse — envelope chung
  middleware/                // CORS, auth (JWT + blacklist), error, logger, recovery, requestID
  security/                  // jwt.go, password.go (bcrypt), token.go (random + hash) — thuần crypto
  database/{postgres.go, schema.sql}
  platform/config/config.go  // đọc ENV, KHÔNG dùng viper
  module/
    user/     // profile (entity/port/repository/usecase/delivery + module/provider/routes)
    auth/     // JWT login/register/refresh/logout, phụ thuộc user qua UserReader interface riêng
    game/     // game MVP: bootstrap, leaderboard, domain/runtime types, sau này thêm WebSocket Hub
```

`platform/` chỉ có `config`. Middleware/database/security/response/apperror nằm thẳng trong
`internal/`, không nằm trong `internal/platform/`. Nếu thêm cross-cutting concern mới, đặt cạnh các
package hiện có, đừng tạo `internal/platform/xxx` mới.

## 3. Luồng dependency

```
Router (gin) → Delivery Handler → Usecase → Port Interface → Repository Impl → *sql.DB
```

## 4. Provider — nơi wiring cụ thể được phép

Lazy init bằng check `nil` (không `sync.Once` — provider được resolve tuần tự lúc `app.New()` khởi
động, trước khi nhận request, không có race condition thực tế). Xem `internal/module/user/provider.go`.

## 5. Cross-module dependency — ví dụ thật: `auth` cần `user`

`auth` không import `user/port`. Nó tự định nghĩa `UserReader` (interface nhỏ, đủ dùng) trong
`auth/port/user_reader.go`, rồi `auth/provider.go` bind bằng `userrepo.NewUserRepository(p.db)` —
import trực tiếp package `user/repository` (không phải `user/port`). Đây là ngoại lệ được phép: chỉ
provider của module A được import package `repository` của module B; usecase của A chỉ biết interface
riêng của A.

## 6. Transaction — cơ bản, không có TxManager

`AuthUsecase` (xem `auth/usecase/auth_usecase.go`) nhận thẳng `*sql.DB` qua constructor. Usecase nào
cần ghi nhiều bảng (`Register`, `Refresh`) tự gọi `db.BeginTx(ctx, nil)`, `defer tx.Rollback()`, gọi
các method `*WithTx` của repository, rồi `tx.Commit()`. Repository expose 2 method song song cho mỗi
thao tác cần transaction: bản dùng `*sql.DB` bình thường và bản nhận thêm `tx *sql.Tx`. Không có
`TxManager`/`UnitOfWork`. Copy pattern y hệt `auth/usecase/register.go` khi cần transaction nhiều bảng
cho module nghiệp vụ mới — đừng tự chế abstraction mới.

## 7. Error handling

```
Repository → lỗi kỹ thuật gốc (sql.ErrNoRows, ...)
  → Usecase: errors.Is(...) rồi convert sang apperror.XXX(message, err)
    → Delivery: chỉ ctx.Error(err), không tự đoán mã lỗi
      → middleware.ErrorHandlerMiddleware(): map AppError → JSON {success, code, message, request_id}
```

Thêm loại lỗi mới → thêm factory function trong `internal/apperror/app_error.go`.

## 8. Auth / middleware

- 2 router group cùng prefix `/api`: 1 không có `AuthMiddleware` (register/login/refresh), 1 có
  (mọi thứ còn lại, kể cả logout).
- `app.go` tự new `authrepo.NewAuthRepository(db)` để làm `TokenBlacklistChecker` cho
  `AuthMiddleware` — ngoại lệ chấp nhận được (hạ tầng bootstrap), đừng lấy làm cớ để `app.go` new thêm
  repo khác.
- Phân quyền role dùng `middleware.RequireRoles("Admin")` gắn trực tiếp trên route (xem
  `user/routes.go`), không có trong middleware chung hay usecase.
- Access token JWT 15 phút, refresh token 7 ngày trong HttpOnly cookie (`refresh_token`,
  path `/api/auth`). Access token trả trong JSON, FE tự giữ (không cookie).

## 9. Entity / DTO / mapping

- `entity/*.go`: struct thuần, không tag json.
- `delivery/dto.go`: Request (`binding:"..."`) + Response (`json:"..."`), map tay trong handler.
- Usecase Input/Output struct nằm ngay trong file action (`RegisterInput` trong `register.go`).

## 10. Checklist: thêm module mới

1. `entity/xxx.go`
2. `port/repository.go` — chỉ method usecase cần
3. `repository/xxx_repo.go` — `*sql.DB`, `var _ port.XxxRepository = (*XxxRepository)(nil)`
4. `usecase/xxx_usecase.go` + 1 file/action, Input/Output struct trong file action
5. `delivery/dto.go` + `delivery/handler.go`
6. `provider.go` — lazy-init nil-check
7. `module.go` — struct + `NewXxxModule(db *sql.DB, ...)`
8. `routes.go` — `RegisterRoutes` + `RegisterPublicRoutes`/`RegisterProtectedRoutes`
9. Đăng ký trong `internal/app/app.go`
10. Cần module khác → interface riêng trong `port/` của mình (mục 5)
11. Cần transaction nhiều bảng → pattern mục 6
12. `go build ./...`

## 11. Không nên làm

- Handler gọi thẳng `*sql.DB`/`repository`.
- Usecase import `gin` hoặc nhận `*gin.Context`.
- Usecase import module khác qua `port`/`repository` trực tiếp (chỉ provider được).
- Tự ý implement `TxManager`/`UnitOfWork`/`sync.Once` "cho chuẩn" — chưa có trong code, đừng thêm rồi
  làm 2 module dùng 2 kiểu transaction khác nhau.
- Thêm `internal/platform/xxx` khi cross-cutting concern tương tự đã có sẵn ở `internal/xxx`.
