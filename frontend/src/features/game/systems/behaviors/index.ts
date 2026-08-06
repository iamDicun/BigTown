import type Phaser from 'phaser'
import type { BehaviorContext, BehaviorHandler, ItemMeta, UpdateState } from './types'
import { fadeBehind } from './fadeBehind'
import { glowNight } from './glowNight'
import { bridge } from './bridge'

export type { BehaviorContext, BehaviorHandler, ItemMeta, UpdateState } from './types'

export const BEHAVIORS: Record<string, BehaviorHandler> = {
  fade_behind: fadeBehind,
  glow_night: glowNight,
  bridge,
}

export function parseItemMeta(raw: string): ItemMeta {
  try {
    return JSON.parse(raw) as ItemMeta
  } catch {
    return {} as ItemMeta
  }
}

export function applyBehaviorsOnCreate(
  sprite: Phaser.GameObjects.Image,
  meta: ItemMeta,
  ctx: BehaviorContext,
): void {
  if (!meta.behaviors) return
  for (const name of meta.behaviors) {
    BEHAVIORS[name]?.onCreate?.(sprite, meta, ctx)
  }
}

export function applyBehaviorsOnUpdate(
  sprite: Phaser.GameObjects.Image,
  meta: ItemMeta,
  state: UpdateState,
): void {
  if (!meta.behaviors) return
  for (const name of meta.behaviors) {
    BEHAVIORS[name]?.onUpdate?.(sprite, meta, state)
  }
}

export function applyBehaviorsOnDestroy(sprite: Phaser.GameObjects.Image, meta: ItemMeta): void {
  if (!meta.behaviors) return
  for (const name of meta.behaviors) {
    BEHAVIORS[name]?.onDestroy?.(sprite)
  }
}
