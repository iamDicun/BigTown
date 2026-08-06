-- =============================================================================
-- BigTown — Placement rotation
-- Migration cho DB hiện có: thêm cột rotation vào map_placements.
-- IDEMPOTENT: ALTER TABLE ... ADD COLUMN IF NOT EXISTS.
-- =============================================================================

ALTER TABLE map_placements ADD COLUMN IF NOT EXISTS rotation INTEGER NOT NULL DEFAULT 0;

-- =============================================================================
-- Kiểm tra:
--   SELECT * FROM map_placements LIMIT 1;
-- =============================================================================
