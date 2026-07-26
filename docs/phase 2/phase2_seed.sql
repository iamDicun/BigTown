-- =============================================================================
-- BigTown — Phase 2 seed data (IDEMPOTENT)
-- Nạp DỮ LIỆU cho mọi quyết định Phase 2: starting coins, nhạc theo map,
-- item trang trí (editor kéo-thả), npc_types + map_npc_spawns (combat).
--
-- Chạy SAU phase2_migrate.sql (cần cột music_asset_key + bảng map_placements).
-- An toàn chạy lại: ON CONFLICT (code) DO UPDATE, và guard NOT EXISTS cho spawns.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f phase2_seed.sql
--
-- ⚠️ ASSET FILE cần copy trước khi chạy game (asset_key trỏ tới /assets/...):
--   asset/Outdoor decoration/*.png  ->  frontend/public/assets/decorations/
--   asset/Enemies/*.png             ->  frontend/public/assets/enemies/
--   (nhạc 'sounds/bgm.mp3' đã có sẵn trong frontend/public/assets/sounds/)
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- QĐ ★ — Cấp sẵn coin cho nhân vật ĐÃ tồn tại (balance tạm để test editor
-- trước khi có combat). GREATEST(...) không hạ coin của ai đã có nhiều hơn.
-- (Nhân vật MỚI nhận coin qua GAME_STARTING_COINS trong code — xem doc.)
-- ---------------------------------------------------------------------------
UPDATE characters SET coins = GREATEST(coins, 5000), updated_at = CURRENT_TIMESTAMP;

-- ---------------------------------------------------------------------------
-- QĐ #3 — Gán nhạc cho map. 'sounds/bgm.mp3' đã tồn tại trong public/assets.
-- Thêm winter/dark_village khi các map đó được seed (hiện chỉ có village).
-- ---------------------------------------------------------------------------
UPDATE maps SET music_asset_key = 'sounds/bgm.mp3'
WHERE code = 'village_adventure';

-- ---------------------------------------------------------------------------
-- QĐ #6 — Item trang trí cho editor (type='decoration').
-- metadata_json: w,h = kích thước sprite; anchorX/Y = neo (đáy-giữa);
-- collides=false (MVP không chặn di chuyển).
-- ---------------------------------------------------------------------------
INSERT INTO items (code, name, type, asset_key, price, metadata_json) VALUES
  ('deco_house_blue', 'Nhà gỗ xanh',   'decoration', 'decorations/House_1_Wood_Base_Blue.png', 500,
     '{"w":64,"h":80,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
  ('deco_oak_tree',   'Cây sồi',       'decoration', 'decorations/Oak_Tree.png',               120,
     '{"w":48,"h":64,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
  ('deco_oak_small',  'Cây sồi nhỏ',   'decoration', 'decorations/Oak_Tree_Small.png',          70,
     '{"w":32,"h":40,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
  ('deco_fence',      'Hàng rào',      'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
  ('deco_bridge',     'Cầu gỗ',        'decoration', 'decorations/Bridge_Wood.png',            150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false}'),
  ('deco_chest',      'Rương gỗ',      'decoration', 'decorations/Chest.png',                   90,
     '{"w":24,"h":24,"anchorX":0.5,"anchorY":1.0,"collides":false}')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name, type = EXCLUDED.type, asset_key = EXCLUDED.asset_key,
  price = EXCLUDED.price, metadata_json = EXCLUDED.metadata_json,
  updated_at = CURRENT_TIMESTAMP;

-- ---------------------------------------------------------------------------
-- QĐ #7 — npc_types (enemy để farm coin/điểm sau này).
-- asset_key trỏ enemies/*.png (copy từ asset/Enemies/).
-- ---------------------------------------------------------------------------
INSERT INTO npc_types (code, name, asset_key, max_hp, attack, reward_score, reward_coin, respawn_ms, metadata_json) VALUES
  ('npc_slime_green', 'Slime xanh', 'enemies/Slime_Green.png', 30, 0, 10,  5,  5000,
     '{"frame_width":32,"frame_height":32}'),
  ('npc_skeleton',    'Bộ xương',   'enemies/Skeleton.png',    60, 0, 25, 12, 8000,
     '{"frame_width":32,"frame_height":32}')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name, asset_key = EXCLUDED.asset_key, max_hp = EXCLUDED.max_hp,
  attack = EXCLUDED.attack, reward_score = EXCLUDED.reward_score,
  reward_coin = EXCLUDED.reward_coin, respawn_ms = EXCLUDED.respawn_ms,
  metadata_json = EXCLUDED.metadata_json, updated_at = CURRENT_TIMESTAMP;

-- ---------------------------------------------------------------------------
-- QĐ #7 — map_npc_spawns: vị trí spawn enemy trên village_adventure.
-- Map world ~ 800x560 (50x35 tiles * 16), spawn player ở (384,512) => đặt quái
-- ở nơi khác. Bảng KHÔNG có unique => dùng NOT EXISTS để idempotent theo
-- (map, npc_type, x, y).
-- ---------------------------------------------------------------------------
INSERT INTO map_npc_spawns (map_id, npc_type_id, spawn_x, spawn_y, spawn_group, respawn_ms)
SELECT m.id, n.id, v.x, v.y, v.grp, v.respawn
FROM (VALUES
    ('npc_slime_green', 160, 200, 'grp_slime', 5000),
    ('npc_slime_green', 240, 160, 'grp_slime', 5000),
    ('npc_slime_green', 620, 240, 'grp_slime', 5000),
    ('npc_skeleton',    680, 420, 'grp_skel',  8000),
    ('npc_skeleton',    120, 430, 'grp_skel',  8000)
) AS v(npc_code, x, y, grp, respawn)
JOIN maps m       ON m.code = 'village_adventure'
JOIN npc_types n  ON n.code = v.npc_code
WHERE NOT EXISTS (
    SELECT 1 FROM map_npc_spawns s
    WHERE s.map_id = m.id AND s.npc_type_id = n.id
      AND s.spawn_x = v.x AND s.spawn_y = v.y
);

COMMIT;

-- Kiểm tra nhanh sau khi seed:
--   SELECT code, music_asset_key FROM maps;
--   SELECT code, price FROM items WHERE type='decoration' ORDER BY price;
--   SELECT code, max_hp, reward_coin FROM npc_types;
--   SELECT count(*) FROM map_npc_spawns;
--   SELECT min(coins), max(coins) FROM characters;
