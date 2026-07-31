-- BigTown MVP seed data: maps.
-- Idempotent (ON CONFLICT) nên chạy lại nhiều lần không lỗi/không tạo trùng.

INSERT INTO maps (
    code, name, tilemap_asset_key, tileset_asset_key,
    collision_asset_key, spawn_x, spawn_y, width, height, tile_size,
    layer_names, above_layer_name, collision_layer_name, music_asset_key
) VALUES (
    'village_adventure',
    'Village Adventure',
    'maps/village_adventure.tmj',
    'Grass_Middle,Path_Middle,Water_Middle,Water_Tile,House_1_Wood_Base_Blue,Oak_Tree,Oak_Tree_Small,Fences,Chest,Outdoor_Decor_Free,Cliff_Tile,Path_Tile,FarmLand_Tile,Beach_Tile',
    NULL,
    2000,
    2560,
    250,
    175,
    16,
    'Ground,DecorationBelow,Objects,DecorationAbove',
    'DecorationAbove',
    'Collision',
    'sounds/bgm.mp3'
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
