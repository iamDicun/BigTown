import Phaser from 'phaser'

import type { GameSceneData } from './BootScene'
import { gameSceneKey } from './GameScene'

export const preloadSceneKey = 'preload'

export class PreloadScene extends Phaser.Scene {
  private sceneData!: GameSceneData

  constructor() {
    super(preloadSceneKey)
  }

  init(data: GameSceneData) {
    this.sceneData = data
  }

  preload() {
    const bootstrap = this.sceneData.bootstrap

    this.load.tilemapTiledJSON('map', `/assets/${bootstrap.tilemap_asset_key}`)

    // Không hardcode danh sách tileset — parse từ tileset_asset_key mà bootstrap trả về, khớp
    // đúng file .png đã copy vào frontend/public/assets/tiles/ (xem docs/Architecture.md mục 9.1).
    for (const tilesetName of bootstrap.tileset_asset_key.split(',')) {
      this.load.image(tilesetName, `/assets/tiles/${tilesetName}.png`)
    }

    this.load.spritesheet('player', '/assets/player/Player.png', { frameWidth: 32, frameHeight: 32 })
  }

  create() {
    this.scene.start(gameSceneKey, this.sceneData)
  }
}
