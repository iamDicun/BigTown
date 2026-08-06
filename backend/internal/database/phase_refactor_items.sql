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

-- Cầu gỗ mới (nếu chưa có)
INSERT INTO items (code, name, type, asset_key, price, metadata_json)
SELECT 'deco_bridge', 'Cầu gỗ', 'decoration', 'decorations/Bridge_Wood.png', 300,
       '{"w":48,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":false,"collision_override":true,"behaviors":["bridge"],"bridge_zones":[{"dx":-20,"dy":-16,"w":8,"h":32},{"dx":20,"dy":-16,"w":8,"h":32}],"bridge_zones_h":[{"dx":0,"dy":-36,"w":48,"h":8},{"dx":0,"dy":-4,"w":48,"h":8}]}'
WHERE NOT EXISTS (SELECT 1 FROM items WHERE code = 'deco_bridge');

-- =============================================================================
-- Kiểm tra kết quả:
--   SELECT code, name, metadata_json FROM items WHERE metadata_json::jsonb ? 'behaviors';
-- =============================================================================
