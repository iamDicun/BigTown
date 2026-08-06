import { http } from '@/shared/api/http'

export interface NPCSpawnDto {
  id: string
  code: string
  name: string
  asset_key: string
  spawn_x: number
  spawn_y: number
  spawn_group?: string
  metadata_json: string
}

// Khớp backend/internal/module/realtime/delivery/dto.go BootstrapResponse.
export interface BootstrapDto {
  tick_rate_ms: number
  map_code: string
  websocket_path: string
  default_room_id: string
  default_channel: string
  protocol_features: string[]
  tilemap_asset_key: string
  tileset_asset_key: string
  spawn_x: number
  spawn_y: number
  map_width: number
  map_height: number
  tile_size: number
  layer_names?: string[]
  above_layer_name?: string
  collision_layer_name?: string
  music_asset_key?: string
  npc_spawns?: NPCSpawnDto[]
}

export function getBootstrap(mapCode?: string) {
  const query = mapCode ? `?map_code=${encodeURIComponent(mapCode)}` : ''
  return http.get<BootstrapDto>(`/realtime/bootstrap${query}`)
}
