import Phaser from 'phaser'

import type { GameSceneData } from './BootScene'
import { gameSceneKey } from './GameScene'
import { getCharacterSpriteUrl } from '../services/character.service'

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
    const config = this.sceneData.spritesheetConfig

    this.load.tilemapTiledJSON('map', `/assets/${bootstrap.tilemap_asset_key}`)

    for (const tilesetName of bootstrap.tileset_asset_key.split(',')) {
      this.load.image(tilesetName, `/assets/tiles/${tilesetName}.png`)
    }

    this.load.spritesheet(
      this.sceneData.textureKey,
      getCharacterSpriteUrl(this.sceneData.baseAssetKey),
      { frameWidth: config.frame_width, frameHeight: config.frame_height },
    )
  }

  create() {
    this.scene.start(gameSceneKey, this.sceneData)
  }
}
