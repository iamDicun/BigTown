# Integration Testing Strategy — BigTown

## 1. Tổng quan & Mục tiêu

Khác với **Unit Testing** (chỉ test business logic đơn lập và mock dependency), **Integration Testing** nhằm kiểm thử sự tương tác thực tế giữa:

```text
HTTP Handlers / Routes ➔ Usecase Business Logic ➔ Repository ➔ PostgreSQL Database (thật)
```

Mục tiêu chính:
- Đảm bảo SQL Query (CRUD, Transactions, JOIN, Indexes) chạy đúng với PostgreSQL thật.
- Đảm bảo luồng mã hóa bảo mật (bcrypt, JWT, Bearer token) hoạt động tốt khi ghép nối các layer HTTP + DB.
- Đảm bảo dữ liệu được dọn dẹp sạch sẽ (isolation) giữa các test cases.

---

## 2. Danh sách Kịch bản Kiểm thử Tích hợp (Integration Test Cases)

| Kịch bản | Tên Flow | Mô tả chi tiết | Các bảng DB tác động |
|:---:|---|---|---|
| **INT-01** | **Auth Flow** | Dang ky user moi ➔ Dang nhap lay JWT ➔ Truye cap protected endpoint ➔ Dang xuat revoke token | `users`, `credentials`, `refresh_tokens`, `access_token_blacklist` |
| **INT-02** | **Character Flow** | Dang ky + Dang nhap ➔ Lay danh sach options ➔ Tao nhan vat moi ➔ Lay thong tin nhan vat cua toi | `users`, `characters`, `maps` |
| **INT-03** | **Chat Flow** | Tao 2 nhan vat ➔ Gui tin nhan chat qua HTTP ➔ Chờ worker ghi DB ➔ Truy van lich su chat theo room_id | `users`, `characters`, `chat_messages` |
| **INT-04** | **Editor Flow** | Lay du lieu editor ➔ Dat vat pham trang tri (tru coin) ➔ Xoa vat pham (hoan coin) ➔ Verify DB | `characters`, `maps`, `decoration_items`, `map_placements` |
| **INT-05** | **Realtime Bootstrap Flow** | Fetch metadata bootstrap room/map theo map code ➔ Verify spawn points & map info | `maps`, `map_npc_spawns` |

---

## 3. Hạ tầng & Môi trường Kiểm thử

### Docker Compose Test Container
Sử dụng container PostgreSQL độc lập để chạy test:
- **Host:** `localhost`
- **Port:** `5434` (để tránh đụng độ với DB dev 5433 hoặc DB prod)
- **Database:** `bigtown_test`
- **User / Password:** `test_user` / `test_pass`

### Teardown & Isolation Strategy
Trước mỗi test function hoặc test suite, hệ thống sẽ thực thi hàm `TruncateTables(db)` để làm sạch dữ liệu trong tất cả các bảng:
```sql
TRUNCATE users, credentials, refresh_tokens, user_identities, characters, chat_messages, map_placements RESTART IDENTITY CASCADE;
```

---

## 4. Hướng dẫn chạy Integration Test

```bash
# 1. Khởi chạy Postgres Test Container
docker compose -f testing/integration/docker-compose.test.yml up -d

# 2. Chạy toàn bộ suite Integration Test
cd backend && go test -v ./testing/integration/...

# 3. Dừng container sau khi test xong
docker compose -f testing/integration/docker-compose.test.yml down
```
