-- =============================================================================
-- BigTown — Phase refactor: Behavior-based item system
-- Migration cho DB hiện có — thêm "behaviors" vào metadata_json của item đặc thù.
-- IDEMPOTENT: chạy lại nhiều lần không lỗi (WHERE NOT EXISTS).
-- =============================================================================

-- Cây sồi: fade khi người chơi đứng phía sau
UPDATE items
SET metadata_json = metadata_json::jsonb || '{"behaviors":["fade_behind"]}'::jsonb
WHERE code = 'deco_oak_tree'
  AND metadata_json::jsonb -> 'behaviors' IS NULL;

-- Cột đèn: glow + day/night cycle
UPDATE items
SET metadata_json = metadata_json::jsonb || '{"behaviors":["glow_night"]}'::jsonb
WHERE code = 'deco_lamppost'
  AND metadata_json::jsonb -> 'behaviors' IS NULL;

-- Cầu: collision zones walkable (nếu có bridge items trong DB)
-- Bridge items có thể không seed sẵn — UPDATE này an toàn chạy trên mọi DB.
UPDATE items
SET metadata_json = metadata_json::jsonb || '{"behaviors":["bridge"],"collides":false,"collision_override":true}'::jsonb
WHERE code LIKE 'deco_bridge_%'
  AND metadata_json::jsonb -> 'behaviors' IS NULL;

-- =============================================================================
-- Kiểm tra kết quả:
--   SELECT code, name, metadata_json FROM items WHERE metadata_json::jsonb ? 'behaviors';
-- =============================================================================
