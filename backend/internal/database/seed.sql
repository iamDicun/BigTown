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
    'village_adventure'
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
  ('deco_oak_tree',     'Cây sồi',       'decoration', 'decorations/Oak_Tree.png',                 220,
     '{"w":64,"h":80,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":64,"collision_h":50,"behaviors":["fade_behind"]}'),
  ('deco_house_blue', 'Nhà gỗ xanh',   'decoration', 'decorations/House_1_Wood_Base_Blue.png', 500,
     '{"w":64,"h":80,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":64,"collision_h":80}'),
  ('deco_chest',      'Rương gỗ',      'decoration', 'decorations/Chest.png',                   90,
     '{"w":24,"h":24,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":24,"collision_h":16}'),

  ('deco_fence_single_top', 'Cột đơn trên', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('deco_fence_single_left', 'Cột đơn trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('deco_fence_t_single', 'Cột T đơn', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('deco_fence_single_right', 'Cột đơn phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('deco_fence_straight', 'Cột nối thẳng', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":4}'),
  ('deco_fence_corner_tl', 'Cột góc trên trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('deco_fence_t_top', 'Cột T trên', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":6}'),
  ('deco_fence_corner_tr', 'Cột góc trên phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('deco_fence_single_bottom', 'Cột đơn dưới', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('deco_fence_t_left', 'Cột T trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":9}'),
  ('deco_fence_cross', 'Cột cộng', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":10}'),
  ('deco_fence_t_right', 'Cột T phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":11}'),
  ('deco_fence_single', 'Cột đơn', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":12}'),
  ('deco_fence_corner_bl', 'Cột góc dưới trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":13}'),
  ('deco_fence_t_bottom', 'Cột T dưới', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":14}'),
  ('deco_fence_corner_br', 'Cột góc dưới phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":15}'),

  -- Grass tufts (No collision)
  ('deco_grass_0', 'Cỏ lá A', 'decoration', 'decorations/Outdoor_Decor_Free.png', 10, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('deco_grass_1', 'Cỏ lá B', 'decoration', 'decorations/Outdoor_Decor_Free.png', 10, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('deco_grass_2', 'Cỏ lá C', 'decoration', 'decorations/Outdoor_Decor_Free.png', 10, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":2}'),

  -- Flower grass tufts (No collision)
  ('deco_grass_flower_7', 'Cỏ hoa A', 'decoration', 'decorations/Outdoor_Decor_Free.png', 15, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('deco_grass_flower_8', 'Cỏ hoa B', 'decoration', 'decorations/Outdoor_Decor_Free.png', 15, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('deco_grass_flower_9', 'Cỏ hoa C', 'decoration', 'decorations/Outdoor_Decor_Free.png', 15, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":9}'),

  -- Rocks / Stump (With collision)
  ('deco_stump', 'Gốc cây khô', 'decoration', 'decorations/Outdoor_Decor_Free.png', 50, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":14}'),
  ('deco_rock_small', 'Đá nhỏ', 'decoration', 'decorations/Outdoor_Decor_Free.png', 30, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":15}'),
  ('deco_rock_medium', 'Đá trung', 'decoration', 'decorations/Outdoor_Decor_Free.png', 40, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":16}'),
  ('deco_rock_pile_1', 'Đống đá A', 'decoration', 'decorations/Outdoor_Decor_Free.png', 45, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":21}'),
  ('deco_rock_tall', 'Đá tảng đứng', 'decoration', 'decorations/Outdoor_Decor_Free.png', 60, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":22}'),
  ('deco_rock_pile_2', 'Đống đá B', 'decoration', 'decorations/Outdoor_Decor_Free.png', 45, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":23}'),

  -- Flowers 56-59 (No collision)
  ('deco_flower_ground_56', 'Hoa tulip đỏ', 'decoration', 'decorations/Outdoor_Decor_Free.png', 20, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":56}'),
  ('deco_flower_ground_57', 'Hoa bồ công anh', 'decoration', 'decorations/Outdoor_Decor_Free.png', 20, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":57}'),
  ('deco_flower_ground_58', 'Hoa anh thảo hồng', 'decoration', 'decorations/Outdoor_Decor_Free.png', 20, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":58}'),
  ('deco_flower_ground_59', 'Hoa cúc vạn thọ', 'decoration', 'decorations/Outdoor_Decor_Free.png', 20, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":59}'),

  -- Flowers 70-73 (No collision)
  ('deco_flower_pot_70', 'Chậu hoa hồng đỏ', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":70}'),
  ('deco_flower_pot_71', 'Chậu cúc vàng', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":71}'),
  ('deco_flower_pot_72', 'Chậu cẩm tú cầu hồng', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":72}'),
  ('deco_flower_pot_73', 'Chậu cúc đồng tiền cam', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":73}'),

  -- Flowers 77-80 (No collision)
  ('deco_flower_bush_77', 'Bụi hoa lồng đèn đỏ', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":77}'),
  ('deco_flower_bush_78', 'Bụi chuông vàng', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":78}'),
  ('deco_flower_bush_79', 'Bụi tường vy hồng', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":79}'),
  ('deco_flower_bush_80', 'Bụi xác pháo cam', 'decoration', 'decorations/Outdoor_Decor_Free.png', 25, '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":80}'),

  -- Lamppost (With base collision)
  ('deco_lamppost', 'Cột đèn', 'decoration', 'decorations/Lamppost.png', 150, '{"w":16,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":16,"collision_h":16,"behaviors":["glow_night"]}'),

   ('deco_bridge', 'Cầu gỗ', 'decoration', 'decorations/Bridge_Wood.png', 300,
    '{"w":48,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":false,"collision_override":true,"behaviors":["bridge"],"bridge_zones":[{"dx":-20,"dy":-16,"w":8,"h":32},{"dx":20,"dy":-16,"w":8,"h":32}],"bridge_zones_h":[{"dx":0,"dy":-36,"w":48,"h":8},{"dx":0,"dy":-4,"w":48,"h":8}]}'),

  -- ============================================================
  -- TILE PLACEMENT ITEMS (đặt tile terrain như decoration)
  -- ============================================================

  -- Water_Tile (16 frames, autotile nước)
  ('tile_water_0',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('tile_water_1',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('tile_water_2',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('tile_water_3',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('tile_water_4',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":4}'),
  ('tile_water_5',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('tile_water_6',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":6}'),
  ('tile_water_7',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('tile_water_8',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('tile_water_9',  'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":9}'),
  ('tile_water_10', 'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":10}'),
  ('tile_water_12', 'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":12}'),
  ('tile_water_13', 'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":13}'),
  ('tile_water_15', 'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":15}'),
  ('tile_water_16', 'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":16}'),
  ('tile_water_17', 'Nước',   'decoration', 'tiles/Water_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":17}'),

  -- Cliff_Tile (13 frames, autotile vách đá)
  ('tile_cliff_0',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('tile_cliff_1',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('tile_cliff_2',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('tile_cliff_3',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('tile_cliff_5',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('tile_cliff_6',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":6}'),
  ('tile_cliff_7',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('tile_cliff_8',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('tile_cliff_9',  'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":9}'),
  ('tile_cliff_10', 'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":10}'),
  ('tile_cliff_12', 'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":12}'),
  ('tile_cliff_13', 'Vách đá',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":13}'),
  ('tile_cliff_15', 'Cỏ',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":15}'),
  ('tile_cliff_16', 'Cỏ',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":16}'),
  ('tile_cliff_17', 'Cỏ',   'decoration', 'tiles/Cliff_Tile.png',  20, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":17}'),

  -- Path_Tile (14 frames, autotile đường đất)
  ('tile_path_0',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('tile_path_1',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('tile_path_2',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('tile_path_3',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('tile_path_5',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('tile_path_6',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":6}'),
  ('tile_path_7',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('tile_path_8',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('tile_path_9',  'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":9}'),
  ('tile_path_10', 'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":10}'),
  ('tile_path_12', 'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":12}'),
  ('tile_path_13', 'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":13}'),
  ('tile_path_15', 'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":15}'),
  ('tile_path_16', 'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":16}'),
  ('tile_path_17', 'Đường đất',   'decoration', 'tiles/Path_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":17}'),

  -- FarmLand_Tile (9 frames)
  ('tile_farm_0', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('tile_farm_1', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('tile_farm_2', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('tile_farm_3', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('tile_farm_4', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":4}'),
  ('tile_farm_5', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('tile_farm_6', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":6}'),
  ('tile_farm_7', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('tile_farm_8', 'Đất trồng', 'decoration', 'tiles/FarmLand_Tile.png', 10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false,"frameWidth":16,"frameHeight":16,"frame":8}'),

  -- Beach_Tile (12 frames)
  ('tile_beach_0',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('tile_beach_1',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('tile_beach_2',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('tile_beach_3',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('tile_beach_4',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":4}'),
  ('tile_beach_5',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('tile_beach_7',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('tile_beach_8',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('tile_beach_9',  'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":9}'),
  ('tile_beach_10', 'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":10}'),
  ('tile_beach_11', 'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":11}'),
  ('tile_beach_12', 'Cát biển',  'decoration', 'tiles/Beach_Tile.png',  10, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":true,"collision_x":8,"collision_y":8,"frameWidth":16,"frameHeight":16,"frame":12}'),

  -- Path_Middle (1 frame)
  ('tile_path_mid', 'Đường giữa', 'decoration', 'tiles/Path_Middle.png', 5, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false}'),

  -- Grass_Middle (1 frame)
  ('tile_grass_mid', 'Cỏ giữa', 'decoration', 'tiles/Grass_Middle.png', 5, '{"w":16,"h":16,"anchorX":0,"anchorY":0,"collides":false}')

ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name, type = EXCLUDED.type, asset_key = EXCLUDED.asset_key,
  price = EXCLUDED.price, metadata_json = EXCLUDED.metadata_json,
  updated_at = CURRENT_TIMESTAMP;

-- ============================================================
-- NPC TYPES (animal, không combat)
-- ============================================================

-- Xóa NPC spawn cũ (enemy từ phase2 cũ nếu có) trước khi seed mới
DELETE FROM map_npc_spawns WHERE map_id = (SELECT id FROM maps WHERE code = 'village_adventure');
DELETE FROM npc_types WHERE code NOT LIKE 'animal_%';

INSERT INTO npc_types (code, name, asset_key, max_hp, attack, reward_score, reward_coin, respawn_ms, metadata_json) VALUES
  ('animal_chicken', 'Gà', 'animals/Chicken.png', 1, 0, 0, 0, 0, '{"frame_width":32,"frame_height":32,"columns":2,"row_idle":0,"row_walk":1,"idle_frame_rate":4,"walk_frame_rate":6,"wander_radius":48,"wander_delay_min":2000,"wander_delay_max":5000}'),
  ('animal_cow',     'Bò', 'animals/Cow.png',     1, 0, 0, 0, 0, '{"frame_width":32,"frame_height":32,"columns":2,"row_idle":0,"row_walk":1,"idle_frame_rate":4,"walk_frame_rate":6,"wander_radius":48,"wander_delay_min":2000,"wander_delay_max":5000}'),
  ('animal_pig',     'Heo', 'animals/Pig.png',    1, 0, 0, 0, 0, '{"frame_width":32,"frame_height":32,"columns":2,"row_idle":0,"row_walk":1,"idle_frame_rate":4,"walk_frame_rate":6,"wander_radius":48,"wander_delay_min":2000,"wander_delay_max":5000}'),
  ('animal_sheep',   'Cừu', 'animals/Sheep.png',  1, 0, 0, 0, 0, '{"frame_width":32,"frame_height":32,"columns":2,"row_idle":0,"row_walk":1,"idle_frame_rate":4,"walk_frame_rate":6,"wander_radius":48,"wander_delay_min":2000,"wander_delay_max":5000}')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name, asset_key = EXCLUDED.asset_key,
  max_hp = EXCLUDED.max_hp, attack = EXCLUDED.attack,
  reward_score = EXCLUDED.reward_score, reward_coin = EXCLUDED.reward_coin,
  respawn_ms = EXCLUDED.respawn_ms, metadata_json = EXCLUDED.metadata_json,
  updated_at = CURRENT_TIMESTAMP;

-- ============================================================
-- MAP NPC SPAWNS (village_adventure)
-- ============================================================

INSERT INTO map_npc_spawns (map_id, npc_type_id, spawn_x, spawn_y, spawn_group, respawn_ms)
SELECT m.id, nt.id, s.x, s.y, s.grp, NULL
FROM maps m, npc_types nt,
(VALUES
  -- Đàn gà 1 (5 con, tọa độ cỏ ~480-512)
  ('animal_chicken', 496, 496, 'chicken_flock_1'),
  ('animal_chicken', 480, 480, 'chicken_flock_1'),
  ('animal_chicken', 480, 496, 'chicken_flock_1'),
  ('animal_chicken', 480, 512, 'chicken_flock_1'),
  ('animal_chicken', 496, 480, 'chicken_flock_1'),
  -- Đàn gà 2 (5 con, ~1472-1504)
  ('animal_chicken', 1488, 288, 'chicken_flock_2'),
  ('animal_chicken', 1472, 272, 'chicken_flock_2'),
  ('animal_chicken', 1472, 288, 'chicken_flock_2'),
  ('animal_chicken', 1472, 304, 'chicken_flock_2'),
  ('animal_chicken', 1488, 272, 'chicken_flock_2'),
  -- Đàn gà 3 (5 con, ~2384-2416)
  ('animal_chicken', 2400, 1792, 'chicken_flock_3'),
  ('animal_chicken', 2384, 1776, 'chicken_flock_3'),
  ('animal_chicken', 2384, 1792, 'chicken_flock_3'),
  ('animal_chicken', 2384, 1808, 'chicken_flock_3'),
  ('animal_chicken', 2400, 1776, 'chicken_flock_3'),
  -- Heo (5 con, ~1072-1104)
  ('animal_pig', 1088, 1392, 'pig_group'),
  ('animal_pig', 1072, 1376, 'pig_group'),
  ('animal_pig', 1072, 1392, 'pig_group'),
  ('animal_pig', 1072, 1408, 'pig_group'),
  ('animal_pig', 1088, 1376, 'pig_group'),
  -- Bò (5 con, ~1776-1808)
  ('animal_cow', 1792, 896, 'cow_group'),
  ('animal_cow', 1776, 880, 'cow_group'),
  ('animal_cow', 1776, 896, 'cow_group'),
  ('animal_cow', 1776, 912, 'cow_group'),
  ('animal_cow', 1792, 880, 'cow_group'),
  -- Cừu (5 con, ~2080-2128)
  ('animal_sheep', 2080, 1072, 'sheep_group'),
  ('animal_sheep', 2096, 1072, 'sheep_group'),
  ('animal_sheep', 2112, 1072, 'sheep_group'),
  ('animal_sheep', 2112, 1088, 'sheep_group'),
  ('animal_sheep', 2112, 1104, 'sheep_group')
) AS s(code, x, y, grp)
WHERE m.code = 'village_adventure' AND nt.code = s.code
ON CONFLICT DO NOTHING;
