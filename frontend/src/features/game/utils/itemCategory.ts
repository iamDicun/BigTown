export type Category = 'plants' | 'structures' | 'nature' | 'tiles' | 'other'

export function categoryOf(code: string): Category {
  if (code.includes('tree') || code.includes('flower') || code.includes('bush') || code === 'deco_stump') return 'plants'
  if (code.includes('house') || code.startsWith('deco_fence') || code.startsWith('deco_bridge') || code.startsWith('deco_wall')) return 'structures'
  if (code.includes('rock') || code.startsWith('deco_grass') || code.startsWith('deco_water') || code.startsWith('deco_path')) return 'nature'
  if (code.startsWith('tile_')) return 'tiles'
  return 'other'
}

export const CATEGORY_LABELS: Record<Category, string> = {
  plants: 'Cây cối',
  structures: 'Công trình',
  nature: 'Tự nhiên',
  tiles: 'Nền/Tiles',
  other: 'Khác',
}
