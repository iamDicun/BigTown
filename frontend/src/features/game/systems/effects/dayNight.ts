import type Phaser from 'phaser'
import {
  type MaskSpec, type DayNightLayers, type Focus,
  makeMaskLayer, syncMaskLayer, FX_DEPTH,
  GLOW_TEX_SIZE, LANTERN_GLOW_KEY, LANTERN_GLOW_ALPHA, BLEND_ADD,
  ensureGlowTexture,
  getDarkness,
} from './common'

// --- Tinh chỉnh --------------------------------------------------------------
const NIGHT_MASK: MaskSpec = { key: 'fx_night_mask', hole: 120, falloff: 250, alpha: 0.94, rgb: '4,9,28' }
const WINTER_NIGHT_MASK: MaskSpec = { key: 'fx_wn_mask', hole: 130, falloff: 300, alpha: 0.85, rgb: '4,10,28' }
// -----------------------------------------------------------------------------

// ============================================================================
// Day / night (village_adventure, winter)
// ============================================================================

export function makeDayNightLayers(
  scene: Phaser.Scene,
  cam: Phaser.Cameras.Scene2D.Camera,
  maskSpec: MaskSpec,
): DayNightLayers {
  const mask = makeMaskLayer(scene, cam, maskSpec)
  mask.img.setAlpha(0)

  ensureGlowTexture(scene)
  const glowScale = (maskSpec.hole * 2.4) / GLOW_TEX_SIZE
  const glow = scene.add.image(cam.scrollX, cam.scrollY, LANTERN_GLOW_KEY)
  glow.setDepth(FX_DEPTH + 1)
  glow.setBlendMode(BLEND_ADD)
  glow.setScale(glowScale)
  glow.setAlpha(0)

  return { mask, glow }
}

export function createDayNight(
  scene: Phaser.Scene,
  cam: Phaser.Cameras.Scene2D.Camera,
) {
  const { mask, glow } = makeDayNightLayers(scene, cam, NIGHT_MASK)
  return { mask, glow, scene, cam }
}

export function createWinterDayNight(
  scene: Phaser.Scene,
  cam: Phaser.Cameras.Scene2D.Camera,
): DayNightLayers {
  return makeDayNightLayers(scene, cam, WINTER_NIGHT_MASK)
}

export function updateDayNightLayers(
  scene: Phaser.Scene,
  dn: DayNightLayers,
  cam: Phaser.Cameras.Scene2D.Camera,
  focus: Focus,
  now: number,
): void {
  const darkness = getDarkness()

  syncMaskLayer(scene, dn.mask, cam)
  dn.mask.img.setPosition(focus.x, focus.y)
  dn.mask.img.setAlpha(darkness)
  dn.mask.img.setVisible(darkness > 0.02)

  const flicker = 0.92 + Math.sin(now * 0.008) * 0.05 + Math.sin(now * 0.021) * 0.03
  dn.glow.setPosition(focus.x, focus.y)
  dn.glow.setAlpha(darkness * LANTERN_GLOW_ALPHA * flicker)
  dn.glow.setVisible(darkness > 0.05)
}

export function updateDayNight(
  fx: ReturnType<typeof createDayNight> & { scene: Phaser.Scene; cam: Phaser.Cameras.Scene2D.Camera },
  focus: Focus,
  now: number,
): void {
  updateDayNightLayers(fx.scene, { mask: fx.mask, glow: fx.glow }, fx.cam, focus, now)
}

export function destroyDayNight(fx: ReturnType<typeof createDayNight>): void {
  fx.mask.img.destroy()
  fx.glow.destroy()
}
