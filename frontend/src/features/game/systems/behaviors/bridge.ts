import Phaser from 'phaser'
import type { BehaviorHandler, ExtraCollider, ItemMeta } from './types'

function resolveZones(sprite: Phaser.GameObjects.Image, meta: ItemMeta): ExtraCollider[] {
  const rotation = (sprite.getData('rotation') as number) ?? 0

  if (rotation === 90 && Array.isArray(meta.bridge_zones_h) && meta.bridge_zones_h.length > 0) {
    return meta.bridge_zones_h as ExtraCollider[]
  }
  if (Array.isArray(meta.bridge_zones) && meta.bridge_zones.length > 0) {
    return meta.bridge_zones as ExtraCollider[]
  }
  if (Array.isArray(meta.extra_colliders) && meta.extra_colliders.length > 0) {
    return meta.extra_colliders as ExtraCollider[]
  }
  return []
}

export const bridge: BehaviorHandler = {
  onCreate(sprite, meta, ctx) {
    const zones = resolveZones(sprite, meta)
    if (zones.length === 0) return

    const existing: Phaser.GameObjects.Zone[] = (sprite.getData('extraZones') as Phaser.GameObjects.Zone[]) ?? []

    for (const c of zones) {
      const zone = ctx.scene.add.zone(
        sprite.x + c.dx,
        sprite.y + c.dy,
        c.w,
        c.h,
      )
      ctx.scene.physics.add.existing(zone, true)
      ctx.collisionGroup.add(zone)
      existing.push(zone)
    }

    sprite.setData('extraZones', existing)
  },

  onUpdate(sprite, _meta, state) {
    if (
      Phaser.Geom.Intersects.RectangleToRectangle(
        state.playerBounds,
        sprite.getBounds(),
      )
    ) {
      state.isOnBridge = true
    }
  },

  onDestroy(sprite) {
    const zones = sprite.getData('extraZones') as Phaser.GameObjects.Zone[] | undefined
    if (zones) {
      for (const z of zones) {
        z.destroy()
      }
      sprite.setData('extraZones', null)
    }
  },
}
