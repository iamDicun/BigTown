-- Schema tối thiểu cho boilerplate: chỉ đủ cho auth (JWT + refresh + blacklist) + user profile.
-- Đổi/thêm bảng nghiệp vụ thật của bạn bên cạnh app_user, đừng sửa 4 bảng này trừ khi bạn cũng sửa
-- theo internal/module/auth và internal/module/user cho khớp.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE app_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(150) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'User'
);

-- Tách khỏi app_user để app_user chỉ giữ thông tin hồ sơ, giống pattern employee/authen của
-- project gốc (xem ARCHITECTURE_GUIDE.md).
CREATE TABLE credential (
    user_id UUID PRIMARY KEY REFERENCES app_user(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE refresh_token (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE token_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
