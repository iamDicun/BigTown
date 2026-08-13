# Unit Testing Strategy — BigTown (Backend + Frontend)

## Tổng quan module cần test

### Phase 2a — Backend (Go)

| # | Module | Layer cần test | Số file test dự kiến | Mức độ ưu tiên |
|---|--------|---------------|---------------------|:---:|
| 1 | `security/` | JWT + password (pure) | 2 | Cao |
| 2 | `module/auth/` | Usecase + Repository | 2 | Cao |
| 3 | `module/character/` | Usecase + Repository | 2 | Cao |
| 4 | `module/chat/` | Usecase + Repository | 2 | Trung bình |
| 5 | `module/editor/` | Usecase + Repository | 2 | Trung bình |
| 6 | `module/realtime/` | Usecase + Room Actor | 2 | Cao |

### Phase 2b — Frontend (Vitest)

| # | Module | Layer cần test | Số file test dự kiến |
|---|--------|---------------|---------------------|
| 1 | `features/auth/` | Store + Service | 2 |
| 2 | `features/game/` | Store + GameSocket | 2 |

---

## Nguyên tắc

- Test từng hàm/module riêng lẻ, không cần database thật, không network
- Mock dependency qua interface (Port)
- Chạy nhanh (vài ms/test), chạy được ngay trên CI không cần Docker
- File test đặt cạnh source code: `xxx_test.go` cùng package

## Những layer nào test

Kiến trúc: `Router → Handler → Usecase → Port (interface) → Repository → PostgreSQL`

| Layer | Test? | Lý do |
|-------|:-----:|-------|
| Usecase | ✅ | Toàn bộ business logic nằm ở đây |
| Repository | ✅ | Query SQL, mapping DB row → entity |
| Domain Entity | ✅ | Validation, business rule thuần túy |
| Security (JWT, bcrypt) | ✅ | Pure function, không dependency ngoài |
| Handler/Delivery | ❌ | Chỉ wire HTTP request → usecase |
| Routes | ❌ | Chỉ đăng ký endpoint |
| Provider/Module | ❌ | Chỉ wire dependency |

---

## 1. Usecase Test — mock Repository

**Cần test:** business logic của từng module, không hit database thật.

**Mock pattern:**

```go
// Mock implement interface Port
type mockUserRepo struct {
    FindByEmailFn func(email string) (*entity.User, error)
    CreateFn      func(user *entity.User) error
}

func (m *mockUserRepo) FindByEmail(email string) (*entity.User, error) {
    return m.FindByEmailFn(email)
}

func (m *mockUserRepo) Create(user *entity.User) error {
    return m.CreateFn(user)
}
```

**Các test case cần viết:**

### Auth

| Test | Input | Expected |
|------|-------|----------|
| Login_Success | email + password đúng | Trả access token + refresh token |
| Login_WrongPassword | email đúng, password sai | Error unauthorized |
| Login_UserNotFound | email không tồn tại | Error not found |
| Register_Success | email mới, password hợp lệ | Tạo user + credential thành công |
| Register_DuplicateEmail | email đã tồn tại | Error conflict |
| Register_WeakPassword | password < 6 ký tự | Error validation |
| RefreshToken_Valid | refresh token còn hạn | Trả access token mới |
| RefreshToken_Expired | refresh token hết hạn | Error expired |
| RefreshToken_Revoked | refresh token đã bị revoke | Error unauthorized |
| Logout | access token hợp lệ | Token bị blacklist |

### Character

| Test | Input | Expected |
|------|-------|----------|
| CreateCharacter_Success | user chưa có character | Tạo character thành công |
| CreateCharacter_AlreadyExists | user đã có character | Error conflict |
| GetMyCharacter_HasCharacter | user đã tạo character | Trả character entity |
| GetMyCharacter_NotFound | user chưa tạo character | Error not found |

### Chat

| Test | Input | Expected |
|------|-------|----------|
| SendMessage_Success | message hợp lệ, user trong room | Lưu DB + publish Centrifuge |
| SendMessage_Empty | message rỗng | Error validation |
| SendMessage_TooLong | message > 500 ký tự | Error validation |
| GetMessages_Success | room có messages | Trả danh sách message theo thời gian |
| GetMessages_Empty | room chưa có message | Trả mảng rỗng |

### Editor (Placement)

| Test | Input | Expected |
|------|-------|----------|
| PlaceItem_Success | đủ coin, vị trí trống | Trừ coin, insert placement |
| PlaceItem_InsufficientCoins | không đủ coin | Error insufficient coins |
| PlaceItem_Occupied | vị trí đã có item | Error conflict |
| PlaceItem_OutOfBounds | x/y ngoài biên map | Error validation |
| RemoveItem_Success | item tồn tại, là chủ sở hữu | Xóa placement |
| RemoveItem_NotOwner | không phải chủ sở hữu | Error forbidden |

### Realtime

| Test | Input | Expected |
|------|-------|----------|
| PlayerMove_Valid | tọa độ trong biên map | Cập nhật vị trí, broadcast |
| PlayerMove_OutOfBounds | x/y ngoài biên | Reject movement |
| PlayerMove_TooClose | đứng quá gần người khác (< 24px) | Reject movement |
| PlayerMove_TooFast | khoảng cách > max speed trong 1 tick | Reject movement (anti-cheat) |
| PlayerWarp_SameMap | warp trong cùng map | Cập nhật tọa độ, broadcast |
| PlayerWarp_NewMap | warp sang map khác | Unsubscribe room cũ, join room mới |
| JoinRoom_Success | map tồn tại | Subscribe + trả snapshot room |
| JoinRoom_MapNotFound | map không tồn tại | Error not found |
| LeaveRoom | đang trong room | Unsubscribe, persist vị trí cuối |

### Room Actor (state machine)

| Test | Input | Expected |
|------|-------|----------|
| Room_JoinFirstPlayer | phòng trống, 1 player vào | Room có 1 player, state đúng |
| Room_MultiplePlayers | 5 players join | Room có 5 players, phân biệt clientID |
| Room_TickBroadcast | 3 players di chuyển trong 1 tick | Gửi 1 broadcast gộp 3 vị trí |
| Room_PlayerLeave | player rời phòng | Xóa khỏi room state, broadcast player_left |
| Room_GracefulShutdown | shutdown actor | Flush pending DB writes, đóng channel |

---

## 2. Repository Test

**Cần test:** query SQL đúng, mapping result → entity đúng, xử lý edge case.

Có thể dùng 2 cách:
- **Integration-style:** Docker Compose test DB, insert/query thật (chậm hơn nhưng chắc chắn)
- **Mock driver:** mock `sql.DB` hoặc dùng `sqlmock` library (nhanh, không cần Docker)

### Test case chung cho mọi repository

| Test | Expected |
|------|----------|
| Insert thành công | Không lỗi, row tồn tại trong DB |
| Insert duplicate key | Lỗi unique constraint |
| FindByX có kết quả | Trả entity đúng |
| FindByX không có kết quả | `sql.ErrNoRows` |
| Update thành công | Row bị thay đổi |
| Delete thành công | Row không còn tồn tại |
| Transaction rollback khi lỗi | Không có gì bị thay đổi |

---

## 3. Entity & Value Object Test — pure unit test

Không cần mock gì cả. Test data validation + business rule.

### Security

| Function | Test |
|----------|------|
| `jwt.GenerateToken(userID, secret, ttl)` | Tạo token → parse lại → đúng claims, đúng userID |
| `jwt.ParseToken(token, secret)` | Token hết hạn → lỗi, token sai secret → lỗi |
| `password.Hash(plain)` | Hash thành công, không phải plain text |
| `password.Verify(hash, plain)` | Đúng password → true, sai → false |

### Entity validation

| Entity | Test |
|--------|------|
| Email | Format sai → invalid, thiếu @ → invalid |
| Coordinate | x < 0 → invalid, x > map.width → invalid |
| Coin | Không được âm, không overflow khi cộng |
| Message | Rỗng → invalid, > max length → invalid |

---

## 4. Frontend Unit Test (Vitest)

### Store (Pinia)

| Store | Test |
|-------|------|
| Auth store | Login success → state có token, Login fail → state không đổi |
| Auth store | Refresh token → gọi API refresh, update access token |
| Game store | Add player → danh sách players tăng |
| Game store | Remove player → player biến mất khỏi danh sách |

### Service

| Service | Test |
|---------|------|
| Auth service | Login gọi đúng POST /api/auth/login với body |
| Auth service | Register lỗi 409 → throw ApiError |
| Game socket | `getDefaultRealtimeUrl()` → URL đúng với VITE_API_BASE_URL |
| Game socket | `parseMoveEvent(data)` → entity đúng format |

---

## Cấu trúc file test (cạnh source)

```text
backend/internal/
├── security/
│   ├── jwt.go
│   ├── jwt_test.go
│   ├── password.go
│   └── password_test.go
├── module/auth/
│   ├── entity/user.go
│   ├── entity/user_test.go
│   ├── usecase/auth_usecase.go
│   ├── usecase/auth_usecase_test.go
│   ├── repository/auth_repo.go
│   └── repository/auth_repo_test.go
├── module/chat/
│   ├── usecase/chat_usecase.go
│   ├── usecase/chat_usecase_test.go
│   ├── repository/chat_repo.go
│   └── repository/chat_repo_test.go
├── module/editor/
│   ├── usecase/editor_usecase.go
│   ├── usecase/editor_usecase_test.go
│   ├── repository/editor_repo.go
│   └── repository/editor_repo_test.go
├── module/character/
│   ├── usecase/character_usecase.go
│   ├── usecase/character_usecase_test.go
│   ├── repository/character_repo.go
│   └── repository/character_repo_test.go
├── module/realtime/
│   ├── usecase/realtime_usecase.go
│   ├── usecase/realtime_usecase_test.go
│   ├── room/actor_room_store.go
│   └── room/actor_room_store_test.go
└── module/leaderboard/
    ├── usecase/leaderboard_usecase.go
    └── usecase/leaderboard_usecase_test.go

frontend/src/features/
├── auth/
│   ├── stores/authStore.ts
│   ├── stores/authStore.test.ts
│   ├── services/auth.service.ts
│   └── services/auth.service.test.ts
└── game/
    ├── stores/gameStore.ts
    ├── stores/gameStore.test.ts
    ├── network/gameSocket.ts
    └── network/gameSocket.test.ts
```

## Chạy test

```bash
# Backend — tất cả unit test
cd backend && go test ./...

# Backend — test 1 module cụ thể
cd backend && go test ./internal/module/auth/...

# Backend — verbose
cd backend && go test -v ./...

# Frontend
cd frontend && npx vitest --run

# CI step (thêm vào ci-cd.yml)
      - name: go test (backend)
        run: go test ./...
        working-directory: backend

      - name: vitest (frontend)
        run: npx vitest --run
        working-directory: frontend
```
