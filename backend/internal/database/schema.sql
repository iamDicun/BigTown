-- BigTown MVP schema: auth/user core + persistent game data.
-- Realtime state such as player position, NPC current HP and cooldowns lives in Go RAM.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE app_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(150) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'User'
);

-- Tách khỏi app_user để app_user chỉ giữ thông tin hồ sơ, giống pattern employee/authen của
-- project gốc (xem ARCHITECTURE_GUIDE.md).
CREATE TABLE credential (
    user_id UUID PRIMARY KEY REFERENCES app_user(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE refresh_token (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE token_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    external_subject VARCHAR(150) NOT NULL,
    tenant_id VARCHAR(150) NOT NULL,
    email VARCHAR(150),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, tenant_id, external_subject),
    UNIQUE(provider, tenant_id, email)
);

CREATE TABLE maps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    tilemap_asset_key VARCHAR(255) NOT NULL,
    tileset_asset_key VARCHAR(255) NOT NULL,
    collision_asset_key VARCHAR(255),
    spawn_x INTEGER NOT NULL,
    spawn_y INTEGER NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE characters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    name VARCHAR(80) NOT NULL,
    map_id UUID REFERENCES maps(id),
    base_asset_key VARCHAR(255) NOT NULL,
    coins INTEGER NOT NULL DEFAULT 0 CHECK (coins >= 0),
    score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0),
    last_x INTEGER,
    last_y INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    type VARCHAR(50) NOT NULL,
    slot VARCHAR(50),
    asset_key VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL DEFAULT 0 CHECK (price >= 0),
    attack_bonus INTEGER NOT NULL DEFAULT 0,
    hp_bonus INTEGER NOT NULL DEFAULT 0,
    metadata_json JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE player_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    acquired_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(character_id, item_id)
);

CREATE TABLE character_equipment (
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    slot VARCHAR(50) NOT NULL,
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE RESTRICT,
    equipped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(character_id, slot)
);

CREATE TABLE npc_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    asset_key VARCHAR(255) NOT NULL,
    max_hp INTEGER NOT NULL CHECK (max_hp > 0),
    attack INTEGER NOT NULL DEFAULT 0,
    reward_score INTEGER NOT NULL DEFAULT 0 CHECK (reward_score >= 0),
    reward_coin INTEGER NOT NULL DEFAULT 0 CHECK (reward_coin >= 0),
    respawn_ms INTEGER NOT NULL DEFAULT 5000 CHECK (respawn_ms >= 0),
    metadata_json JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE map_npc_spawns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    map_id UUID NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
    npc_type_id UUID NOT NULL REFERENCES npc_types(id) ON DELETE RESTRICT,
    spawn_x INTEGER NOT NULL,
    spawn_y INTEGER NOT NULL,
    spawn_group VARCHAR(80),
    respawn_ms INTEGER CHECK (respawn_ms IS NULL OR respawn_ms >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE reward_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    npc_type_id UUID REFERENCES npc_types(id) ON DELETE SET NULL,
    score_delta INTEGER NOT NULL DEFAULT 0,
    coin_delta INTEGER NOT NULL DEFAULT 0,
    metadata_json JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id VARCHAR(120) NOT NULL,
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    message_type VARCHAR(30) NOT NULL DEFAULT 'text',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_characters_score ON characters(score DESC);
CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);
CREATE INDEX idx_player_items_character_id ON player_items(character_id);
CREATE INDEX idx_reward_events_character_id ON reward_events(character_id);
CREATE INDEX idx_map_npc_spawns_map_id ON map_npc_spawns(map_id);
CREATE INDEX idx_chat_messages_room_created_at ON chat_messages(room_id, created_at DESC);
