export type Category = 'tree' | 'house' | 'fence' | 'bridge' | 'flower' | 'rock' | 'ground' | 'other'

export function categoryOf(code: string): Category {
  if (code.includes('tree') || code === 'deco_stump') return 'tree'
  if (code.includes('house')) return 'house'
  if (code.startsWith('deco_fence')) return 'fence'
  if (code.startsWith('deco_bridge')) return 'bridge'
  if (code.includes('flower')) return 'flower'
  if (code.includes('rock')) return 'rock'
  if (code.startsWith('deco_grass')) return 'ground'
  return 'other'
}

export const CATEGORY_LABELS: Record<Category, string> = {
  tree: 'Cay',
  house: 'Nha',
  fence: 'Hang rao',
  bridge: 'Cau',
  flower: 'Hoa',
  rock: 'Da',
  ground: 'Nen co',
  other: 'Khac',
}
