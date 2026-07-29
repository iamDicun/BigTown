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