-- =============================================================================
-- BigTown — Phase 2 integration / migration (IDEMPOTENT)
-- Áp mọi QUYẾT ĐỊNH schema từ Phase 2 lên MỘT DB ĐANG CHẠY.
--
-- Vì sao cần file này: schema.sql là "create-only" và chỉ chạy khi Postgres
-- khởi tạo volume mới (docker-entrypoint-initdb.d). DB đã tồn tại (vd Render)
-- sẽ KHÔNG tự có cột/bảng mới → chạy file này để "kéo" DB lên đúng Phase 2.
--
-- An toàn chạy lại nhiều lần: dùng IF NOT EXISTS / ADD COLUMN IF NOT EXISTS.
-- Cách chạy:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f phase2_migrate.sql
-- (hoặc dùng apply_phase2.sh)
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- QĐ #3 — Nhạc theo map (lưu DB, bỏ hardcode path).
-- Thêm cột music_asset_key vào maps (đường dẫn tương đối từ /assets/).
-- ---------------------------------------------------------------------------
ALTER TABLE maps ADD COLUMN IF NOT EXISTS music_asset_key VARCHAR(255);

-- ---------------------------------------------------------------------------
-- QĐ #6 — Kéo thả asset trang trí map (multiplayer building).
-- Bảng map_placements: nguồn sự thật bền vững cho decoration người chơi đặt.
-- (Định nghĩa trùng khít với bản sẽ thêm vào schema.sql cho DB tương lai.)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS map_placements (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    map_id       UUID NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    item_id      UUID NOT NULL REFERENCES items(id) ON DELETE RESTRICT,
    x            INTEGER NOT NULL,
    y            INTEGER NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_map_placements_map_id ON map_placements(map_id);

-- ---------------------------------------------------------------------------
-- QĐ #7 — Combat: các bảng npc_types / map_npc_spawns / reward_events đã có
-- trong schema.sql gốc. Guard IF NOT EXISTS chỉ để an toàn với DB rất cũ
-- (nếu đã tồn tại thì các câu dưới là no-op).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS npc_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    asset_key VARCHAR(255) NOT NULL,
    max_hp INTEGER NOT NULL CHECK (max_hp > 0),
    attack INTEGER NOT NULL DEFAULT 0,
    reward_score INTEGER NOT NULL DEFAULT 0 CHECK (reward_score >= 0),
    reward_coin INTEGER NOT NULL DEFAULT 0 CHECK (reward_coin >= 0),
    respawn_ms INTEGER NOT NULL DEFAULT 5000 CHECK (respawn_ms >= 0),
    metadata_json JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS map_npc_spawns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    map_id UUID NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
    npc_type_id UUID NOT NULL REFERENCES npc_types(id) ON DELETE RESTRICT,
    spawn_x INTEGER NOT NULL,
    spawn_y INTEGER NOT NULL,
    spawn_group VARCHAR(80),
    respawn_ms INTEGER CHECK (respawn_ms IS NULL OR respawn_ms >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_map_npc_spawns_map_id ON map_npc_spawns(map_id);

COMMIT;

-- Ghi chú: QĐ ★ (starting coins qua GAME_STARTING_COINS) là thay đổi CODE
-- (character INSERT + config), không phải schema. Phần cập nhật coin cho
-- nhân vật ĐÃ tồn tại nằm ở phase2_seed.sql (dữ liệu, không phải DDL).
