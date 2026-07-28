import type Phaser from 'phaser'
import {
  type SnowFlake, type DayNightLayers,
  rand, FX_DEPTH,
  SNOW_FLAKE_KEY, ensureSnowFlakeTexture,
} from './common'

const SNOW_COUNT = 80

// ============================================================================
// Snow (winter)
// ============================================================================

export function createSnow(
  scene: Phaser.Scene,
  cam: Phaser.Cameras.Scene2D.Camera,
  dn: DayNightLayers | undefined,
) {
  ensureSnowFlakeTexture(scene)
  const v = cam.worldView
  const w = v.width || cam.width / (cam.zoom || 1)
  const h = v.height || cam.height / (cam.zoom || 1)
  const l = v.width ? v.x : cam.scrollX
  const t = v.height ? v.y : cam.scrollY

  const flakes: SnowFlake[] = []
  for (let i = 0; i < SNOW_COUNT; i++) {
    const img = scene.add.image(rand(l, l + w), rand(t, t + h), SNOW_FLAKE_KEY)
    img.setDepth(FX_DEPTH - 1)
    img.setAlpha(rand(0.45, 0.9))
    img.setScale(rand(0.4, 1.0))
    flakes.push({ img, speed: rand(40, 100), wind: rand(-15, 15) })
  }

  return { flakes, lastTime: 0, dn, scene, cam }
}

export function updateSnow(
  fx: ReturnType<typeof createSnow> & { lastTime: number; scene: Phaser.Scene; cam: Phaser.Cameras.Scene2D.Camera },
  now: number,
): void {
  if (fx.lastTime === 0) { fx.lastTime = now; return }
  const dt = Math.min(now - fx.lastTime, 200) / 1000
  fx.lastTime = now

  const v = fx.cam.worldView
  const l = v.x
  const r = v.x + v.width
  const t = v.y
  const b = v.y + v.height

  for (const f of fx.flakes) {
    f.img.y += f.speed * dt
    f.img.x += f.wind * dt
    if (f.img.y > b + 16) { f.img.y = t - 16; f.img.x = rand(l, r) }
    if (f.img.x < l - 16) f.img.x = r + 16
    else if (f.img.x > r + 16) f.img.x = l - 16
  }
}

export function destroySnow(fx: ReturnType<typeof createSnow>): void {
  for (const f of fx.flakes) f.img.destroy()
  if (fx.dn) {
    fx.dn.mask.img.destroy()
    fx.dn.glow.destroy()
  }
}
