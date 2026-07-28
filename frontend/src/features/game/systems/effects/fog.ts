import type Phaser from 'phaser'
import {
  type MaskSpec, type CloudSpec, type FogCloud,
  makeMaskLayer, syncMaskLayer, FX_DEPTH,
  ensureCloudTexture,
} from './common'

// --- Tinh chỉnh --------------------------------------------------------------
const FOG_MASK: MaskSpec = { key: 'fx_fog_mask', hole: 120, falloff: 400, alpha: 0.97, rgb: '12,16,26' }

const FOG_CLOUDS: CloudSpec[] = [
  { key: 'fx_fog_low', rgb: '140,148,162', lo: 0.06, hi: 0.76, maxA: 0.50, tileScale: 3.4, vx: 4, vy: 1.2, par: 0.6, depth: FX_DEPTH - 2 },
  { key: 'fx_fog_mid', rgb: '170,178,190', lo: 0.20, hi: 0.84, maxA: 0.32, tileScale: 2.1, vx: -8, vy: 2.2, par: 0.4, depth: FX_DEPTH - 1 },
]
// -----------------------------------------------------------------------------

function makeFogCloud(scene: Phaser.Scene, cam: Phaser.Cameras.Scene2D.Camera, spec: CloudSpec): FogCloud {
  ensureCloudTexture(scene, spec.key, spec.rgb, spec.lo, spec.hi, spec.maxA)
  const ts = scene.add.tileSprite(0, 0, cam.width, cam.height, spec.key)
  ts.setOrigin(0, 0)
  ts.setScrollFactor(0)
  ts.setDepth(spec.depth)
  ts.setTileScale(spec.tileScale)
  return { ts, vx: spec.vx, vy: spec.vy, par: spec.par, driftX: Math.random() * 1000, driftY: Math.random() * 1000 }
}

// ============================================================================
// Fog (dark_village)
// ============================================================================

export function createFog(
  scene: Phaser.Scene,
  cam: Phaser.Cameras.Scene2D.Camera,
) {
  const mask = makeMaskLayer(scene, cam, FOG_MASK)
  const clouds = FOG_CLOUDS.map((spec) => makeFogCloud(scene, cam, spec))
  return { mask, clouds, lastTime: 0, scene, cam }
}

export function updateFog(
  fx: ReturnType<typeof createFog> & { lastTime: number; scene: Phaser.Scene; cam: Phaser.Cameras.Scene2D.Camera },
  now: number,
  focus: { x: number; y: number },
): void {
  if (fx.lastTime === 0) fx.lastTime = now
  const dt = Math.min(now - fx.lastTime, 200) / 1000
  fx.lastTime = now

  syncMaskLayer(fx.scene, fx.mask, fx.cam)
  fx.mask.img.setPosition(focus.x, focus.y)

  for (let i = 0; i < fx.clouds.length; i++) {
    const c = fx.clouds[i]
    if (c.ts.width !== fx.cam.width || c.ts.height !== fx.cam.height) {
      c.ts.setSize(fx.cam.width, fx.cam.height)
    }
    c.driftX += c.vx * dt
    c.driftY += c.vy * dt
    c.ts.tilePositionX = fx.cam.scrollX * c.par + c.driftX
    c.ts.tilePositionY = fx.cam.scrollY * c.par + c.driftY
    c.ts.setAlpha(1 + Math.sin(now * 0.0004 + i * 2.1) * 0.08)
  }
}

export function destroyFog(fx: ReturnType<typeof createFog>): void {
  fx.mask.img.destroy()
  for (const c of fx.clouds) c.ts.destroy()
}
