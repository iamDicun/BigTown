import { http } from '@/shared/api/http'

// Khớp backend/internal/module/character/delivery/dto.go CharacterResponse.
export interface CharacterDto {
  id: string
  name: string
  map_id: string | null
  base_asset_key: string
  coins: number
  score: number
  last_x: number | null
  last_y: number | null
}

export function getMe() {
  return http.get<CharacterDto>('/characters/me')
}
