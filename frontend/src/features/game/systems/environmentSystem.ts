import type Phaser from 'phaser'

// ============================================================================
// Environment FX
//   dark_village      -> fog (1 lớp mask bám nhân vật + 2 lớp mây trôi)
//   winter            -> snow
//   village_adventure -> day/night + vòng sáng đèn lồng quanh nhân vật
//
// Khác biệt cốt lõi so với bản cũ:
//  1) Overlay đặt trong WORLD SPACE và bám theo NHÂN VẬT, không phải tâm camera
//     -> lỗ sáng luôn đúng chỗ dù camera lerp hay bị clamp ở biên map.
//  2) hole / falloff tính bằng pixel THẬT rồi mới suy ra scale, thay vì
//     setScale(2~3) lên texture cố định (bản cũ scale làm gần như cả màn hình
//     rơi vào vùng trong suốt của gradient -> không thấy gì).
//  3) Depth cao hơn mọi tile layer.
//  4) Tự co giãn theo kích thước viewport / zoom.
// ============================================================================

const FX_DEPTH = 9000 // phải > depth của above layer & player
const TEX_SIZE = 1024 // canvas của mask (chỉ tạo 1 lần / bán kính)
const GLOW_TEX_SIZE = 256

type MaskSpec = {
  key: string
  hole: number // bán kính vùng nhìn rõ (pixel world)
  falloff: number // độ dài dải chuyển trong suốt -> đặc
  alpha: number // alpha tối đa ở rìa
  rgb: string // màu overlay, ví dụ '10,14,22'
}

// --- Tinh chỉnh cảm giác của effect ở đây -----------------------------------
const FOG_MASK: MaskSpec = { key: 'fx_fog_mask', hole: 130, falloff: 380, alpha: 0.96, rgb: '10,14,22' }
const FOG_CLOUD: MaskSpec = { key: 'fx_fog_cloud', hole: 50, falloff: 600, alpha: 0.45, rgb: '160,175,195' }
const NIGHT_MASK: MaskSpec = { key: 'fx_night_mask', hole: 120, falloff: 250, alpha: 0.94, rgb: '4,9,28' }

const FOG_CLOUD_DRIFT = 120 // biên độ trôi của lớp mây (px)
const SNOW_FLAKE_KEY = 'fx_snow_flake'
const SNOW_COUNT = 80
const DAYNIGHT_CYCLE_MS = 720_000 // 12 phút
const LANTERN_GLOW_KEY = 'fx_lantern_glow'
const LANTERN_GLOW_ALPHA = 0.5
const BLEND_ADD = 1 // = Phaser.BlendModes.ADD (không import runtime Phaser)
// ---------------------------------------------------------------------------

type MaskLayer = {
  img: Phaser.GameObjects.Image
  spec: MaskSpec
  radius: number
  driftAmp: number
  driftSpeed: number
  phase: number
}

type SnowFlake = { img: Phaser.GameObjects.Image; speed: number; wind: number }

type Focus = { x: number; y: number }

export type EnvironmentFx =
  | { type: 'fog'; scene: Phaser.Scene; cam: Phaser.Cameras.Scene2D.Camera; layers: MaskLayer[] }
  | { type: 'snow'; scene: Phaser.Scene; cam: Phaser.Cameras.Scene2D.Camera; flakes: SnowFlake[]; lastTime: number }
  | {
      type: 'dayNight'
      scene: Phaser.Scene
      cam: Phaser.Cameras.Scene2D.Camera
      mask: MaskLayer
      glow: Phaser.GameObjects.Image
    }
  | { type: 'none' }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function rand(min: number, max: number): number {
  return Math.random() * (max - min) + min
}

/** Bán kính cần thiết để 1 ảnh tâm-ở-nhân-vật che kín toàn bộ vùng nhìn thấy. */
function coverRadius(cam: Phaser.Cameras.Scene2D.Camera): number {
  const zoom = cam.zoom || 1
  const w = cam.worldView.width || cam.width / zoom
  const h = cam.worldView.height || cam.height / zoom
  return Math.ceil(Math.hypot(w, h)) // worst case: nhân vật nằm ở góc view
}

/**
 * Tạo (và cache) texture gradient: trong suốt ở tâm -> đặc dần ra ngoài.
 * Trả về key + bán kính thực tế mà ảnh sẽ phủ khi vẽ ở scale tương ứng.
 */
function ensureMaskTexture(
  scene: Phaser.Scene,
  spec: MaskSpec,
  wantRadius: number,
): { key: string; radius: number } {
  // làm tròn để không sinh vô số texture khi resize
  const radius = Math.max(spec.hole + spec.falloff + 32, Math.ceil(wantRadius / 128) * 128)
  const key = `${spec.key}@${radius}`
  if (scene.textures.exists(key)) return { key, radius }

  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = TEX_SIZE
  const ctx = canvas.getContext('2d')!
  const c = TEX_SIZE / 2

  // quy đổi pixel world -> pixel texture
  const r0 = (spec.hole / radius) * c
  const r1 = c
  // offset của điểm đạt alpha tối đa, tính trong khoảng [r0, r1]
  const solidAt = Math.min(0.995, spec.falloff / (radius - spec.hole))

  const g = ctx.createRadialGradient(c, c, r0, c, c, r1)
  g.addColorStop(0, `rgba(${spec.rgb},0)`)
  g.addColorStop(solidAt, `rgba(${spec.rgb},${spec.alpha})`)
  g.addColorStop(1, `rgba(${spec.rgb},${spec.alpha})`)
  ctx.fillStyle = g
  // fill cả canvas: 4 góc ngoài r1 sẽ lấy màu của stop cuối -> luôn kín
  ctx.fillRect(0, 0, TEX_SIZE, TEX_SIZE)

  scene.textures.addCanvas(key, canvas)
  return { key, radius }
}

function makeMaskLayer(
  scene: Phaser.Scene,
  cam: Phaser.Cameras.Scene2D.Camera,
  spec: MaskSpec,
  driftAmp = 0,
  driftSpeed = 1,
): MaskLayer {
  const { key, radius } = ensureMaskTexture(scene, spec, coverRadius(cam) + driftAmp)
  const img = scene.add.image(cam.scrollX, cam.scrollY, key)
  img.setDepth(FX_DEPTH)
  img.setScale(radius / (TEX_SIZE / 2))
  return { img, spec, radius, driftAmp, driftSpeed, phase: Math.random() * Math.PI * 2 }
}

/** Viewport đổi kích thước / zoom -> đổi texture cho khớp. */
function syncMaskLayer(scene: Phaser.Scene, layer: MaskLayer, cam: Phaser.Cameras.Scene2D.Camera): void {
  const want = coverRadius(cam) + layer.driftAmp
  if (want <= layer.radius && want > layer.radius * 0.6) return
  const next = ensureMaskTexture(scene, layer.spec, want)
  layer.radius = next.radius
  layer.img.setTexture(next.key)
  layer.img.setScale(next.radius / (TEX_SIZE / 2))
}

function ensureGlowTexture(scene: Phaser.Scene): void {
  if (scene.textures.exists(LANTERN_GLOW_KEY)) return
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = GLOW_TEX_SIZE
  const ctx = canvas.getContext('2d')!
  const c = GLOW_TEX_SIZE / 2
  const g = ctx.createRadialGradient(c, c, 0, c, c, c)
  g.addColorStop(0, 'rgba(255,214,150,0.95)')
  g.addColorStop(0.45, 'rgba(255,180,90,0.35)')
  g.addColorStop(1, 'rgba(255,150,60,0)')
  ctx.fillStyle = g
  ctx.fillRect(0, 0, GLOW_TEX_SIZE, GLOW_TEX_SIZE)
  scene.textures.addCanvas(LANTERN_GLOW_KEY, canvas)
}

function ensureSnowFlakeTexture(scene: Phaser.Scene): void {
  if (scene.textures.exists(SNOW_FLAKE_KEY)) return
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = 10
  const ctx = canvas.getContext('2d')!
  const c = 5
  const g = ctx.createRadialGradient(c, c, 0, c, c, c)
  g.addColorStop(0, 'rgba(255,255,255,0.95)')
  g.addColorStop(0.5, 'rgba(230,240,255,0.6)')
  g.addColorStop(1, 'rgba(200,220,255,0)')
  ctx.fillStyle = g
  ctx.fillRect(0, 0, 10, 10)
  scene.textures.addCanvas(SNOW_FLAKE_KEY, canvas)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

export function createEnvironmentFx(scene: Phaser.Scene, mapCode: string): EnvironmentFx {
  const cam = scene.cameras.main

  switch (mapCode) {
    case 'dark_village': {
      const layers = [
        makeMaskLayer(scene, cam, FOG_MASK), // lớp chính: khoá cứng vào nhân vật
        makeMaskLayer(scene, cam, FOG_CLOUD, FOG_CLOUD_DRIFT, 1),
        makeMaskLayer(scene, cam, FOG_CLOUD, FOG_CLOUD_DRIFT * 0.7, -0.6),
      ]
      return { type: 'fog', scene, cam, layers }
    }

    case 'winter': {
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
      return { type: 'snow', scene, cam, flakes, lastTime: 0 }
    }

    case 'village_adventure': {
      const mask = makeMaskLayer(scene, cam, NIGHT_MASK)
      mask.img.setAlpha(0)

      ensureGlowTexture(scene)
      const glow = scene.add.image(cam.scrollX, cam.scrollY, LANTERN_GLOW_KEY)
      glow.setDepth(FX_DEPTH + 1) // vẽ trên mask
      glow.setBlendMode(BLEND_ADD)
      glow.setScale((NIGHT_MASK.hole * 2.4) / GLOW_TEX_SIZE)
      glow.setAlpha(0)

      return { type: 'dayNight', scene, cam, mask, glow }
    }

    default:
      console.warn('[environmentFx] map_code không khớp effect nào:', mapCode)
      return { type: 'none' }
  }
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

/** `target` nên là sprite của local player. Thiếu thì fallback về tâm camera. */
export function updateEnvironmentFx(fx: EnvironmentFx, now: number, target?: Focus | null): void {
  if (fx.type === 'none') return

  const v = fx.cam.worldView
  const focus: Focus = target ?? { x: v.centerX, y: v.centerY }

  switch (fx.type) {
    case 'fog': {
      for (const layer of fx.layers) {
        syncMaskLayer(fx.scene, layer, fx.cam)
        const t = now * 0.00018 * layer.driftSpeed
        const ox = Math.sin(t + layer.phase) * layer.driftAmp
        const oy = Math.cos(t * 0.8 + layer.phase) * layer.driftAmp * 0.6
        layer.img.setPosition(focus.x + ox, focus.y + oy)
      }
      break
    }

    case 'snow': {
      if (fx.lastTime === 0) {
        fx.lastTime = now
        return
      }
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

        if (f.img.y > b + 16) {
          f.img.y = t - 16
          f.img.x = rand(l, r)
        }
        if (f.img.x < l - 16) f.img.x = r + 16
        else if (f.img.x > r + 16) f.img.x = l - 16
      }
      break
    }

    case 'dayNight': {
      const t = (now % DAYNIGHT_CYCLE_MS) / DAYNIGHT_CYCLE_MS
      const darkness = 0.5 - 0.5 * Math.cos(t * Math.PI * 2) // 0 = trưa, 1 = nửa đêm

      syncMaskLayer(fx.scene, fx.mask, fx.cam)
      fx.mask.img.setPosition(focus.x, focus.y)
      fx.mask.img.setAlpha(darkness)
      fx.mask.img.setVisible(darkness > 0.02)

      // đèn lồng: nhấp nháy nhẹ cho có sinh khí
      const flicker = 0.92 + Math.sin(now * 0.008) * 0.05 + Math.sin(now * 0.021) * 0.03
      fx.glow.setPosition(focus.x, focus.y)
      fx.glow.setAlpha(darkness * LANTERN_GLOW_ALPHA * flicker)
      fx.glow.setVisible(darkness > 0.05)
      break
    }
  }
}

// ---------------------------------------------------------------------------
// Destroy — gọi trong scene shutdown / trước khi warp sang map khác
// ---------------------------------------------------------------------------

export function destroyEnvironmentFx(fx: EnvironmentFx): EnvironmentFx {
  switch (fx.type) {
    case 'fog':
      for (const l of fx.layers) l.img.destroy()
      break
    case 'snow':
      for (const f of fx.flakes) f.img.destroy()
      break
    case 'dayNight':
      fx.mask.img.destroy()
      fx.glow.destroy()
      break
  }
  return { type: 'none' }
}
