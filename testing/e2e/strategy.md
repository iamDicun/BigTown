# E2E Testing Strategy — BigTown

Tài liệu định hướng và danh sách kịch bản kiểm thử đầu-cuối (**End-to-End Testing**) với Playwright + TypeScript.

---

## 🎯 Mục tiêu
Kiểm thử trải nghiệm thực tế của người dùng trên trình duyệt (Chromium) tương tác với Vue 3 UI, Phaser Game, Go Backend API và PostgreSQL Database.

## 📊 Kịch bản E2E Cốt lõi (Critical User Journeys)

| Mã test | Tên Luồng | Mô tả Luồng Người Dùng |
|---------|-----------|------------------------|
| **E2E-01** | Auth Flow | Mở ứng dụng ➔ Chuyển qua tab Đăng ký ➔ Đăng ký tài khoản mới ➔ Đăng nhập thành công ➔ Verify lưu phiên đăng nhập |
| **E2E-02** | Character Flow | Đăng nhập ➔ Tạo nhân vật mới ➔ Chọn nhân vật ➔ Chuyển tới thế giới Game |
| **E2E-03** | Editor Flow | Đăng nhập & Vào Game ➔ Mở menu Editor ➔ Đặt vật phẩm trang trí ➔ Trừ coin ➔ Xóa vật phẩm ➔ Hoàn coin |
| **E2E-04** | Chat Flow | Gửi tin nhắn chat từ UI ➔ Kiểm tra tin nhắn hiển thị thành công trong bảng tin nhắn chat |

---

## 🛠️ Kiến trúc Thư mục
Tuân thủ **Skill automation-test-generator**:
- **Data-Driven JSON:** `testing/e2e/src/test-data/`
- **Page Object Models:** `testing/e2e/src/pages/`
- **Playwright Specs:** `testing/e2e/src/specs/`
