import type Phaser from 'phaser'

import type { BootstrapDto } from '../services/realtime.service'

export const TILE_SIZE = 16

const ABOVE_LAYER_NAME = 'DecorationAbove'
const DEFAULT_TILE_LAYER_NAMES = ['Ground', 'DecorationBelow', 'Objects', ABOVE_LAYER_NAME]

function resolveTileLayerNames(bootstrap: BootstrapDto): string[] {
  if (bootstrap.layer_names && bootstrap.layer_names.length > 0) {
    return bootstrap.layer_names
  }
  return DEFAULT_TILE_LAYER_NAMES
}

function resolveAboveLayerName(bootstrap: BootstrapDto): string {
  return bootstrap.above_layer_name || ABOVE_LAYER_NAME
}

function resolveCollisionLayerName(bootstrap: BootstrapDto): string {
  return bootstrap.collision_layer_name || 'Collision'
}

export type WarpZone = {
  zone: Phaser.GameObjects.Zone
  destMap: string
  destX: number
  destY: number
}

export type MapBuildResult = {
  map: Phaser.Tilemaps.Tilemap
  collisionGroup: Phaser.Physics.Arcade.StaticGroup
  aboveLayer: Phaser.Tilemaps.TilemapLayerBase | null
  warpZones: WarpZone[]
}

export function buildMap(scene: Phaser.Scene, bootstrap: BootstrapDto): MapBuildResult {
  const map = scene.make.tilemap({ key: 'map' })

  const tilesetNames = bootstrap.tileset_asset_key.split(',')
  const tilesets = tilesetNames.map((name) => {
    const tileset = map.addTilesetImage(name, name)
    if (!tileset) throw new Error(`Tileset not found in map data: ${name}`)
    return tileset
  })

  const layerNames = resolveTileLayerNames(bootstrap)
  const aboveLayerName = resolveAboveLayerName(bootstrap)
  const collisionLayerName = resolveCollisionLayerName(bootstrap)

  let aboveLayer: Phaser.Tilemaps.TilemapLayerBase | null = null
  let collisionTilesLayer: Phaser.Tilemaps.TilemapLayerBase | null = null

  for (const layerName of layerNames) {
    const layer = map.createLayer(layerName, tilesets, 0, 0)
    if (!layer) continue

    // Adjust Y coordinates for tiles with heights larger than map's tileHeight
    layer.forEachTile((tile: any) => {
      if (tile && tile.index > 0 && tile.tileset) {
        const heightDiff = tile.tileset.tileHeight - map.tileHeight
        if (heightDiff > 0) {
          tile.pixelY -= heightDiff
        }
      }
    })

    if (layerName === collisionLayerName) {
      collisionTilesLayer = layer
      layer.setVisible(false)
      continue
    }
    if (layerName === aboveLayerName) aboveLayer = layer
  }

  if (aboveLayer) {
    aboveLayer.setDepth(10)
  }

  const collisionGroup = buildCollisionGroup(scene, map, collisionLayerName, collisionTilesLayer)
  const warpZones = buildWarpZones(scene, map)

  return { map, collisionGroup, aboveLayer, warpZones }
}

export const PLAYER_DEPTH = 3

function buildCollisionGroup(
  scene: Phaser.Scene,
  map: Phaser.Tilemaps.Tilemap,
  collisionLayerName: string,
  collisionTilesLayer: Phaser.Tilemaps.TilemapLayerBase | null,
): Phaser.Physics.Arcade.StaticGroup {
  const staticGroup = scene.physics.add.staticGroup()

  buildCollisionFromObjectLayer(staticGroup, scene, map, collisionLayerName)

  if (collisionTilesLayer) {
    buildCollisionFromRenderedLayer(staticGroup, scene, collisionTilesLayer)
  }

  return staticGroup
}

function buildCollisionFromObjectLayer(
  staticGroup: Phaser.Physics.Arcade.StaticGroup,
  scene: Phaser.Scene,
  map: Phaser.Tilemaps.Tilemap,
  layerName: string,
): void {
  const collisionLayer = map.getObjectLayer(layerName)
  if (!collisionLayer) return

  for (const obj of collisionLayer.objects) {
    const width = obj.width ?? 0
    const height = obj.height ?? 0
    if (width <= 0 || height <= 0) continue

    const centerX = (obj.x ?? 0) + width / 2
    const centerY = (obj.y ?? 0) + height / 2

    const zone = scene.add.zone(centerX, centerY, width, height)
    scene.physics.add.existing(zone, true)
    staticGroup.add(zone)
  }
}

function buildCollisionFromRenderedLayer(
  staticGroup: Phaser.Physics.Arcade.StaticGroup,
  scene: Phaser.Scene,
  layer: Phaser.Tilemaps.TilemapLayerBase,
): void {
  layer.forEachTile((tile: Phaser.Tilemaps.Tile) => {
    if (!tile || tile.index < 0) return
    if (!isCollidableTile(tile)) return

    const zone = scene.add.zone(tile.getCenterX(), tile.getCenterY(), tile.width, tile.height)
    scene.physics.add.existing(zone, true)
    staticGroup.add(zone)
  })
}

function isCollidableTile(tile: Phaser.Tilemaps.Tile): boolean {
  if (!tile.properties) return true
  if (!('collides' in tile.properties)) return true
  return isTruthy(tile.properties.collides)
}

function isTruthy(value: unknown): boolean {
  if (value === undefined || value === null) return false
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') return value.toLowerCase() !== 'false' && value !== '0' && value !== ''
  return true
}

function buildWarpZones(scene: Phaser.Scene, map: Phaser.Tilemaps.Tilemap): WarpZone[] {
  const warps: WarpZone[] = []
  for (const layer of map.objects) {
    if (!layer?.objects) continue
    for (const obj of layer.objects) {
      if (obj.type !== 'warp') continue
      const destMap = getProperty(obj, 'dest_map')
      const destX = parseInt(getProperty(obj, 'dest_x') || '0', 10)
      const destY = parseInt(getProperty(obj, 'dest_y') || '0', 10)
      if (!destMap) continue

      const width = obj.width ?? 32
      const height = obj.height ?? 32
      const centerX = (obj.x ?? 0) + width / 2
      const centerY = (obj.y ?? 0) + height / 2

      const zone = scene.add.zone(centerX, centerY, width, height)
      scene.physics.add.existing(zone, true)
      warps.push({ zone, destMap, destX, destY })
    }
  }
  return warps
}

function getProperty(obj: any, name: string): string | undefined {
  for (const prop of obj.properties ?? []) {
    if (prop.name === name) return prop.value?.toString()
  }
  return undefined
}
