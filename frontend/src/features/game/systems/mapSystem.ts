import type Phaser from 'phaser'

import type { BootstrapDto } from '../services/realtime.service'

export const TILE_SIZE = 16

const TILE_LAYER_NAMES = ['Ground', 'DecorationBelow', 'Objects', 'DecorationAbove']

export type MapBuildResult = {
  map: Phaser.Tilemaps.Tilemap
  collisionGroup: Phaser.Physics.Arcade.StaticGroup
}

// Dựng tilemap + layer + collision group từ bootstrap (tilemap/tileset đã embed sẵn, xem
// asset/tools/embed_tilesets.js). Tách khỏi GameScene để dễ mở rộng map/layer sau này mà không
// phải sửa trực tiếp scene chính — xem docs/Phaser-Frontend-Guide.md mục 19.
export function buildMap(scene: Phaser.Scene, bootstrap: BootstrapDto): MapBuildResult {
  const map = scene.make.tilemap({ key: 'map' })

  const tilesetNames = bootstrap.tileset_asset_key.split(',')
  const tilesets = tilesetNames.map((name) => {
    const tileset = map.addTilesetImage(name, name)
    if (!tileset) throw new Error(`Tileset not found in map data: ${name}`)
    return tileset
  })

  for (const layerName of TILE_LAYER_NAMES) {
    map.createLayer(layerName, tilesets, 0, 0)
  }

  return { map, collisionGroup: buildCollisionGroup(scene, map) }
}

// Collision đọc từ object layer "Collision" (map thực tế build bằng asset/tools/generate_map.js
// dùng object layer, không dùng tile property như draft đầu Phaser-Frontend-Guide).
// Object layer "NPCSpawns" (animal/villager) là flavor/decoration, cố tình không đọc ở đây —
// enemy combat thật sẽ có spawn riêng ở phase sau, xem docs/Movement-Chat-Spawn-Plan.md mục I.
function buildCollisionGroup(scene: Phaser.Scene, map: Phaser.Tilemaps.Tilemap): Phaser.Physics.Arcade.StaticGroup {
  const staticGroup = scene.physics.add.staticGroup()

  const collisionLayer = map.getObjectLayer('Collision')
  if (!collisionLayer) return staticGroup

  for (const obj of collisionLayer.objects) {
    const width = obj.width ?? 0
    const height = obj.height ?? 0
    const centerX = (obj.x ?? 0) + width / 2
    const centerY = (obj.y ?? 0) + height / 2

    const zone = scene.add.zone(centerX, centerY, width, height)
    scene.physics.add.existing(zone, true)
    staticGroup.add(zone)
  }

  return staticGroup
}
