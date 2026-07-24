import { http } from '@/shared/api/http'

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
  // Cho phép mỗi map định nghĩa tên layer riêng. Nếu thiếu, mapSystem dùng default cũ.
  layer_names?: string[]
  above_layer_name?: string
  collision_layer_name?: string
}

export function getBootstrap(mapCode?: string) {
  const query = mapCode ? `?map_code=${encodeURIComponent(mapCode)}` : ''
  return http.get<BootstrapDto>(`/realtime/bootstrap${query}`)
}
