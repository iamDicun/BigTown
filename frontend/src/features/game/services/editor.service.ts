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

export interface EditorDataDto {
  items: DecorationItemDto[]
  placements: PlacementDto[]
  coins: number
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

export function deletePlacement(id: string) {
  return http.delete<DeletePlacementResultDto>(`/editor/place/${id}`)
}
