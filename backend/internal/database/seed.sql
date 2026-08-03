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
  ('deco_oak_tree',     'Cây sồi',       'decoration', 'decorations/Oak_Tree.png',                 220,
     '{"w":64,"h":80,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":64,"collision_h":50}'),
  ('deco_house_blue', 'Nhà gỗ xanh',   'decoration', 'decorations/House_1_Wood_Base_Blue.png', 500,
     '{"w":64,"h":80,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":64,"collision_h":80}'),
  ('deco_chest',      'Rương gỗ',      'decoration', 'decorations/Chest.png',                   90,
     '{"w":24,"h":24,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":24,"collision_h":16}'),

  ('deco_fence_single_top', 'Cột đơn trên', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":0}'),
  ('deco_fence_single_left', 'Cột đơn trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":1}'),
  ('deco_fence_t_single', 'Cột T đơn', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":2}'),
  ('deco_fence_single_right', 'Cột đơn phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":3}'),
  ('deco_fence_straight', 'Cột nối thẳng', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":4}'),
  ('deco_fence_corner_tl', 'Cột góc trên trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":5}'),
  ('deco_fence_t_top', 'Cột T trên', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":6}'),
  ('deco_fence_corner_tr', 'Cột góc trên phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":7}'),
  ('deco_fence_single_bottom', 'Cột đơn dưới', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":8}'),
  ('deco_fence_t_left', 'Cột T trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":9}'),
  ('deco_fence_cross', 'Cột cộng', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":10}'),
  ('deco_fence_t_right', 'Cột T phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":11}'),
  ('deco_fence_single', 'Cột đơn', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":12}'),
  ('deco_fence_corner_bl', 'Cột góc dưới trái', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":13}'),
  ('deco_fence_t_bottom', 'Cột T dưới', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":14}'),
  ('deco_fence_corner_br', 'Cột góc dưới phải', 'decoration', 'decorations/Fences.png',                  30,
     '{"w":16,"h":16,"anchorX":0.5,"anchorY":1.0,"collides":true,"frameWidth":16,"frameHeight":16,"frame":15}'),

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
  ('deco_lamppost', 'Cột đèn', 'decoration', 'decorations/Lamppost.png', 150, '{"w":16,"h":48,"anchorX":0.5,"anchorY":1.0,"collides":true,"collision_w":16,"collision_h":16}')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name, type = EXCLUDED.type, asset_key = EXCLUDED.asset_key,
  price = EXCLUDED.price, metadata_json = EXCLUDED.metadata_json,
  updated_at = CURRENT_TIMESTAMP;
