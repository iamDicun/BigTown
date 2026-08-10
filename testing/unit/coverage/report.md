# Unit Test Checklist — Backend

> Cập nhật mỗi khi thêm test case mới. Tick `[x]` khi đã viết và pass.

---

## 1. Security

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Generate token → parse lại đúng userID, role | `TestGenerateAndParseToken` | [x] |
| 2 | Token hết hạn → lỗi `ErrTokenExpired` | `TestParseToken_Expired` | [x] |
| 3 | Token với sai secret → lỗi | `TestParseToken_WrongSecret` | [x] |
| 4 | Chuỗi không phải JWT → lỗi | `TestParseToken_InvalidString` | [x] |
| 5 | Hash password → verify đúng password trả true | `TestHashAndCheckPassword` | [x] |
| 6 | Hash password → verify sai password trả false | `TestCheckPassword_WrongPassword` | [x] |
| 7 | Hash empty password không lỗi | `TestHashPassword_Empty` | [x] |
| 8 | `GenerateRandomToken` trả token không rỗng | — | [ ] |
| 9 | `HashToken` cho output nhất quán với cùng input | — | [ ] |

---

## 2. Auth — Usecase

### Login

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Email + password đúng → trả access token + refresh token | `TestLogin_Success` | [x] |
| 2 | Email không tồn tại → lỗi unauthorized | `TestLogin_UserNotFound` | [x] |
| 3 | Đúng email, sai password → lỗi unauthorized | `TestLogin_WrongPassword` | [x] |
| 4 | User có email nhưng chưa có credential → lỗi | `TestLogin_NoCredential` | [x] |
| 5 | Email có dấu cách, viết hoa → tự trim + lowercase | — | [ ] |
| 6 | Lỗi DB khi gọi `FindByEmail` → lỗi internal | — | [ ] |

### Register

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Email mới, password hợp lệ → tạo user + credential | `TestRegister_Success` | [x] |
| 2 | Email đã tồn tại → lỗi duplicate | `TestRegister_DuplicateEmail` | [x] |
| 3 | Password quá ngắn → lỗi validation | — | [ ] |
| 4 | Lỗi DB khi BeginTx → lỗi internal | — | [ ] |
| 5 | Lỗi DB khi Commit → rollback | — | [ ] |

### Refresh Token

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Refresh token hợp lệ → trả access token mới + refresh token mới, token cũ bị revoke | `TestRefresh_Success` | [x] |
| 2 | Refresh token không tồn tại → lỗi invalid | `TestRefresh_TokenNotFound` | [x] |
| 3 | Refresh token đã bị revoke → lỗi revoked | `TestRefresh_TokenRevoked` | [x] |
| 4 | Refresh token hết hạn → lỗi expired | — | [ ] |
| 5 | User đã bị xóa sau khi refresh token được tạo → lỗi | — | [ ] |

### Logout

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Có access token + refresh token hợp lệ → blacklist + revoke | `TestLogout_Success` | [x] |
| 2 | Thiếu access token → lỗi | `TestLogout_MissingAccessToken` | [x] |
| 3 | Refresh token đã hết hạn → lỗi expired | — | [ ] |
| 4 | Refresh token đã bị revoke → lỗi revoked | — | [ ] |

### Teams SSO

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | SSO token hợp lệ, user đã từng đăng nhập → trả token | [ ] |
| 2 | SSO token hợp lệ, user mới → tạo user + identity + trả token | [ ] |
| 3 | SSO token rỗng → lỗi bad request | [ ] |
| 4 | Teams token verifier trả lỗi → lỗi unauthorized | [ ] |
| 5 | Claims thiếu email → lỗi bad request | [ ] |

---

## 3. Character — Usecase

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | User chưa có character → tạo character thành công | [ ] |
| 2 | User đã có character → lỗi conflict | [ ] |
| 3 | `GetMyCharacter` có character → trả entity đúng | [ ] |
| 4 | `GetMyCharacter` chưa có character → lỗi not found | [ ] |
| 5 | Tạo character với tên rỗng → lỗi validation | [ ] |
| 6 | Lỗi DB khi insert → lỗi internal | [ ] |

---

## 4. Chat — Usecase

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | Message hợp lệ → lưu DB + publish | [ ] |
| 2 | Message rỗng → lỗi validation | [ ] |
| 3 | Message > 500 ký tự → lỗi validation | [ ] |
| 4 | Room có messages → trả danh sách đúng thứ tự thời gian | [ ] |
| 5 | Room không có messages → trả mảng rỗng | [ ] |
| 6 | Lỗi publish Centrifuge → vẫn lưu DB thành công | [ ] |

---

## 5. Editor — Usecase

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | Đủ coin, vị trí trống → trừ coin, insert placement | [ ] |
| 2 | Không đủ coin → lỗi insufficient coins | [ ] |
| 3 | Vị trí đã có item → lỗi conflict | [ ] |
| 4 | x/y ngoài biên map → lỗi validation | [ ] |
| 5 | Chủ sở hữu xóa item của mình → thành công | [ ] |
| 6 | Không phải chủ sở hữu xóa → lỗi forbidden | [ ] |

---

## 6. Realtime — Usecase + Room Actor

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | Movement trong biên map → accept + broadcast | [ ] |
| 2 | Movement ngoài biên → reject | [ ] |
| 3 | Đứng quá gần người khác (<24px) → reject | [ ] |
| 4 | Khoảng cách > max speed trong 1 tick → reject (anti-cheat) | [ ] |
| 5 | Warp trong cùng map → broadcast | [ ] |
| 6 | Warp sang map khác → leave room cũ, join room mới | [ ] |
| 7 | Join room có map tồn tại → subscribe + snapshot | [ ] |
| 8 | Join room map không tồn tại → lỗi not found | [ ] |
| 9 | Leave room → unsubscribe + persist vị trí | [ ] |
| 10 | Room actor: 1 player join phòng trống → state đúng | [ ] |
| 11 | Room actor: 5 players join → 5 client, phân biệt clientID | [ ] |
| 12 | Room actor: tick broadcast gộp N vị trí → 1 message | [ ] |
| 13 | Room actor: player leave → xóa khỏi state, broadcast `player_left` | [ ] |
| 14 | Room actor: shutdown → flush pending writes, đóng channel | [ ] |

---

## 7. Leaderboard — Usecase

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | Có dữ liệu → trả danh sách theo score DESC | [ ] |
| 2 | Không có dữ liệu → trả mảng rỗng | [ ] |
| 3 | Limit kết quả đúng (top N) | [ ] |

---

## 8. Frontend — Vitest

| # | Test case | Trạng thái |
|---|-----------|:----------:|
| 1 | Auth store: login success → state có token | [ ] |
| 2 | Auth store: login fail → state không đổi | [ ] |
| 3 | Auth store: logout → clear token | [ ] |
| 4 | Game store: add player → danh sách tăng | [ ] |
| 5 | Game store: remove player → player biến mất | [ ] |
| 6 | GameSocket: `getDefaultRealtimeUrl()` với absolute URL | [ ] |
| 7 | GameSocket: `getDefaultRealtimeUrl()` với relative URL | [ ] |

---

## Tổng kết

| Module | Tổng test case | Đã viết | Còn thiếu |
|--------|:---:|:---:|:---:|
| Security | 9 | 7 | 2 |
| Auth usecase | 21 | 10 | 11 |
| Character usecase | 6 | 0 | 6 |
| Chat usecase | 6 | 0 | 6 |
| Editor usecase | 6 | 0 | 6 |
| Realtime usecase + actor | 14 | 0 | 14 |
| Leaderboard usecase | 3 | 0 | 3 |
| Frontend (Vitest) | 7 | 0 | 7 |
| **Tổng** | **72** | **17** | **55** |
