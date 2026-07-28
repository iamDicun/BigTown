import type Phaser from 'phaser'

// ============================================================================
// Shared types, constants, & helpers cho toàn bộ hiệu ứng môi trường
// ============================================================================

export const FX_DEPTH = 9000
export const TEX_SIZE = 1024
export const GLOW_TEX_SIZE = 256
export const LANTERN_GLOW_KEY = 'fx_lantern_glow'
export const LANTERN_GLOW_ALPHA = 0.5
export const BLEND_ADD = 1

export type MaskSpec = {
  key: string
  hole: number
  falloff: number
  alpha: number
  rgb: string
}

export type MaskLayer = {
  img: Phaser.GameObjects.Image
  spec: MaskSpec
  radius: number
  driftAmp: number
  driftSpeed: number
  phase: number
}

export type CloudSpec = {
  key: string
  rgb: string
  lo: number
  hi: number
  maxA: number
  tileScale: number
  vx: number
  vy: number
  par: number
  depth: number
}

export type FogCloud = {
  ts: Phaser.GameObjects.TileSprite
  vx: number
  vy: number
  par: number
  driftX: number
  driftY: number
}

export type SnowFlake = { img: Phaser.GameObjects.Image; speed: number; wind: number }

export type DayNightLayers = { mask: MaskLayer; glow: Phaser.GameObjects.Image }

export type Focus = { x: number; y: number }

// ============================================================================
// Time sync (real-clock → game time 0-12, 24h thật = 12 phút game)
// ============================================================================

const GAME_MINUTES_PER_HOUR = 120 / 60 // TODO: 1/60 để test 12 giây; trả về 120/60

let _syncRealMinute = 0
let _syncTimestamp = 0

export function syncWorldTime(): void {
  const d = new Date()
  _syncRealMinute = d.getHours() * 60 + d.getMinutes() + d.getSeconds() / 60
  _syncTimestamp = performance.now()
}

export function getGameTime(): number {
  const elapsedMin = (performance.now() - _syncTimestamp) / 60000
  return ((_syncRealMinute + elapsedMin) / GAME_MINUTES_PER_HOUR) % 12
}

export function getDarkness(): number {
  const gt = getGameTime()
  return 0.5 - 0.5 * Math.cos((gt / 12) * Math.PI * 2)
}

// ============================================================================
// Radial gradient mask
// ============================================================================

export function rand(min: number, max: number): number {
  return Math.random() * (max - min) + min
}

export function coverRadius(cam: Phaser.Cameras.Scene2D.Camera): number {
  const zoom = cam.zoom || 1
  const w = cam.worldView.width || cam.width / zoom
  const h = cam.worldView.height || cam.height / zoom
  return Math.ceil(Math.hypot(w, h))
}

export function ensureMaskTexture(
  scene: Phaser.Scene,
  spec: MaskSpec,
  wantRadius: number,
): { key: string; radius: number } {
  const radius = Math.max(spec.hole + spec.falloff + 32, Math.ceil(wantRadius / 128) * 128)
  const key = `${spec.key}@${radius}`
  if (scene.textures.exists(key)) return { key, radius }

  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = TEX_SIZE
  const ctx = canvas.getContext('2d')!
  const c = TEX_SIZE / 2

  const r0 = (spec.hole / radius) * c
  const r1 = c
  const solidAt = Math.min(0.995, spec.falloff / (radius - spec.hole))

  const g = ctx.createRadialGradient(c, c, r0, c, c, r1)
  g.addColorStop(0, `rgba(${spec.rgb},0)`)
  g.addColorStop(solidAt, `rgba(${spec.rgb},${spec.alpha})`)
  g.addColorStop(1, `rgba(${spec.rgb},${spec.alpha})`)
  ctx.fillStyle = g
  ctx.fillRect(0, 0, TEX_SIZE, TEX_SIZE)

  scene.textures.addCanvas(key, canvas)
  return { key, radius }
}

export function makeMaskLayer(
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

export function syncMaskLayer(scene: Phaser.Scene, layer: MaskLayer, cam: Phaser.Cameras.Scene2D.Camera): void {
  const want = coverRadius(cam) + layer.driftAmp
  if (want <= layer.radius && want > layer.radius * 0.6) return
  const next = ensureMaskTexture(scene, layer.spec, want)
  layer.radius = next.radius
  layer.img.setTexture(next.key)
  layer.img.setScale(next.radius / (TEX_SIZE / 2))
}

// ============================================================================
// Noise fBm tileable texture (cho sương mù cuộn)
// ============================================================================

function mulberry32(a: number): () => number {
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

export function ensureCloudTexture(
  scene: Phaser.Scene,
  key: string,
  rgb: string,
  lo: number,
  hi: number,
  maxA: number,
): void {
  if (scene.textures.exists(key)) return

  const SIZE = 256
  const CELLS = 8
  const rnd = mulberry32(1337)
  const grid = Array.from({ length: CELLS * CELLS }, () => rnd())
  const val = (x: number, y: number): number =>
    grid[(((y % CELLS) + CELLS) % CELLS) * CELLS + (((x % CELLS) + CELLS) % CELLS)]
  const smooth = (t: number): number => t * t * (3 - 2 * t)
  const noise = (x: number, y: number): number => {
    const xi = Math.floor(x)
    const yi = Math.floor(y)
    const xf = x - xi
    const yf = y - yi
    const u = smooth(xf)
    const w = smooth(yf)
    const a = val(xi, yi)
    const b = val(xi + 1, yi)
    const cc = val(xi, yi + 1)
    const d = val(xi + 1, yi + 1)
    return (a * (1 - u) + b * u) * (1 - w) + (cc * (1 - u) + d * u) * w
  }

  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = SIZE
  const ctx = canvas.getContext('2d')!
  const imgData = ctx.createImageData(SIZE, SIZE)
  const [r, g, b] = rgb.split(',').map(Number)

  for (let y = 0; y < SIZE; y++) {
    for (let x = 0; x < SIZE; x++) {
      let n = 0
      let amp = 0.6
      let freq = CELLS / SIZE
      for (let o = 0; o < 3; o++) {
        n += noise(x * freq, y * freq) * amp
        amp *= 0.5
        freq *= 2
      }
      const a = Math.max(0, Math.min(1, (n - lo) / (hi - lo))) * maxA
      const i = (y * SIZE + x) * 4
      imgData.data[i] = r
      imgData.data[i + 1] = g
      imgData.data[i + 2] = b
      imgData.data[i + 3] = a * 255
    }
  }

  ctx.putImageData(imgData, 0, 0)
  scene.textures.addCanvas(key, canvas)
}

// ============================================================================
// Glow texture (lantern)
// ============================================================================

export function ensureGlowTexture(scene: Phaser.Scene): void {
  if (scene.textures.exists(LANTERN_GLOW_KEY)) return
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = GLOW_TEX_SIZE
  const ctx = canvas.getContext('2d')!
  const c = GLOW_TEX_SIZE / 2
  const g = ctx.createRadialGradient(c, c, 0, c, c, c)
  g.addColorStop(0, 'rgba(255,200,130,0.55)')
  g.addColorStop(0.45, 'rgba(255,160,80,0.22)')
  g.addColorStop(1, 'rgba(255,140,50,0)')
  ctx.fillStyle = g
  ctx.fillRect(0, 0, GLOW_TEX_SIZE, GLOW_TEX_SIZE)
  scene.textures.addCanvas(LANTERN_GLOW_KEY, canvas)
}

// ============================================================================
// Snow flake texture
// ============================================================================

export const SNOW_FLAKE_KEY = 'fx_snow_flake'

export function ensureSnowFlakeTexture(scene: Phaser.Scene): void {
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
