import { http } from '@/shared/api/http'

export interface DecorationItemDto {
  id: string
  code: string
  name: string
  type: string
  asset_key: string
  price: number
  metadata_json: string
}

export interface PlacementDto {
  id: string
  map_id: string
  character_id: string
  item_id: string
  x: number
  y: number
  created_at: string
}

export interface SpawnedCoinDto {
  id: string
  type: string
  x: number
  y: number
}

export interface EditorDataDto {
  items: DecorationItemDto[]
  placements: PlacementDto[]
  coins: number
  spawned_coins: SpawnedCoinDto[]
}

export interface PlaceItemPayload {
  item_id: string
  map_code: string
  x: number
  y: number
}

export interface PlaceItemResultDto {
  placement: PlacementDto
  new_coins: number
}

export interface DeletePlacementResultDto {
  new_coins: number
}

export function getEditorData(mapCode: string) {
  return http.get<EditorDataDto>(`/editor?map_code=${mapCode}`)
}

export function placeItem(payload: PlaceItemPayload) {
  return http.post<PlaceItemResultDto>('/editor/place', payload)
}

export function deletePlacement(id: string, mapCode: string) {
  return http.delete<DeletePlacementResultDto>(`/editor/place/${id}?map_code=${encodeURIComponent(mapCode)}`)
}

export function claimCoinPickup(mapCode: string, coinId: string) {
  return http.post<{ new_coins: number }>('/editor/coin-pickup', { map_code: mapCode, coin_id: coinId })
}
