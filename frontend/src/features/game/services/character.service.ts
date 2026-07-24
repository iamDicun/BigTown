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

export interface SpritesheetConfigDto {
  frame_width: number
  frame_height: number
  columns: number
  row_idle_down: number
  row_walk_down: number
  row_idle_up: number
  row_walk_up: number
  row_walk_side: number
  walk_frame_rate: number
  idle_frame_rate: number
}

export interface CharacterOptionDto {
  name: string
  base_asset_key: string
  preview_url: string
  spritesheet: SpritesheetConfigDto
}

export interface CreateCharacterPayload {
  name: string
  base_asset_key: string
}

export function getMe() {
  return http.get<CharacterDto>('/characters/me')
}

export function getOptions() {
  return http.get<CharacterOptionDto[]>('/characters/options')
}

export function createCharacter(payload: CreateCharacterPayload) {
  return http.post<CharacterDto>('/characters', payload)
}

export function getCharacterSpriteUrl(baseAssetKey: string) {
  if (baseAssetKey === 'player' || baseAssetKey === 'cute_fantasy/player_base') {
    return '/assets/player/Player.png'
  }
  return `/assets/player/${baseAssetKey}.png`
}
