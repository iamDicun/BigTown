import type Phaser from 'phaser'

const MASK_RADIUS = 72
const FADE_ALPHA_CENTER = 0.15
const FADE_DURATION_MS = 120
const PROBE_PADDING = { x: 4, top: 16, bottom: 0 }
const PLAYER_FADE_ALPHA = 0.3
const UPDATE_THROTTLE_MS = 40

export type AboveLayerFade = {
  layer: Phaser.Tilemaps.TilemapLayerBase
  fadedTiles: Map<string, Phaser.Tilemaps.Tile>
  tileAlpha: Map<string, number>
  lastUpdateTime: number
}

export function createAboveLayerFade(
  _scene: Phaser.Scene,
  layer: Phaser.Tilemaps.TilemapLayerBase,
): AboveLayerFade {
  return { layer, fadedTiles: new Map(), tileAlpha: new Map(), lastUpdateTime: 0 }
}

export function updateAboveLayerFade(
  scene: Phaser.Scene,
  fade: AboveLayerFade,
  sprite: Phaser.GameObjects.Sprite,
  time: number,
  underPlacement: boolean,
): void {
  if (time - fade.lastUpdateTime < UPDATE_THROTTLE_MS) return
  fade.lastUpdateTime = time

  const cx = sprite.x
  const cy = sprite.y
  const probeSize = MASK_RADIUS * 2

  const tiles = fade.layer.getTilesWithinWorldXY(
    cx - MASK_RADIUS, cy - MASK_RADIUS, probeSize, probeSize,
  )
  const activeKeys = new Set<string>()

  for (const tile of tiles) {
    if (!tile || tile.index <= 0) continue

    const key = tileKey(tile)
    const dist = Math.hypot(tile.getCenterX() - cx, tile.getCenterY() - cy)
    if (dist > MASK_RADIUS) continue

    activeKeys.add(key)
    const t = dist / MASK_RADIUS
    const eased = t * t
    const target = FADE_ALPHA_CENTER + eased * (1 - FADE_ALPHA_CENTER)
    const rounded = Math.round(target * 100) / 100

    if (fade.tileAlpha.get(key) === rounded) continue

    fade.tileAlpha.set(key, rounded)
    fade.fadedTiles.set(key, tile)
    scene.tweens.killTweensOf(tile)
    scene.tweens.add({ targets: tile, alpha: rounded, duration: FADE_DURATION_MS })
  }

  for (const [key, tile] of fade.fadedTiles) {
    if (activeKeys.has(key)) continue
    fade.tileAlpha.delete(key)
    scene.tweens.killTweensOf(tile)
    scene.tweens.add({ targets: tile, alpha: 1, duration: FADE_DURATION_MS })
    fade.fadedTiles.delete(key)
  }

  const bounds = sprite.getBounds()
  const probeX = bounds.x - PROBE_PADDING.x
  const probeY = bounds.y - PROBE_PADDING.top
  const probeW = bounds.width + PROBE_PADDING.x * 2
  const probeH = bounds.height + PROBE_PADDING.top + PROBE_PADDING.bottom

  const underTiles = fade.layer.getTilesWithinWorldXY(probeX, probeY, probeW, probeH)
  const underCanopy = underTiles.some((t) => t && t.index > 0)

  const isCovered = underCanopy || underPlacement

  if (isCovered && sprite.alpha !== PLAYER_FADE_ALPHA) {
    scene.tweens.killTweensOf(sprite)
    scene.tweens.add({ targets: sprite, alpha: PLAYER_FADE_ALPHA, duration: FADE_DURATION_MS })
  } else if (!isCovered && sprite.alpha !== 1) {
    scene.tweens.killTweensOf(sprite)
    scene.tweens.add({ targets: sprite, alpha: 1, duration: FADE_DURATION_MS })
  }
}

function tileKey(tile: Phaser.Tilemaps.Tile): string {
  return `${tile.x}_${tile.y}`
}
