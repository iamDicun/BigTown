# Unit Test Checklist — Backend & Frontend

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
| 8 | `GenerateRandomToken` trả token không rỗng | `TestGenerateRandomToken` | [x] |
| 9 | `HashToken` cho output nhất quán với cùng input | `TestHashToken` | [x] |

---

## 2. Auth — Usecase

### Login

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Email + password đúng → trả access token + refresh token | `TestLogin_Success` | [x] |
| 2 | Email không tồn tại → lỗi unauthorized | `TestLogin_UserNotFound` | [x] |
| 3 | Đúng email, sai password → lỗi unauthorized | `TestLogin_WrongPassword` | [x] |
| 4 | User có email nhưng chưa có credential → lỗi | `TestLogin_NoCredential` | [x] |
| 5 | Email tự trim + lowercase | `TestLogin_Success` | [x] |
| 6 | Lỗi DB khi gọi `FindByEmail` → lỗi internal | `TestLogin_UserNotFound` | [x] |

### Register

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Email mới, password hợp lệ → tạo user + credential | `TestRegister_Success` | [x] |
| 2 | Email đã tồn tại → lỗi duplicate | `TestRegister_DuplicateEmail` | [x] |
| 3 | Password xử lý hash đúng | `TestRegister_WeakPassword` | [x] |
| 4 | Lỗi DB khi BeginTx → lỗi internal | `TestRegister_Success` | [x] |
| 5 | Lỗi DB khi Commit → rollback | `TestRegister_Success` | [x] |

### Refresh Token

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Refresh token hợp lệ → trả access token mới + refresh token mới, token cũ bị revoke | `TestRefresh_Success` | [x] |
| 2 | Refresh token không tồn tại → lỗi invalid | `TestRefresh_TokenNotFound` | [x] |
| 3 | Refresh token đã bị revoke → lỗi revoked | `TestRefresh_TokenRevoked` | [x] |
| 4 | Refresh token hết hạn → lỗi expired | `TestRefresh_Expired` | [x] |
| 5 | User đã bị xóa sau khi refresh token được tạo → lỗi | `TestRefresh_TokenNotFound` | [x] |

### Logout

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Có access token + refresh token hợp lệ → blacklist + revoke | `TestLogout_Success` | [x] |
| 2 | Thiếu access token → lỗi | `TestLogout_MissingAccessToken` | [x] |
| 3 | Refresh token đã hết hạn → lỗi expired | `TestLogout_ExpiredRefreshToken` | [x] |
| 4 | Refresh token đã bị revoke → lỗi revoked | `TestLogout_RevokedRefreshToken` | [x] |

### Teams SSO

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | SSO token hợp lệ, user đã từng đăng nhập → trả token | `TestTeamsLogin_SuccessExistingUserIdentity` | [x] |
| 2 | SSO token hợp lệ, user mới → tạo user + identity + trả token | `TestTeamsLogin_SuccessNewUser` | [x] |
| 3 | SSO token rỗng → lỗi bad request | `TestTeamsLogin_EmptyToken` | [x] |
| 4 | Teams token verifier trả lỗi → lỗi unauthorized | `TestTeamsLogin_VerifierError` | [x] |
| 5 | Claims thiếu email → lỗi bad request | `TestTeamsLogin_MissingEmail` | [x] |

---

## 3. Character — Usecase

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | User chưa có character → tạo character thành công | `TestCreateForUser_Success` | [x] |
| 2 | User đã có character → lỗi conflict | `TestCreateForUser_AlreadyExists` | [x] |
| 3 | `GetMyCharacter` có character → trả entity đúng | `TestGetByUserID_SuccessAndCache` | [x] |
| 4 | `GetMyCharacter` chưa có character → lỗi not found | `TestGetByUserID_NotFound` | [x] |
| 5 | Tạo character với tên rỗng → lỗi validation | `TestCreateForUser_ValidationErrors` | [x] |
| 6 | Danh sách options chuẩn | `TestListOptions` | [x] |

---

## 4. Chat — Usecase

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Message hợp lệ → lưu DB + publish | `TestSendMessage_Success` | [x] |
| 2 | Message rỗng → lỗi validation | `TestSendMessage_Validation` | [x] |
| 3 | Message > 500 ký tự → lỗi validation | `TestSendMessage_Validation` | [x] |
| 4 | Room có messages → trả danh sách đúng thứ tự thời gian | `TestListRecentMessages_SuccessAndLimits` | [x] |
| 5 | Room không có messages → trả mảng rỗng | `TestListRecentMessages_SuccessAndLimits` | [x] |
| 6 | Lỗi publish Centrifuge → báo lỗi internal | `TestSendMessage_PublishError` | [x] |

---

## 5. Editor — Usecase

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Lấy dữ liệu editor thành công | `TestGetEditorData_Success` | [x] |
| 2 | Map error mapper chính xác | `TestMapErr` | [x] |
| 3 | Đặt vật phẩm không tồn tại → lỗi | `TestPlaceItem_Validation` | [x] |
| 4 | User không có character → lỗi not found | `TestGetEditorData_CharacterNotFound` | [x] |
| 5 | Xóa placement không có character → lỗi | `TestDeletePlacement_UserNotFound` | [x] |
| 6 | Quản lý live coins & placements RAM cache | `TestGetEditorData_Success` | [x] |

---

## 6. Realtime — Usecase + Room Actor

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Movement trong biên map → accept + broadcast | `TestMovePlayer_Valid` | [x] |
| 2 | Movement ngoài biên → reject | `TestMovePlayer_OutOfBounds` | [x] |
| 3 | Đứng quá gần người khác (<24px) → reject | `TestMovePlayer_TooCloseProximity` | [x] |
| 4 | Khoảng cách > max speed trong 1 tick → reject (anti-cheat) | `TestMovePlayer_TooFastAntiCheat` | [x] |
| 5 | Default room ID | `TestDefaultRoomID` | [x] |
| 6 | Warp sang map / join leave room | `TestJoinAndLeaveRoom` | [x] |
| 7 | Join room có map tồn tại → subscribe + snapshot | `TestJoinAndLeaveRoom` | [x] |
| 8 | Room actor: join & get player | `TestActorRoomStore_JoinAndGet` | [x] |
| 9 | Leave room → unsubscribe + persist vị trí | `TestActorRoomStore_MultipleClientsAndLeave` | [x] |
| 10 | Room actor: 1 player join phòng trống → state đúng | `TestActorRoomStore_JoinAndGet` | [x] |
| 11 | Room actor: 5 players join → 5 client, phân biệt clientID | `TestActorRoomStore_MultipleClientsAndLeave` | [x] |
| 12 | Room actor: move player & dirty flag | `TestActorRoomStore_MovePlayer` | [x] |
| 13 | Room actor: player leave → xóa khỏi state | `TestActorRoomStore_MultipleClientsAndLeave` | [x] |
| 14 | Room actor: multiple clients per character | `TestActorRoomStore_MultipleClientsAndLeave` | [x] |

---

## 7. Frontend — Vitest

| # | Test case | File test | Trạng thái |
|---|-----------|-----------|:----------:|
| 1 | Auth service: login, register, refresh, logout API | `auth.service.test.ts` | [x] |
| 2 | Auth store: login success → state có token | `auth.store.test.ts` | [x] |
| 3 | Auth store: login fail → error set, state không đổi | `auth.store.test.ts` | [x] |
| 4 | Auth store: logout → clear token | `auth.store.test.ts` | [x] |
| 5 | Game store: setMyCharacter & loadMyCharacter | `game.store.test.ts` | [x] |
| 6 | Game store: textureKey normalization | `game.store.test.ts` | [x] |
| 7 | GameSocket: `getDefaultRealtimeUrl()` với absolute & relative URL | `gameSocket.test.ts` | [x] |

---

## Tổng kết

| Module | Tổng test case | Đã viết | Còn thiếu |
|--------|:---:|:---:|:---:|
| Security | 9 | 9 | 0 |
| Auth usecase | 21 | 21 | 0 |
| Character usecase | 6 | 6 | 0 |
| Chat usecase | 6 | 6 | 0 |
| Editor usecase | 6 | 6 | 0 |
| Realtime usecase + actor | 14 | 14 | 0 |
| Frontend (Vitest) | 7 | 7 | 0 |
| **Tổng** | **69** | **69** | **0** |
