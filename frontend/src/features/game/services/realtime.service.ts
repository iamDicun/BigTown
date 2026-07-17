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
}

export function getBootstrap() {
  return http.get<BootstrapDto>('/realtime/bootstrap')
}
