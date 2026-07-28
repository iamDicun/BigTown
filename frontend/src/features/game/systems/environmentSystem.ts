import type Phaser from 'phaser'
import { type Focus } from './effects/common'
import { createFog, updateFog, destroyFog } from './effects/fog'
import { createSnow, updateSnow, destroySnow } from './effects/snow'
import { createDayNight, createWinterDayNight, updateDayNight, updateDayNightLayers, destroyDayNight } from './effects/dayNight'

// Re-export for GameScene
export { syncWorldTime } from './effects/common'

// ============================================================================
// EnvironmentFx — union type cho 3 map
// ============================================================================

type FogFx = ReturnType<typeof createFog> & { type: 'fog' }
type SnowFx = ReturnType<typeof createSnow> & { type: 'snow' }
type DayNightFx = ReturnType<typeof createDayNight> & { type: 'dayNight' }

export type EnvironmentFx =
  | FogFx
  | SnowFx
  | DayNightFx
  | { type: 'none' }

// ============================================================================
// Create
// ============================================================================

export function createEnvironmentFx(scene: Phaser.Scene, mapCode: string): EnvironmentFx {
  const cam = scene.cameras.main

  switch (mapCode) {
    case 'dark_village': {
      const fx = createFog(scene, cam)
      return { ...fx, type: 'fog' }
    }
    case 'winter': {
      const dn = createWinterDayNight(scene, cam)
      const fx = createSnow(scene, cam, dn)
      return { ...fx, type: 'snow' }
    }
    case 'village_adventure': {
      const fx = createDayNight(scene, cam)
      return { ...fx, type: 'dayNight' }
    }
    default:
      return { type: 'none' }
  }
}

// ============================================================================
// Update
// ============================================================================

export function updateEnvironmentFx(fx: EnvironmentFx, now: number, target?: Focus | null): void {
  if (fx.type === 'none') return

  const v = (fx as FogFx | SnowFx | DayNightFx).cam.worldView
  const focus: Focus = target ?? { x: v.centerX, y: v.centerY }

  switch (fx.type) {
    case 'fog':
      updateFog(fx, now, focus)
      break
    case 'snow': {
      updateSnow(fx, now)
      if (fx.dn) updateDayNightLayers(fx.scene, fx.dn, fx.cam, focus, now)
      break
    }
    case 'dayNight':
      updateDayNight(fx, focus, now)
      break
  }
}

// ============================================================================
// Destroy
// ============================================================================

export function destroyEnvironmentFx(fx: EnvironmentFx): EnvironmentFx {
  switch (fx.type) {
    case 'fog':
      destroyFog(fx)
      break
    case 'snow':
      destroySnow(fx)
      break
    case 'dayNight':
      destroyDayNight(fx)
      break
  }
  return { type: 'none' }
}
