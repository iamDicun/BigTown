import Phaser from 'phaser'
import type { BehaviorHandler, ExtraCollider } from './types'

function resolveZones(sprite: Phaser.GameObjects.Image, meta: Record<string, unknown>): ExtraCollider[] {
  if (Array.isArray(meta.bridge_zones) && meta.bridge_zones.length > 0) {
    return meta.bridge_zones as ExtraCollider[]
  }
  if (Array.isArray(meta.extra_colliders) && meta.extra_colliders.length > 0) {
    return meta.extra_colliders as ExtraCollider[]
  }

  const itemCode = sprite.getData('itemCode') as string
  if (itemCode && itemCode.startsWith('deco_bridge_h_')) {
    return [
      { dx: 0, dy: -28, w: 48, h: 8 },
      { dx: 0, dy: -4,  w: 48, h: 8 },
    ]
  }
  if (itemCode && itemCode.startsWith('deco_bridge_v_')) {
    return [
      { dx: -20, dy: -16, w: 8, h: 32 },
      { dx:  20, dy: -16, w: 8, h: 32 },
    ]
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

  onUpdate(sprite, meta, state) {
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
