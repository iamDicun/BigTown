# Integration Test Checklist & Report — BigTown

> Checklist kiểm thử tích hợp (Integration Test) tương tác với PostgreSQL Database thật.

---

## 1. Auth Flow Integration Test (`INT-01`)

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Register user mới → verify row trong bảng `app_user` & `credential` | `TestAuthIntegration_RegisterAndLogin` | [x] |
| 2 | Login sai password → trả lỗi Unauthorized (401) | `TestAuthIntegration_LoginWrongPassword` | [x] |
| 3 | Access token hợp lệ → gọi protected route `/api/characters/me` | `TestAuthIntegration_ProtectedEndpoint` | [x] |
| 4 | Logout → blacklist access token & revoke refresh token trong DB | `TestAuthIntegration_Logout` | [x] |

---

## 2. Character Flow Integration Test (`INT-02`)

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Lấy danh sách options nhân vật (`/api/characters/options`) | `TestCharacterIntegration_ListOptions` | [x] |
| 2 | Tạo nhân vật mới → verify row trong bảng `characters` DB | `TestCharacterIntegration_CreateAndGetMe` | [x] |
| 3 | Lấy thông tin nhân vật của tôi (`/api/characters/me`) | `TestCharacterIntegration_CreateAndGetMe` | [x] |
| 4 | Tạo nhân vật trùng lặp → lỗi conflict | `TestCharacterIntegration_DuplicateCharacter` | [x] |

---

## 3. Chat Flow Integration Test (`INT-03`)

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Gửi tin nhắn chat HTTP → async worker ghi thành công vào bảng `chat_messages` | `TestChatIntegration_SendMessageAndList` | [x] |
| 2 | Truy vấn danh sách tin nhắn theo room_id (`/api/rooms/:roomId/chat/messages`) | `TestChatIntegration_SendMessageAndList` | [x] |

---

## 4. Editor Flow Integration Test (`INT-04`)

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Lấy dữ liệu editor (`/api/editor?map_code=...`) | `TestEditorIntegration_GetEditorData` | [x] |
| 2 | Đặt vật phẩm trang trí → trừ coin và lưu placement vào RAM/DB | `TestEditorIntegration_PlaceAndDelete` | [x] |
| 3 | Xóa vật phẩm trang trí → hoàn coin và xóa placement | `TestEditorIntegration_PlaceAndDelete` | [x] |

---

## 5. Realtime Bootstrap Integration Test (`INT-05`)

| # | Test case | Hàm test | Trạng thái |
|---|-----------|----------|:----------:|
| 1 | Lấy metadata default map & spawn points (`/api/realtime/bootstrap`) | `TestRealtimeIntegration_Bootstrap` | [x] |

---

## Tổng kết

| Flow | Tổng test cases | Đã pass | Còn thiếu |
|------|:---:|:---:|:---:|
| INT-01 Auth Flow | 4 | 4 | 0 |
| INT-02 Character Flow | 4 | 4 | 0 |
| INT-03 Chat Flow | 2 | 2 | 0 |
| INT-04 Editor Flow | 3 | 3 | 0 |
| INT-05 Realtime Bootstrap | 1 | 1 | 0 |
| **Tổng** | **14** | **14** | **0** |
