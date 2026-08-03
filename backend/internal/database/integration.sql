ALTER TABLE maps
ADD COLUMN layer_names VARCHAR(500),
ADD COLUMN above_layer_name VARCHAR(80),
ADD COLUMN collision_layer_name VARCHAR(80);


INSERT INTO maps (
    code, name, tilemap_asset_key, tileset_asset_key,
    spawn_x, spawn_y, width, height,
    layer_names, above_layer_name, collision_layer_name
) VALUES (
    'winter',
    'Nivalis',
    'maps/winter.tmj',
    'collision,snow,snow_x2,snow_tree_x3,snow_village,snow_ground,snow_x3,snow_village_x3,snow_village_x4,snow_x4',
    64, 84,                      -- cần lấy từ server config
    128, 148,
    'Ground1,DecorationBelow,DecorationAbove,Collision',
    'DecorationAbove',
    'Collision'
);


INSERT INTO maps (
    code, name, tilemap_asset_key, tileset_asset_key,
    spawn_x, spawn_y, width, height,
    layer_names, above_layer_name, collision_layer_name
) VALUES (
    '027-1',
    'Graveyard',
    'maps/027-1.tmj',
    'collision,woodland_ground,woodland_x2,woodland_x3,woodland_village,woodland_x5,woodland_x4,woodland_x8,woodland_swamp,forest,woodland_graveyard_x3,woodland_graveyard_x4,woodland_graveyard_crypt,woodland_graveyard_ground,woodland_village_x3',
    71, 85,                      -- cần lấy từ server config
    150, 120,
    'Ground,DecorationAbove,DecorationBelow,Collision',
    'DecorationAbove',
    'Collision'
);

ALTER TABLE maps ADD COLUMN IF NOT EXISTS tile_size INTEGER NOT NULL DEFAULT 16;


ALTER TABLE maps ADD COLUMN IF NOT EXISTS music_asset_key VARCHAR(255);
UPDATE maps SET music_asset_key = 'sounds/bgm.mp3' WHERE code = 'village_adventure';


UPDATE characters SET coins = GREATEST(coins, 5000), updated_at = CURRENT_TIMESTAMP;


INSERT INTO maps (
    code, name, tilemap_asset_key, tileset_asset_key,
    collision_asset_key, spawn_x, spawn_y, width, height, tile_size,
    layer_names, above_layer_name, collision_layer_name, music_asset_key
) VALUES
(
    'dark_village',
    'Graveyard',
    'maps/dark_village.tmj',
    'collision,woodland_ground,woodland_x2,woodland_x3,woodland_village,woodland_x5,woodland_x4,woodland_x8,woodland_swamp,forest,woodland_graveyard_x3,woodland_graveyard_x4,woodland_graveyard_crypt,woodland_graveyard_ground,woodland_village_x3',
    NULL,
    2272,
    2688,
    150,
    120,
    32,
    'Ground,DecorationBelow,DecorationAbove,Collision',
    'DecorationAbove',
    'Collision',
    'sounds/dark_village.mp3'
),
(
    'winter',
    'Nivalis',
    'maps/winter.tmj',
    'collision,snow,snow_x2,snow_tree_x3,snow_village,snow_ground,snow_x3,snow_village_x3,snow_village_x4,snow_x4',
    NULL,
    2272,
    2720,
    128,
    148,
    32,
    'Ground,DecorationBelow,DecorationAbove,Collision',
    'DecorationAbove',
    'Collision',
    'sounds/winter.mp3'
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    tilemap_asset_key = EXCLUDED.tilemap_asset_key,
    tileset_asset_key = EXCLUDED.tileset_asset_key,
    collision_asset_key = EXCLUDED.collision_asset_key,
    spawn_x = EXCLUDED.spawn_x,
    spawn_y = EXCLUDED.spawn_y,
    width = EXCLUDED.width,
    height = EXCLUDED.height,
    tile_size = EXCLUDED.tile_size,
    layer_names = EXCLUDED.layer_names,
    above_layer_name = EXCLUDED.above_layer_name,
    collision_layer_name = EXCLUDED.collision_layer_name,
    music_asset_key = EXCLUDED.music_asset_key,
    updated_at = CURRENT_TIMESTAMP;

-- Phase 2 - Step 6: Map placements and decorations DDL & DML
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_map_placements_coords ON map_placements(map_id, x, y);

DELETE FROM map_placements;
DELETE FROM items WHERE type = 'decoration';

INSERT INTO items (code, name, type, asset_key, price, metadata_json) VALUES
  ('deco_house_blue', 'Nhà gỗ xanh',   'decoration', 'decorations/House_1_Wood_Base_Blue.png', 500,
     '{"w":64,"h":80,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":64,"collision_h":24}'),
  ('deco_oak_tree',   'Cây sồi lớn',   'decoration', 'decorations/Oak_Tree.png',               120,
     '{"w":48,"h":64,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":16,"collision_h":16}'),
  ('deco_chest',      'Rương gỗ',      'decoration', 'decorations/Chest.png',                   90,
     '{"w":24,"h":24,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":24,"collision_h":16}'),

  ('deco_oak_small_1', 'Cây sồi nhỏ A', 'decoration', 'decorations/Oak_Tree_Small.png',          70,
     '{"w":32,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":12,"collision_h":12,"frameWidth":32,"frameHeight":48,"frame":0}'),
  ('deco_oak_small_2', 'Cây sồi nhỏ B', 'decoration', 'decorations/Oak_Tree_Small.png',          70,
     '{"w":32,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":12,"collision_h":12,"frameWidth":32,"frameHeight":48,"frame":1}'),
  ('deco_oak_small_3', 'Cây sồi nhỏ C', 'decoration', 'decorations/Oak_Tree_Small.png',          70,
     '{"w":32,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":12,"collision_h":12,"frameWidth":32,"frameHeight":48,"frame":2}'),

  ('deco_fence_h',      'Rào ngang',     'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('deco_fence_v',      'Rào dọc',       'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":4}'),
  ('deco_fence_post',   'Trụ rào',       'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('deco_fence_corner', 'Rào góc',       'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":2}'),

  ('deco_bridge_h_start', 'Cầu ngang - Đầu',  'decoration', 'decorations/Bridge_Wood.png',       150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":48,"frameHeight":32,"frame":0}'),
  ('deco_bridge_h_mid',   'Cầu ngang - Thân',  'decoration', 'decorations/Bridge_Wood.png',       150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":48,"frameHeight":32,"frame":1}'),
  ('deco_bridge_h_end',   'Cầu ngang - Đuôi',  'decoration', 'decorations/Bridge_Wood.png',       150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":48,"frameHeight":32,"frame":2}'),
  ('deco_bridge_v_start', 'Cầu dọc - Đầu',    'decoration', 'decorations/Bridge_Wood.png',       150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":48,"frameHeight":32,"frame":3}'),
  ('deco_bridge_v_mid',   'Cầu dọc - Thân',    'decoration', 'decorations/Bridge_Wood.png',       150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":48,"frameHeight":32,"frame":4}'),
  ('deco_bridge_v_end',   'Cầu dọc - Đuôi',    'decoration', 'decorations/Bridge_Wood.png',       150,
     '{"w":48,"h":32,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":48,"frameHeight":32,"frame":5}')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name, type = EXCLUDED.type, asset_key = EXCLUDED.asset_key,
  price = EXCLUDED.price, metadata_json = EXCLUDED.metadata_json,
  updated_at = CURRENT_TIMESTAMP;