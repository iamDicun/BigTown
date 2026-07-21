import type Phaser from 'phaser'

// Mờ đúng những tile DecorationAbove đang che local player (tán cây, mái nhà...) để thấy nhân vật
// bên dưới, thay vì mờ nguyên cả layer. Chỉ tác động tile thật sự overlap bounding box player —
// tile không liên quan giữ nguyên alpha 1. Xem mapSystem.ts (aboveLayer) và
// docs/Phaser-Frontend-Guide.md mục 19 (tách system riêng khi thêm tính năng mới).
const FADE_ALPHA = 0.35
const FADE_DURATION_MS = 150

// Mở rộng bounding box player một chút, đặc biệt phía trên (canopy thường vẽ nhô cao hơn vị trí
// đứng của player) để bắt đúng tile đang che, không chỉ tile trùng khít pixel-perfect.
const PROBE_PADDING = { x: 4, top: 16, bottom: 0 }

export type AboveLayerFade = {
  layer: Phaser.Tilemaps.TilemapLayerBase
  fadedTiles: Map<string, Phaser.Tilemaps.Tile>
}

export function createAboveLayerFade(layer: Phaser.Tilemaps.TilemapLayerBase): AboveLayerFade {
  return { layer, fadedTiles: new Map() }
}

// Gọi mỗi frame từ GameScene.update(). Idempotent — tile đang mờ đúng vị trí sẽ không bị tween lại.
export function updateAboveLayerFade(
  scene: Phaser.Scene,
  fade: AboveLayerFade,
  sprite: Phaser.GameObjects.Sprite,
): void {
  const bounds = sprite.getBounds()
  const probeX = bounds.x - PROBE_PADDING.x
  const probeY = bounds.y - PROBE_PADDING.top
  const probeWidth = bounds.width + PROBE_PADDING.x * 2
  const probeHeight = bounds.height + PROBE_PADDING.top + PROBE_PADDING.bottom

  const overlapping = fade.layer.getTilesWithinWorldXY(probeX, probeY, probeWidth, probeHeight)
  const overlappingKeys = new Set<string>()

  for (const tile of overlapping) {
    const key = tileKey(tile)
    overlappingKeys.add(key)

    if (!fade.fadedTiles.has(key)) {
      fade.fadedTiles.set(key, tile)
      scene.tweens.killTweensOf(tile)
      scene.tweens.add({ targets: tile, alpha: FADE_ALPHA, duration: FADE_DURATION_MS })
    }
  }

  for (const [key, tile] of fade.fadedTiles) {
    if (overlappingKeys.has(key)) continue

    scene.tweens.killTweensOf(tile)
    scene.tweens.add({ targets: tile, alpha: 1, duration: FADE_DURATION_MS })
    fade.fadedTiles.delete(key)
  }
}

function tileKey(tile: Phaser.Tilemaps.Tile): string {
  return `${tile.x}_${tile.y}`
}
