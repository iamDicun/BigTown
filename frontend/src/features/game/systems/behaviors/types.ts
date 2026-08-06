import type Phaser from 'phaser'

export interface ExtraCollider {
  dx: number
  dy: number
  w: number
  h: number
}

export interface ItemMeta {
  w: number
  h: number
  anchorX: number
  anchorY: number
  collides: boolean
  collision_w?: number
  collision_h?: number
  collision_x?: number
  collision_y?: number
  collision_override?: boolean
  frameWidth?: number
  frameHeight?: number
  frame?: number
  behaviors?: string[]
  bridge_zones?: ExtraCollider[]
  bridge_zones_h?: ExtraCollider[]
  extra_colliders?: ExtraCollider[]
  [key: string]: unknown
}

export interface BehaviorContext {
  scene: Phaser.Scene
  collisionGroup: Phaser.Physics.Arcade.StaticGroup
  tileSize: number
}

export interface UpdateState {
  player: Phaser.GameObjects.Sprite
  playerBounds: Phaser.Geom.Rectangle
  darkness: number
  flicker: number
  sceneTime: number
  isBehindDecoration: boolean
  isOnBridge: boolean
}

export interface BehaviorHandler {
  onCreate?(sprite: Phaser.GameObjects.Image, meta: ItemMeta, ctx: BehaviorContext): void
  onUpdate?(sprite: Phaser.GameObjects.Image, meta: ItemMeta, state: UpdateState): void
  onDestroy?(sprite: Phaser.GameObjects.Image): void
}
