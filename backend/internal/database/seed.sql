-- BigTown MVP seed data: maps.
-- Idempotent (ON CONFLICT) nên chạy lại nhiều lần không lỗi/không tạo trùng.

INSERT INTO maps (
    code, name, tilemap_asset_key, tileset_asset_key,
    collision_asset_key, spawn_x, spawn_y, width, height
) VALUES (
    'village_adventure',
    'Village Adventure',
    'maps/village_adventure.tmj',
    'Grass_Middle,Path_Middle,Water_Middle,Water_Tile,House_1_Wood_Base_Blue,Oak_Tree,Oak_Tree_Small,Fences,Chest,Outdoor_Decor_Free',
    NULL,
    384,
    512,
    50,
    35
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
    updated_at = CURRENT_TIMESTAMP;
